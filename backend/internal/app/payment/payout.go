package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	domainorg "marketplace-backend/internal/domain/organization"
	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// PayoutService owns every state transition that moves money out of
// platform escrow: per-milestone transfers, proposal-wide transfers,
// dispute partial transfers, refunds, manual payouts, retries, and the
// post-Premium fee waiver on already-funded records.
//
// SRP rationale: every method here mutates the transfer-side of a
// payment record (or the org's auto-payout consent flag). PI lifecycle
// stays on ChargeService; reads stay on WalletService.
//
// Dependencies:
//   - records: every read + write on payment_records
//   - orgs:    Stripe account / KYC / consent reads + writes
//   - stripe:  CreateTransfer / GetAccount / CreateRefund / CreatePayout
//   - referralDistributor: optional fire-and-forget hook on per-milestone
//     transfer success (drives the apporteur commission split)
//   - proposalStatuses: optional gate on RequestPayout / RetryFailedTransfer
//     so escrow funds never leave the platform before the mission is
//     marked completed (prevents the wallet "Retirer" side-channel bug)
type PayoutService struct {
	records repository.PaymentRecordRepository
	orgs    repository.OrganizationRepository
	stripe  portservice.StripeService

	// referralDistributor is the apporteur commission hook fired after
	// a successful per-milestone transfer. Nil when the referral feature
	// is not active — every guard in the body checks for nil before
	// invoking.
	referralDistributor portservice.ReferralCommissionDistributor

	// proposalStatuses gates payout transfers on mission completion.
	// Wired post-construction because payment is built before proposal
	// in main.go (proposal depends on payment's PaymentProcessor). When
	// nil, RequestPayout logs a warning and falls back to the legacy
	// behaviour so the payment feature stays bootable without proposal.
	proposalStatuses portservice.ProposalStatusReader
}

// PayoutServiceDeps groups every dependency NewPayoutService needs.
type PayoutServiceDeps struct {
	Records       repository.PaymentRecordRepository
	Organizations repository.OrganizationRepository
	Stripe        portservice.StripeService
}

// NewPayoutService wires the payout / transfer sub-service. Optional
// dependencies (referral distributor, proposal status reader) are wired
// post-construction via setters.
func NewPayoutService(deps PayoutServiceDeps) *PayoutService {
	return &PayoutService{
		records: deps.Records,
		orgs:    deps.Organizations,
		stripe:  deps.Stripe,
	}
}

// SetReferralDistributor plugs the referral commission distributor in
// post-construction. Safe to call at app startup after both services
// exist. Passing nil disables the hook.
func (p *PayoutService) SetReferralDistributor(d portservice.ReferralCommissionDistributor) {
	p.referralDistributor = d
}

// SetProposalStatusReader plugs the proposal status lookup used by
// RequestPayout to keep escrow funds from being transferred before the
// mission is marked completed. Setter pattern because the proposal
// service is constructed AFTER payment in main.go (proposal depends on
// payment's PaymentProcessor). Passing nil leaves RequestPayout in a
// degraded mode that logs a warning and falls back to the pre-fix
// behaviour rather than erroring out — the feature must keep working
// in unusual wirings (tests, migrations, one-off binaries).
func (p *PayoutService) SetProposalStatusReader(r portservice.ProposalStatusReader) {
	p.proposalStatuses = r
}

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

