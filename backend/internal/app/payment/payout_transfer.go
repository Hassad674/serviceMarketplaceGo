package payment

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	portservice "marketplace-backend/internal/port/service"
)

// payout_transfer.go — the transfer-side methods of PayoutService:
// every state transition that pushes money OUT of platform escrow to a
// provider's connected account or back to the client. Phase 3.1 split
// this off payout.go to keep both files under the 600-line ceiling.
// Manual-payout entry points (RequestPayout, RetryFailedTransfer) and
// their helpers live in payout_request.go.

// TransferToProvider releases EVERY pending payment record of a
// proposal to the provider's connected account. Used at macro
// completion and by the outbox worker where no specific milestone id
// is known.
//
// Iterates ListByProposalID (ordered oldest first) and delegates to
// TransferMilestone for each record that is still
// succeeded+TransferPending. Records in any other state are skipped
// silently so a repeat call after partial success is idempotent.
//
// For milestone-scoped releases (mid-project approve or auto-approve)
// callers MUST use TransferMilestone directly — calling this with a
// multi-milestone proposal where only ONE milestone is released would
// incorrectly double-transfer already-released jalons (they are
// skipped via the gate, but intent-wise the caller should be
// explicit).
func (p *PayoutService) TransferToProvider(ctx context.Context, proposalID uuid.UUID) error {
	records, err := p.records.ListByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("list payment records: %w", err)
	}
	if len(records) == 0 {
		return domain.ErrPaymentRecordNotFound
	}

	var firstErr error
	var releasedAny bool
	for _, r := range records {
		// Skip anything not held in escrow. This makes the call
		// idempotent: replaying after a partial success re-hits
		// only the records still TransferPending.
		if r.Status != domain.RecordStatusSucceeded || r.TransferStatus != domain.TransferPending {
			continue
		}
		if r.MilestoneID == uuid.Nil {
			// Defensive: the phase-4 migration forbids a zero
			// milestone_id, but if we somehow find one skip it
			// rather than crash. Log and move on.
			slog.Warn("transfer: skipping record with zero milestone_id",
				"record_id", r.ID, "proposal_id", proposalID)
			continue
		}
		if err := p.TransferMilestone(ctx, r.MilestoneID); err != nil {
			// Collect the first error but keep going so one stuck
			// milestone doesn't block the others (each milestone
			// is independent — partial success is better than none).
			if firstErr == nil {
				firstErr = err
			}
			slog.Warn("transfer: milestone release failed",
				"milestone_id", r.MilestoneID, "proposal_id", proposalID, "error", err)
			continue
		}
		releasedAny = true
	}

	if firstErr != nil {
		return firstErr
	}
	if !releasedAny {
		// Nothing to transfer — every record was already released or
		// not-yet-succeeded. Preserve the pre-refactor "transfer already
		// done" sentinel so callers that rely on it keep working.
		return domain.ErrTransferAlreadyDone
	}
	return nil
}

