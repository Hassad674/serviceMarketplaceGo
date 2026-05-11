package referral

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/service"
)

// nowUTC is the small wall-clock helper used by the retry orchestrator.
// Kept as a package-level var (not a function) so unit tests that need
// determinism can override it with `monkey.Patch(nowUTC, …)` style
// fakes if the determinism story ever tightens. The current tests use
// the real clock since they only assert relative ordering, not exact
// timestamps.
var nowUTC = func() time.Time { return time.Now().UTC() }

// ErrCommissionNotOwned is returned when the requesting user is not
// the apporteur on the parent referral of the target commission. The
// handler maps this to 403.
var ErrCommissionNotOwned = errors.New("commission not owned by requesting user")

// RetryCommission implements service.ReferralCommissionRetryService.
//
// Called by the wallet handler when an apporteur clicks "Retirer" on
// a commission row stuck in pending_kyc or failed. The orchestrator:
//
//  1. Loads the commission row by id; returns ErrCommissionNotFound
//     when missing.
//  2. Resolves the parent attribution → parent referral, asserts the
//     requesting user is the apporteur on that referral.
//  3. Branches on the current status:
//       - paid                 → AlreadyPaid (handler returns 409)
//       - cancelled/clawed_back → NotRetriable (409)
//       - pending_kyc / failed → run the Connect-ready gate; on success
//         attempt the Stripe transfer; on failure update the row and
//         return the outcome.
//
// The Stripe idempotency key reuses the existing
// "referral_commission_{id}" template so a retry that succeeds on
// Stripe but fails persisting locally can be safely re-driven without
// double-paying the apporteur.
func (s *Service) RetryCommission(ctx context.Context, requestingUserID, commissionID uuid.UUID) (service.ReferralCommissionRetryOutcome, error) {
	commission, err := s.referrals.FindCommissionByID(ctx, commissionID)
	if err != nil {
		return service.ReferralCommissionRetryOutcome{}, err
	}

	parent, owner, ownErr := s.assertCommissionOwned(ctx, commission, requestingUserID)
	if ownErr != nil {
		return service.ReferralCommissionRetryOutcome{}, ownErr
	}

	switch commission.Status {
	case referral.CommissionPaid:
		return service.ReferralCommissionRetryOutcome{
			Result: service.ReferralCommissionRetryAlreadyPaid,
		}, nil
	case referral.CommissionCancelled, referral.CommissionClawedBack:
		return service.ReferralCommissionRetryOutcome{
			Result: service.ReferralCommissionRetryNotRetriable,
		}, nil
	case referral.CommissionPending, referral.CommissionPendingKYC, referral.CommissionFailed:
		// retriable
	default:
		return service.ReferralCommissionRetryOutcome{
			Result: service.ReferralCommissionRetryNotRetriable,
		}, nil
	}

	prevStatus := commission.Status
	outcome, err := s.driveCommissionRetry(ctx, commission, parent, owner)
	// Audit log every retry attempt — even when the gate trips and no
	// Stripe call fires. The audit reviewer cares about every attempt
	// (frequency = anti-fraud signal) not just successful transfers.
	s.recordCommissionRetryAudit(ctx, requestingUserID, commission.ID, prevStatus, outcome)
	return outcome, err
}

// recordCommissionRetryAudit emits one audit row per retry attempt.
// Best-effort — never blocks the caller. Captures the prev_status →
// new_status transition + the stripe failure reason (when applicable)
// so the audit trail tells the full retire story for a commission.
func (s *Service) recordCommissionRetryAudit(
	ctx context.Context,
	requestingUserID, commissionID uuid.UUID,
	prevStatus referral.CommissionStatus,
	outcome service.ReferralCommissionRetryOutcome,
) {
	if s.audits == nil {
		return
	}
	metadata := map[string]any{
		"prev_status":    string(prevStatus),
		"retry_result":   string(outcome.Result),
		"stripe_account": outcome.StripeAccount,
	}
	if outcome.FailureReason != "" {
		metadata["stripe_error"] = outcome.FailureReason
	}
	resourceID := commissionID
	entry, err := audit.NewEntry(audit.NewEntryInput{
		UserID:       &requestingUserID,
		Action:       audit.ActionCommissionRetryAttempted,
		ResourceType: audit.ResourceTypeReferralCommission,
		ResourceID:   &resourceID,
		Metadata:     metadata,
	})
	if err != nil {
		slog.Warn("commission retry audit: NewEntry failed",
			"commission_id", commissionID, "error", err)
		return
	}
	if err := s.audits.Log(ctx, entry); err != nil {
		slog.Warn("commission retry audit: persist failed",
			"commission_id", commissionID, "error", err)
	}
}