// RequestPayout triggers manual transfers for all pending payments
// belonging to the caller's organization.
//
// Only records whose proposal has reached "completed" are transferred —
// the wallet handler's AvailableAmount already gates the UI on the same
// rule, this enforces it server-side so clicking Retirer cannot pull
// funds still in escrow for an active / disputed mission.
func (p *PayoutService) RequestPayout(ctx context.Context, userID, orgID uuid.UUID) (*PayoutResult, error) {
	_ = userID // audit hook
	stripeAccountID, _, err := p.orgs.GetStripeAccount(ctx, orgID)
	if err != nil || stripeAccountID == "" {
		if errors.Is(err, sql.ErrNoRows) || stripeAccountID == "" {
			return nil, domain.ErrStripeAccountNotFound
		}
		return nil, fmt.Errorf("lookup stripe account: %w", err)
	}

	records, err := p.records.ListByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}

	// Log the degraded-mode warning once per call (not per record) so
	// production logs a single line per payout instead of a flood.
	statusesWired := p.proposalStatuses != nil
	if !statusesWired {
		slog.Warn("payment: RequestPayout falling back to legacy behaviour — ProposalStatusReader not wired; escrow funds may be released before mission completion",
			"org_id", orgID)
	}

	var transferred int64
	for _, r := range records {
		if r.Status != domain.RecordStatusSucceeded || r.TransferStatus != domain.TransferPending {
			continue
		}

		// Gate on mission status. Skip anything that isn't completed.
		// Empty string (proposal missing) is treated as "not completed"
		// — defensive, never over-pay on a stale / orphan record.
		if statusesWired {
			status, sErr := p.proposalStatuses.GetProposalStatus(ctx, r.ProposalID)
			if sErr != nil {
				slog.Error("payout: proposal status lookup failed, skipping",
					"proposal_id", r.ProposalID, "error", sErr)
				continue
			}
			if status != "completed" {
				continue
			}
		}

		transferID, tErr := p.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
			Amount:             r.ProviderPayout,
			Currency:           r.Currency,
			DestinationAccount: stripeAccountID,
			TransferGroup:      r.ProposalID.String(),
			// Idempotency scoped to the record id so multi-milestone
			// proposals don't collide on the same key.
			IdempotencyKey: fmt.Sprintf("transfer_%s_%s", r.ID, stripeAccountID),
		})
		if tErr != nil {
			slog.Error("payout transfer failed", "proposal_id", r.ProposalID, "error", tErr)
			r.MarkTransferFailed()
			_ = p.records.Update(ctx, r)
			continue
		}

		if err := r.MarkTransferred(transferID); err != nil {
			slog.Error("mark transferred", "error", err)
			continue
		}
		_ = p.records.Update(ctx, r)
		transferred += r.ProviderPayout
	}

	if transferred == 0 {
		return &PayoutResult{Status: "nothing_to_transfer", Message: "No funds available for transfer"}, nil
	}

	return p.fireBankPayout(ctx, orgID, stripeAccountID, transferred, records)
}

// fireBankPayout extracts the post-transfer bank-leg payout into a
// helper to keep RequestPayout under the 50-line function budget. It
// runs the Stripe payout, records auto-payout consent, and returns the
// final PayoutResult.
//
// Now that escrow funds have moved platform → connected account, fire
// an explicit Stripe payout so the bank transfer happens at the moment
// the user clicks Retirer. This is required because every connected
// account is created with payout_schedule.interval = "manual" (see
// adapter/stripe/account.go) — without this call, funds would sit on
// the connected account's Stripe balance and never reach the user's
// bank account.
//
// The currency comes from a record we just transferred — every payment
// record on a single connected account shares the same currency, so
// picking one is correct. If a future change allows mixed currencies on
// a single org, this should be grouped by currency before issuing
// payouts.
func (p *PayoutService) fireBankPayout(ctx context.Context, orgID uuid.UUID, stripeAccountID string, transferred int64, records []*domain.PaymentRecord) (*PayoutResult, error) {
	currency := pickPayoutCurrency(records)

	// Idempotency key: deterministic per-org-per-amount slice so a
	// duplicate Retirer click cannot double-debit the connected
	// account balance. Stripe returns the same payout id on replay.
	idemKey := fmt.Sprintf("payout_%s_%d", orgID, transferred)
	payoutID, pErr := p.stripe.CreatePayout(ctx, portservice.CreatePayoutInput{
		ConnectedAccountID: stripeAccountID,
		Amount:             transferred,
		Currency:           currency,
		IdempotencyKey:     idemKey,
		Description:        "Wallet payout",
	})
	if pErr != nil {
		// The transfers already succeeded — funds are safely on the
		// connected account. Surface the bank-leg failure but don't
		// roll back: a later RequestPayout (or the user re-clicking)
		// can retry, or Stripe Dashboard can issue a manual payout.
		// ERROR level keeps prod paged.
		slog.Error("payout: bank transfer failed after platform→connected transfer succeeded",
			"org_id", orgID,
			"amount", transferred,
			"connected_account", stripeAccountID,
			"error", pErr)
		return &PayoutResult{
			Status:  "transferred_pending_bank",
			Message: fmt.Sprintf("Transferred %d centimes — bank transfer pending", transferred),
		}, nil
	}
	slog.Info("payout: bank transfer initiated",
		"org_id", orgID, "payout_id", payoutID, "amount", transferred)

	p.recordAutoPayoutConsent(ctx, orgID)

	return &PayoutResult{
		Status:  "transferred",
		Message: fmt.Sprintf("Transferred %d centimes to your account", transferred),
	}, nil
}