// TransferMilestone releases a single milestone's payment record to the
// provider's connected account. This is the primary release path for
// multi-milestone proposals — per-milestone releases (CompleteProposal
// mid-project, AutoApproveMilestone) MUST use this so the correct
// record is transferred and the referral commission hook fires against
// the just-released milestone.
//
// Returns ErrPaymentNotSucceeded if the record is not held in escrow,
// ErrTransferAlreadyDone if it was already transferred, and
// ErrStripeAccountNotFound if the provider has not completed KYC.
func (p *PayoutService) TransferMilestone(ctx context.Context, milestoneID uuid.UUID) error {
	record, err := p.records.GetByMilestoneID(ctx, milestoneID)
	if err != nil {
		return fmt.Errorf("find payment record: %w", err)
	}

	if record.Status != domain.RecordStatusSucceeded {
		return domain.ErrPaymentNotSucceeded
	}
	if record.TransferStatus != domain.TransferPending {
		return domain.ErrTransferAlreadyDone
	}

	stripeAccountID, _, err := p.orgs.GetStripeAccountByUserID(ctx, record.ProviderID)
	if err != nil || stripeAccountID == "" {
		// Diagnostic — print every input to the resolution chain so a
		// post-mortem on a "provider has no Stripe connected account"
		// failure can be done without re-running with extra debug
		// instrumentation. Critical when the wallet UI reports the
		// account as ready (resolved through the JWT org id) but the
		// transfer path resolves through the provider user id and lands
		// on a different / empty mapping.
		slog.Warn("transfer: provider stripe account resolution returned empty",
			"payment_record_id", record.ID,
			"proposal_id", record.ProposalID,
			"milestone_id", record.MilestoneID,
			"provider_user_id", record.ProviderID,
			"client_user_id", record.ClientID,
			"resolved_stripe_account_id", stripeAccountID,
			"resolution_error", err,
		)
		return domain.ErrStripeAccountNotFound
	}

	transferID, err := p.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
		Amount:             record.ProviderPayout,
		Currency:           record.Currency,
		DestinationAccount: stripeAccountID,
		TransferGroup:      record.ProposalID.String(),
		// Idempotency scoped to the record id so multi-milestone
		// proposals don't collide on the same key (proposal-only key
		// was buggy).
		IdempotencyKey: fmt.Sprintf("transfer_%s_%s", record.ID, stripeAccountID),
	})
	if err != nil {
		// BUG-09: previously this branch did `_ = p.records.Update(...)`
		// which lost the MarkTransferFailed state when the DB write
		// failed. The record then stays as Succeeded+TransferPending
		// (the wallet shows it ready to retry) but the Stripe transfer
		// permanently failed — every retry hits the same Stripe error
		// without the user seeing the failure. Surface the DB error
		// alongside the Stripe error so the caller knows the record
		// is desynced from Stripe state.
		record.MarkTransferFailed()
		if uErr := p.records.Update(ctx, record); uErr != nil {
			slog.Error("payment: failed to persist MarkTransferFailed — record desynced from Stripe",
				"record_id", record.ID,
				"proposal_id", record.ProposalID,
				"milestone_id", record.MilestoneID,
				"stripe_error", err,
				"db_error", uErr,
			)
			// Both errors matter — combine them so the caller can
			// decide whether to alert. The Stripe error is the
			// primary one (the original create-transfer failure).
			return fmt.Errorf("create stripe transfer: %w (mark failed save also failed: %v)", err, uErr)
		}
		return fmt.Errorf("create stripe transfer: %w", err)
	}

	if err := record.MarkTransferred(transferID); err != nil {
		return fmt.Errorf("mark transferred: %w", err)
	}

	if err := p.records.Update(ctx, record); err != nil {
		return err
	}

	// Referral commission split — fire-and-forget into the referral
	// feature. A non-nil distributor means the referral feature is
	// wired; nil means we're running without it, skip silently. Errors
	// here are logged and swallowed so a flaky referral service never
	// blocks the provider's primary transfer.
	if p.referralDistributor != nil && record.MilestoneID != uuid.Nil {
		_, rErr := p.referralDistributor.DistributeIfApplicable(ctx, portservice.ReferralCommissionDistributorInput{
			ProposalID:       record.ProposalID,
			MilestoneID:      record.MilestoneID,
			GrossAmountCents: record.ProposalAmount,
			Currency:         record.Currency,
		})
		if rErr != nil {
			slog.Warn("referral: commission distribution failed",
				"proposal_id", record.ProposalID,
				"milestone_id", record.MilestoneID,
				"error", rErr)
		}
	}

	return nil
}