// assertCommissionOwned resolves the parent referral for a commission
// and verifies the requesting user is the apporteur. Returns the
// parent referral, the apporteur id, or an error.
func (s *Service) assertCommissionOwned(
	ctx context.Context,
	commission *referral.Commission,
	requestingUserID uuid.UUID,
) (*referral.Referral, uuid.UUID, error) {
	att, err := s.referrals.FindAttributionByID(ctx, commission.AttributionID)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("load attribution: %w", err)
	}
	parent, err := s.referrals.GetByID(ctx, att.ReferralID)
	if err != nil {
		return nil, uuid.Nil, fmt.Errorf("load parent referral: %w", err)
	}
	if parent.ReferrerID != requestingUserID {
		return nil, uuid.Nil, ErrCommissionNotOwned
	}
	return parent, parent.ReferrerID, nil
}

// driveCommissionRetry runs the Connect-ready gate and, when ready,
// re-attempts the Stripe transfer. Side effects are limited to the
// single commission row + the notification fan-out.
func (s *Service) driveCommissionRetry(
	ctx context.Context,
	commission *referral.Commission,
	parent *referral.Referral,
	apporteurID uuid.UUID,
) (service.ReferralCommissionRetryOutcome, error) {
	stripeAccount := s.resolveStripeAccount(ctx, apporteurID)
	if !s.connectReadyForReferrer(ctx, stripeAccount) {
		// Persist pending_kyc when the row was previously failed; a row
		// already in pending_kyc stays as-is.
		if commission.Status == referral.CommissionFailed {
			commission.Status = referral.CommissionPendingKYC
			commission.FailureReason = ""
			if uerr := s.referrals.UpdateCommission(ctx, commission); uerr != nil {
				slog.Warn("commission retry: persist pending_kyc transition failed",
					"commission_id", commission.ID, "error", uerr)
			}
		}
		return service.ReferralCommissionRetryOutcome{
			Result:        service.ReferralCommissionRetryKYCRequired,
			StripeAccount: stripeAccount,
		}, nil
	}

	transferID, err := s.stripe.CreateTransfer(ctx, service.CreateTransferInput{
		Amount:             commission.CommissionCents,
		Currency:           commission.Currency,
		DestinationAccount: stripeAccount,
		TransferGroup:      fmt.Sprintf("referral_%s", parent.ID),
		IdempotencyKey:     fmt.Sprintf("referral_commission_%s", commission.ID),
	})
	if err != nil {
		// Force-set failed status — MarkFailed only allows pending /
		// pending_kyc, but a retry from a `failed` row must remain
		// failed when Stripe rejects it again, and the failure_reason
		// must be refreshed.
		commission.Status = referral.CommissionFailed
		commission.FailureReason = err.Error()
		if uerr := s.referrals.UpdateCommission(ctx, commission); uerr != nil {
			slog.Warn("commission retry: persist failed-status failed",
				"commission_id", commission.ID, "error", uerr)
		}
		slog.Error("referral: commission retry stripe transfer failed",
			"commission_id", commission.ID, "error", err)
		return service.ReferralCommissionRetryOutcome{
			Result:        service.ReferralCommissionRetryFailed,
			StripeAccount: stripeAccount,
			FailureReason: err.Error(),
		}, nil
	}

	// Stripe transfer succeeded — promote to paid regardless of
	// previous status (pending_kyc OR failed).
	commission.Status = referral.CommissionPaid
	commission.StripeTransferID = transferID
	commission.FailureReason = ""
	now := nowUTC()
	commission.PaidAt = &now
	if uerr := s.referrals.UpdateCommission(ctx, commission); uerr != nil {
		return service.ReferralCommissionRetryOutcome{}, fmt.Errorf("update commission to paid on retry: %w", uerr)
	}
	s.notifyCommissionPaid(ctx, parent.ID, apporteurID, commission.CommissionCents, transferID)
	return service.ReferralCommissionRetryOutcome{
		Result:        service.ReferralCommissionRetryPaid,
		StripeAccount: stripeAccount,
	}, nil
}

// resolveStripeAccount delegates to the stripe-account resolver
// when wired. Returns empty string on resolution failure so the
// caller can treat "no account" identically to "lookup failed".
func (s *Service) resolveStripeAccount(ctx context.Context, userID uuid.UUID) string {
	if s.stripeAccounts == nil {
		return ""
	}
	id, err := s.stripeAccounts.ResolveStripeAccountID(ctx, userID)
	if err != nil {
		slog.Warn("referral: resolve stripe account failed during retry",
			"user_id", userID, "error", err)
		return ""
	}
	return id
}
