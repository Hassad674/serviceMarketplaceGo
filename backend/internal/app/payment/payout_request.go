package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	domainorg "marketplace-backend/internal/domain/organization"
	domain "marketplace-backend/internal/domain/payment"
	portservice "marketplace-backend/internal/port/service"
)

// payout_request.go — manual-payout entry points + retry path. Phase
// 3.1 split this off payout.go to keep both files under the 600-line
// ceiling. The "transfer side" methods that release funds straight to
// providers (TransferToProvider, TransferMilestone, RefundToClient,
// dispute splits, fee waivers, KYC pre-check) live in
// payout_transfer.go.
//
// What lives here:
//   - RequestPayout — the "Retirer" wallet button: drains every
//     completed-mission record of the org to the connected account, then
//     fires a bank-leg payout.
//   - RetryFailedTransfer — recovery path for individual records stuck
//     in TransferFailed.
//   - shared helpers: fireBankPayout, pickPayoutCurrency,
//     recordAutoPayoutConsent, plus the four small extracted helpers
//     used by RetryFailedTransfer to keep the orchestrator under 50 lines
//     (loadRetryRecord, assertRetryAllowed, resolveProviderAccount,
//     assertProviderPayoutsEnabled, maybeStampRetryConsent).

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
			// BUG-NEW-01: previously `_ = p.records.Update(ctx, r)`
			// silently swallowed DB errors, leaving the record stuck
			// as Succeeded+TransferPending after a Stripe failure.
			// The wallet then showed it ready to retry but the Stripe
			// transfer was permanently failed. Surface the desync so
			// ops can see the gap in a structured log line.
			if uErr := p.records.Update(ctx, r); uErr != nil {
				slog.Error("payout: failed to persist MarkTransferFailed — record desynced from Stripe",
					"record_id", r.ID,
					"proposal_id", r.ProposalID,
					"milestone_id", r.MilestoneID,
					"stripe_error", tErr,
					"db_error", uErr,
				)
			}
			continue
		}

		if err := r.MarkTransferred(transferID); err != nil {
			slog.Error("mark transferred", "error", err)
			continue
		}
		// BUG-NEW-01: previously `_ = p.records.Update(ctx, r)` — a
		// DB failure here lost the MarkTransferred state (transferID +
		// TransferCompleted). The wallet then re-listed the record as
		// pending and a future RequestPayout would double-transfer
		// (Stripe idempotency-key dedupes, but we'd still record a
		// second success on the same row). Surface the desync.
		if uErr := p.records.Update(ctx, r); uErr != nil {
			slog.Error("payout: failed to persist MarkTransferred — record desynced from Stripe",
				"record_id", r.ID,
				"proposal_id", r.ProposalID,
				"milestone_id", r.MilestoneID,
				"transfer_id", transferID,
				"db_error", uErr,
			)
			continue
		}
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
		// BUG-NEW-01: previously `_ = p.records.Update(ctx, record)`
		// silently swallowed DB failures. The record was reset to
		// Pending earlier in this method (line ~631). If Stripe fails
		// AND the Update to mark it Failed fails, the record stays
		// stuck at Pending — meaning subsequent reads can't tell
		// whether the retry actually completed. Surface the desync so
		// ops can see both errors and reconcile manually.
		if uErr := p.records.Update(ctx, record); uErr != nil {
			slog.Error("retry: failed to persist MarkTransferFailed — record desynced from Stripe",
				"record_id", record.ID,
				"proposal_id", record.ProposalID,
				"milestone_id", record.MilestoneID,
				"stripe_error", tErr,
				"db_error", uErr,
			)
			return nil, fmt.Errorf("retry stripe transfer: %w (mark failed save also failed: %v)", tErr, uErr)
		}
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
//
// The read goes through GetByIDForOrg so RLS denies a payment
// record owned by another tenant — the existing application-level
// `providerOrg.ID != orgID` check below stays as the redundant
// defense-in-depth gate.
func (p *PayoutService) loadRetryRecord(ctx context.Context, orgID, recordID uuid.UUID) (*domain.PaymentRecord, *domainorg.Organization, error) {
	record, err := p.records.GetByIDForOrg(ctx, recordID, orgID)
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