// pickPayoutCurrency returns the currency of the first transferred
// record, falling back to "eur" when none is set. Extracted from
// RequestPayout so the loop body stays readable and so the rule is
// testable in isolation.
func pickPayoutCurrency(records []*domain.PaymentRecord) string {
	for _, r := range records {
		if r.Status == domain.RecordStatusSucceeded && r.TransferStatus == domain.TransferCompleted && r.Currency != "" {
			return r.Currency
		}
	}
	return "eur"
}

// recordAutoPayoutConsent stamps the AutoPayoutEnabledAt column on the
// org IF the org has not already consented. Idempotent. Failures are
// logged but never break the (already successful) payout.
//
// First-payout consent: subsequent milestone releases auto-transfer
// without waiting on another explicit click. Idempotent on the domain
// side — only the first call stamps the timestamp.
func (p *PayoutService) recordAutoPayoutConsent(ctx context.Context, orgID uuid.UUID) {
	org, err := p.orgs.FindByID(ctx, orgID)
	if err != nil || org == nil {
		return
	}
	if org.HasAutoPayoutConsent() {
		return
	}
	org.MarkAutoPayoutEnabled(time.Now())
	if uErr := p.orgs.Update(ctx, org); uErr != nil {
		slog.Warn("payout: failed to record auto-payout consent",
			"org_id", orgID, "error", uErr)
		return
	}
	slog.Info("payout: auto-payout consent recorded — future milestones will release without manual click",
		"org_id", orgID)
}