// TransferPartialToProvider applies a dispute resolution split to the
// provider's payment record and, when possible, transfers the funds to
// their Stripe account.
//
// CRITICAL invariant: the record's ProviderPayout MUST always reflect
// the admin / amiable / scheduler decision, even if the Stripe transfer
// cannot happen right now (e.g. the provider has not yet completed KYC
// or their Stripe account is not payouts-enabled). Otherwise
// RequestPayout later transfers the original proposal amount,
// over-paying the provider and leaving the platform short by the
// client's refunded portion.
//
// Three possible outcomes:
//   - amount == 0 (full refund to client): the record is marked
//     completed with zero payout, no Stripe call.
//   - amount > 0 and provider has a Stripe account: transfer is
//     attempted. If it succeeds, the record is marked completed. If
//     Stripe rejects (e.g. payouts not enabled yet), the new
//     ProviderPayout is still persisted with TransferPending so a later
//     RequestPayout can retry with the correct amount.
//   - amount > 0 and provider has NO Stripe account yet: the record is
//     persisted with the new ProviderPayout but TransferPending. When
//     the provider finishes KYC and calls RequestPayout, they receive
//     the split amount — not the original proposal amount.
func (p *PayoutService) TransferPartialToProvider(ctx context.Context, proposalID uuid.UUID, amount int64) error {
	record, err := p.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("find payment record: %w", err)
	}
	if record.Status != domain.RecordStatusSucceeded {
		return domain.ErrPaymentNotSucceeded
	}

	// Full refund to client — nothing to transfer, mark the record as
	// resolved with zero payout so the wallet removes it from escrow.
	if amount == 0 {
		// State guard rejection (BUG-02): the record is no longer in a
		// state where ApplyDisputeResolution is valid (already
		// transferred or never succeeded). Surface the error so the
		// caller can log + skip — overwriting ProviderPayout silently
		// would lose the provider's money.
		if err := record.ApplyDisputeResolution(0, ""); err != nil {
			return fmt.Errorf("apply zero-payout dispute resolution: %w", err)
		}
		return p.records.Update(ctx, record)
	}

	// Provider has no Stripe account yet (KYC incomplete). Persist the
	// new payout but keep TransferPending so a post-KYC RequestPayout
	// completes the transfer with the correct (split) amount. This is a
	// valid state, not an error: the dispute flow continues normally.
	stripeAccountID, _, accErr := p.orgs.GetStripeAccountByUserID(ctx, record.ProviderID)
	if accErr != nil || stripeAccountID == "" {
		slog.Info("dispute: transfer deferred pending provider KYC",
			"proposal_id", proposalID,
			"old_payout", record.ProviderPayout,
			"new_payout", amount,
		)
		record.ProviderPayout = amount
		record.UpdatedAt = time.Now()
		return p.records.Update(ctx, record)
	}

	// Provider is KYC-ready — attempt the Stripe transfer.
	transferID, err := p.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
		Amount:             amount,
		Currency:           record.Currency,
		DestinationAccount: stripeAccountID,
		TransferGroup:      proposalID.String(),
		IdempotencyKey:     fmt.Sprintf("dispute_transfer_%s_%d", proposalID, amount),
	})
	if err != nil {
		// Stripe rejected (e.g. payouts not enabled yet). Persist the
		// new ProviderPayout so RequestPayout retries with the correct
		// amount, then surface the error so the dispute flow can log
		// it.
		slog.Warn("dispute: stripe transfer failed, payout deferred",
			"proposal_id", proposalID,
			"new_payout", amount,
			"error", err,
		)
		record.ProviderPayout = amount
		record.UpdatedAt = time.Now()
		if saveErr := p.records.Update(ctx, record); saveErr != nil {
			return fmt.Errorf("stripe partial transfer: %w (save failed: %v)", err, saveErr)
		}
		return fmt.Errorf("stripe partial transfer: %w", err)
	}

	// Transfer succeeded — mark completed with the new amount.
	// State guard rejection (BUG-02) is surfaced so a caller cannot
	// double-apply the resolution after a webhook replay.
	if err := record.ApplyDisputeResolution(amount, transferID); err != nil {
		return fmt.Errorf("apply dispute resolution: %w", err)
	}
	return p.records.Update(ctx, record)
}

// RefundToClient creates a partial or full refund on the original
// payment. If the provider portion is 0 (full refund), marks the record
// as refunded so the wallet excludes it from escrow.
func (p *PayoutService) RefundToClient(ctx context.Context, proposalID uuid.UUID, amount int64) error {
	if amount <= 0 {
		return nil
	}
	record, err := p.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("find payment record: %w", err)
	}
	if record.StripePaymentIntentID == "" {
		return fmt.Errorf("no payment intent for refund")
	}

	_, err = p.stripe.CreateRefund(ctx, record.StripePaymentIntentID, amount)
	if err != nil {
		return fmt.Errorf("stripe refund: %w", err)
	}

	// Full refund (provider gets nothing) → mark the entire payment as
	// refunded. State guard rejection (BUG-02): MarkRefunded only
	// accepts a Succeeded record. A replay on an already-Refunded record
	// returns an error here; we surface it so the dispute flow can
	// decide whether the duplicate is benign (idempotent retry) or a
	// true bug.
	if record.ProviderPayout == 0 {
		if err := record.MarkRefunded(); err != nil {
			return fmt.Errorf("mark record refunded: %w", err)
		}
	}
	return p.records.Update(ctx, record)
}