// RetryFailedTransfer re-issues the Stripe transfer for a single
// payment record stuck in TransferFailed. This is the recovery path
// for records where a previous RequestPayout hit a Stripe error
// (destination account state, network blip, etc.) — without it, the
// provider is stuck with no way to recover the funds.
//
// Takes the payment record id (NOT the proposal id) because a proposal
// can own N records — one per milestone — so only the stable record id
// uniquely identifies which failed transfer to retry. The UI passes
// record.id from the wallet DTO.
//
// Preconditions (else ErrTransferNotRetriable):
//   - the record exists and belongs to an org the user is a member of;
//   - record.Status == RecordStatusSucceeded (funds are held in escrow);
//   - record.TransferStatus == TransferFailed;
//   - the proposal is in "completed" status (same rule as RequestPayout —
//     no side-channel around escrow).
//
// Also requires the provider to have a Stripe connected account
// (ErrStripeAccountNotFound). The idempotency key is derived from the
// record id (unique per milestone), so if the original transfer
// actually succeeded silently (Stripe accepted but we marked failed on
// a timeout) Stripe returns the existing transfer ID and the record
// resolves cleanly.
func (p *PayoutService) RetryFailedTransfer(ctx context.Context, userID, orgID, recordID uuid.UUID) (*PayoutResult, error) {
	record, providerOrg, err := p.loadRetryRecord(ctx, orgID, recordID)
	if err != nil {
		return nil, err
	}
	_ = userID // audit hook — user is already authenticated via middleware

	if err := p.assertRetryAllowed(ctx, record); err != nil {
		return nil, err
	}

	stripeAccountID, err := p.resolveProviderAccount(ctx, record, orgID)
	if err != nil {
		return nil, err
	}

	if err := p.assertProviderPayoutsEnabled(ctx, stripeAccountID); err != nil {
		return nil, err
	}

	// Reset status to Pending BEFORE the Stripe call so concurrent
	// reads of the wallet can't see the row stuck as failed. If the
	// retry itself fails, we re-mark Failed below.
	record.TransferStatus = domain.TransferPending
	record.UpdatedAt = time.Now()
	if uErr := p.records.Update(ctx, record); uErr != nil {
		return nil, fmt.Errorf("reset transfer status: %w", uErr)
	}

	transferID, tErr := p.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
		Amount:             record.ProviderPayout,
		Currency:           record.Currency,
		DestinationAccount: stripeAccountID,
		TransferGroup:      record.ProposalID.String(),
		IdempotencyKey:     fmt.Sprintf("transfer_%s_%s", record.ID, stripeAccountID),
	})
	if tErr != nil {
		slog.Error("retry transfer failed", "record_id", record.ID, "proposal_id", record.ProposalID, "error", tErr)
		record.MarkTransferFailed()
		_ = p.records.Update(ctx, record)
		return nil, fmt.Errorf("retry stripe transfer: %w", tErr)
	}

	if mErr := record.MarkTransferred(transferID); mErr != nil {
		return nil, fmt.Errorf("mark transferred: %w", mErr)
	}
	if uErr := p.records.Update(ctx, record); uErr != nil {
		return nil, fmt.Errorf("persist retried transfer: %w", uErr)
	}

	// Treat a successful retry as the same "first payout" consent as
	// RequestPayout — the user explicitly clicked to release the funds
	// and Stripe accepted, so subsequent milestone releases on this
	// org can auto-transfer.
	p.maybeStampRetryConsent(ctx, providerOrg)

	return &PayoutResult{
		Status:  "transferred",
		Message: fmt.Sprintf("Transferred %d centimes to your account", record.ProviderPayout),
	}, nil
}

// loadRetryRecord fetches the record + verifies the caller's org owns
// the provider side. Extracted from RetryFailedTransfer so the orchestrator
// stays under 50 lines.
func (p *PayoutService) loadRetryRecord(ctx context.Context, orgID, recordID uuid.UUID) (*domain.PaymentRecord, *domainorg.Organization, error) {
	record, err := p.records.GetByID(ctx, recordID)
	if err != nil {
		// Surface "not found" with a typed sentinel so the handler can
		// return 404 (vs. swallowing into 500). The repository returns
		// sql.ErrNoRows when no row matches; everything else is a real
		// infra error and stays wrapped.
		if errors.Is(err, sql.ErrNoRows) || errors.Is(err, domain.ErrPaymentRecordNotFound) {
			return nil, nil, domain.ErrPaymentRecordNotFound
		}
		return nil, nil, fmt.Errorf("find payment record: %w", err)
	}
	if record == nil {
		return nil, nil, domain.ErrPaymentRecordNotFound
	}

	// Auth: the record's ProviderID must belong to the caller's org.
	// Admin-override is handled at the handler level via RequireRole.
	providerOrg, err := p.orgs.FindByUserID(ctx, record.ProviderID)
	if err != nil || providerOrg == nil {
		slog.Warn("retry: provider org lookup failed",
			"payment_record_id", record.ID,
			"provider_user_id", record.ProviderID,
			"requesting_org_id", orgID,
			"error", err)
		return nil, nil, domain.ErrTransferNotRetriable
	}
	if providerOrg.ID != orgID {
		slog.Warn("retry: provider org mismatch",
			"payment_record_id", record.ID,
			"provider_user_id", record.ProviderID,
			"resolved_provider_org", providerOrg.ID,
			"requesting_org_id", orgID)
		return nil, nil, domain.ErrTransferNotRetriable
	}
	return record, providerOrg, nil
}