// CanProviderReceivePayouts reports whether the given provider
// organization has a Stripe Connect account that is ready to receive
// transfers (account id known AND payouts_enabled == true on the live
// Stripe account).
//
// Used as a pre-check by the proposal milestone-release path so we
// never flip a milestone to "released" + send a "milestone paid"
// notification when the underlying Stripe transfer would fail (no
// account, KYC pending, capability disabled, …).
//
// A nil error with a false bool means "the provider is not ready" —
// that is the expected, non-error happy path for ghost providers. A
// non-nil error means we could not even determine readiness (Stripe
// API down, org lookup failed) — the caller MUST treat this as
// not-ready and surface it the same way to avoid a partial release.
func (p *PayoutService) CanProviderReceivePayouts(ctx context.Context, providerOrgID uuid.UUID) (bool, error) {
	accountID, _, err := p.orgs.GetStripeAccount(ctx, providerOrgID)
	if err != nil {
		return false, fmt.Errorf("get stripe account: %w", err)
	}
	if strings.TrimSpace(accountID) == "" {
		return false, nil
	}
	if p.stripe == nil {
		// No Stripe wired — degrade safely. Same posture as the
		// existing wallet-overview path which silently skips the
		// GetAccount call.
		return false, nil
	}
	info, err := p.stripe.GetAccount(ctx, accountID)
	if err != nil {
		return false, fmt.Errorf("get stripe account capabilities: %w", err)
	}
	if info == nil {
		return false, nil
	}
	return info.PayoutsEnabled, nil
}

// HasAutoPayoutConsent implements service.PaymentProcessor. Reads the
// AutoPayoutEnabledAt timestamp from the organization. The first
// successful RequestPayout / RetryFailedTransfer stamps the column;
// from that point on, callers can transfer milestone funds without
// waiting on another explicit click. Returns false (with nil error)
// when the org cannot be found so a missing record fails closed.
func (p *PayoutService) HasAutoPayoutConsent(ctx context.Context, providerOrgID uuid.UUID) (bool, error) {
	org, err := p.orgs.FindByID(ctx, providerOrgID)
	if err != nil {
		return false, fmt.Errorf("find org for auto-payout consent: %w", err)
	}
	if org == nil {
		return false, nil
	}
	return org.HasAutoPayoutConsent(), nil
}

// WaivePlatformFeeOnActiveRecords zeroes the platform fee on every
// payment_record of the org that is still in flight — i.e. the money
// is held in escrow by the platform and has not yet been transferred
// to the provider's connected account. Called when the org's
// subscription becomes active so missions started before the upgrade
// stop carrying a fee from the moment the user goes Premium.
//
// In flight = transfer_status IN ('pending', 'failed'). Already-
// transferred records (transfer_status = 'completed') are NOT touched
// because the money has already been split between the platform and
// the provider — refunding the fee retroactively would require a
// second Stripe transfer that this V1 doesn't model.
//
// Errors at the per-record level are logged and the loop continues so
// a single bad record doesn't block the rest of the org's records
// from being credited. The aggregate count of waived records is
// logged at INFO so the operator can confirm in a single line that
// the hook fired correctly.
func (p *PayoutService) WaivePlatformFeeOnActiveRecords(ctx context.Context, providerOrgID uuid.UUID) error {
	if p.records == nil {
		return nil
	}
	records, err := p.records.ListByOrganization(ctx, providerOrgID)
	if err != nil {
		return fmt.Errorf("list records: %w", err)
	}
	var waived int
	for _, r := range records {
		if r.TransferStatus != domain.TransferPending && r.TransferStatus != domain.TransferFailed {
			continue
		}
		if r.PlatformFeeAmount == 0 {
			// Already at zero (concurrent waiver, manual edit, etc.) — skip.
			continue
		}
		r.PlatformFeeAmount = 0
		r.ProviderPayout = r.ProposalAmount
		r.UpdatedAt = time.Now()
		if uErr := p.records.Update(ctx, r); uErr != nil {
			slog.Warn("waiver: failed to update record",
				"record_id", r.ID, "org_id", providerOrgID, "error", uErr)
			continue
		}
		waived++
	}
	slog.Info("waiver: platform fee waived on active records",
		"org_id", providerOrgID, "waived_count", waived, "total_records", len(records))
	return nil
}