// assertRetryAllowed checks the record state + proposal status gate.
func (p *PayoutService) assertRetryAllowed(ctx context.Context, record *domain.PaymentRecord) error {
	if record.Status != domain.RecordStatusSucceeded || record.TransferStatus != domain.TransferFailed {
		slog.Warn("retry: record state invalid for retry",
			"payment_record_id", record.ID,
			"record_status", record.Status,
			"transfer_status", record.TransferStatus)
		return domain.ErrTransferNotRetriable
	}

	// Same gate as RequestPayout — never open a side-channel that
	// releases escrow funds for an active / disputed mission.
	if p.proposalStatuses != nil {
		status, sErr := p.proposalStatuses.GetProposalStatus(ctx, record.ProposalID)
		if sErr != nil {
			return fmt.Errorf("lookup proposal status: %w", sErr)
		}
		if status != "completed" {
			slog.Warn("retry: proposal not completed",
				"payment_record_id", record.ID,
				"proposal_id", record.ProposalID,
				"proposal_status", status)
			return domain.ErrTransferNotRetriable
		}
	}
	return nil
}

// resolveProviderAccount returns the Stripe account id for the record's
// provider, or ErrStripeAccountNotFound when missing.
func (p *PayoutService) resolveProviderAccount(ctx context.Context, record *domain.PaymentRecord, orgID uuid.UUID) (string, error) {
	stripeAccountID, _, accErr := p.orgs.GetStripeAccountByUserID(ctx, record.ProviderID)
	if accErr != nil || stripeAccountID == "" {
		// Same diagnostic as TransferMilestone — when a retry surfaces
		// "no Stripe account" we need the full resolution chain in the
		// logs to debug the gap between this lookup and the wallet UI's
		// /payment-info/account-status which resolves via the JWT org id.
		slog.Warn("retry: provider stripe account resolution returned empty",
			"payment_record_id", record.ID,
			"proposal_id", record.ProposalID,
			"milestone_id", record.MilestoneID,
			"provider_user_id", record.ProviderID,
			"requesting_org_id", orgID,
			"resolved_stripe_account_id", stripeAccountID,
			"resolution_error", accErr,
		)
		return "", domain.ErrStripeAccountNotFound
	}
	return stripeAccountID, nil
}

// assertProviderPayoutsEnabled is the KYC readiness pre-check — distinct
// from "no account at all". Many real failures (e.g. transferring to a
// freshly-onboarded provider) happen when the account row exists but
// Stripe still has payouts_enabled=false because identity docs are
// pending or the capability is throttled. Calling CreateTransfer in
// that state burns the idempotency key and bounces with a generic
// Stripe error — the retry button silently no-ops. We block here with
// a typed sentinel so the handler can return 412 + a clear message
// pointing the user at /payment-info instead.
func (p *PayoutService) assertProviderPayoutsEnabled(ctx context.Context, stripeAccountID string) error {
	if p.stripe == nil {
		return nil
	}
	info, infoErr := p.stripe.GetAccount(ctx, stripeAccountID)
	if infoErr != nil {
		return fmt.Errorf("get stripe account capabilities: %w", infoErr)
	}
	if info == nil || !info.PayoutsEnabled {
		return domain.ErrProviderPayoutsDisabled
	}
	return nil
}

// maybeStampRetryConsent stamps auto-payout consent on the provider org
// after a successful retry. Idempotent.
func (p *PayoutService) maybeStampRetryConsent(ctx context.Context, providerOrg *domainorg.Organization) {
	if providerOrg == nil || providerOrg.HasAutoPayoutConsent() {
		return
	}
	providerOrg.MarkAutoPayoutEnabled(time.Now())
	if uErr := p.orgs.Update(ctx, providerOrg); uErr != nil {
		slog.Warn("retry: failed to record auto-payout consent",
			"org_id", providerOrg.ID, "error", uErr)
		return
	}
	slog.Info("retry: auto-payout consent recorded — future milestones will release without manual click",
		"org_id", providerOrg.ID)
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
