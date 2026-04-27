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

	"marketplace-backend/internal/domain/billing"
	domain "marketplace-backend/internal/domain/payment"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	domainuser "marketplace-backend/internal/domain/user"
	portservice "marketplace-backend/internal/port/service"
)

// FeePreviewResult bundles the pure fee calculation with two flags the
// UI acts on.
//
// ViewerIsProvider answers "would the authenticated user pay this fee on
// a proposal against the given recipient?" — the UI hides the preview
// entirely when this is false so a client never sees the prestataire's
// cost structure.
//
// ViewerIsSubscribed answers "does the user currently have Premium?" —
// when true, Billing.FeeCents is already zeroed by the service so the
// caller can render the summary as-is. The flag lets the UI show a
// Premium badge / highlight differently without recomputing the grid.
type FeePreviewResult struct {
	Billing            billing.Result
	ViewerIsProvider   bool
	ViewerIsSubscribed bool
}

// CreatePaymentIntent creates a Stripe PaymentIntent for a milestone
// payment. Phase 4: the idempotency key is the milestone id, not the
// proposal id, so a proposal with N milestones can be funded N times
// (one PaymentIntent per milestone) without the second call reusing
// the first milestone's intent.
func (s *Service) CreatePaymentIntent(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	if s.stripe == nil {
		return nil, errors.New("stripe not configured")
	}

	existing, err := s.records.GetByMilestoneID(ctx, input.MilestoneID)
	if err == nil && existing != nil {
		return s.createPaymentIntentFromExisting(ctx, input)
	}

	stripeFee := domain.EstimateStripeFee(input.ProposalAmount)

	// Platform fee is computed from the billing schedule using the provider's
	// role (agency pays the agency grid, everyone else pays the freelance
	// grid). The fee is frozen into the payment_record row at creation time —
	// future schedule changes never retro-modify historical records.
	platformFee, err := s.computePlatformFee(ctx, input.ProviderID, input.ProposalAmount)
	if err != nil {
		return nil, fmt.Errorf("compute platform fee: %w", err)
	}

	record := domain.NewPaymentRecord(
		input.ProposalID, input.MilestoneID, input.ClientID, input.ProviderID,
		input.ProposalAmount, stripeFee, platformFee,
	)

	pi, err := s.stripe.CreatePaymentIntent(ctx, portservice.CreatePaymentIntentInput{
		AmountCentimes: record.ClientTotalAmount,
		Currency:       "eur",
		ProposalID:     input.ProposalID.String(),
		MilestoneID:    input.MilestoneID.String(),
		ClientID:       input.ClientID.String(),
		ProviderID:     input.ProviderID.String(),
		TransferGroup:  input.ProposalID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("create payment intent: %w", err)
	}
	record.StripePaymentIntentID = pi.PaymentIntentID

	if err := s.records.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("persist payment record: %w", err)
	}

	return &portservice.PaymentIntentOutput{
		ClientSecret:    pi.ClientSecret,
		PaymentRecordID: record.ID,
		ProposalAmount:  record.ProposalAmount,
		StripeFee:       record.StripeFeeAmount,
		PlatformFee:     record.PlatformFeeAmount,
		ClientTotal:     record.ClientTotalAmount,
		ProviderPayout:  record.ProviderPayout,
	}, nil
}

// createPaymentIntentFromExisting re-creates a PaymentIntent for an
// existing milestone record (idempotent via Stripe's idempotency key).
func (s *Service) createPaymentIntentFromExisting(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	existing, err := s.records.GetByMilestoneID(ctx, input.MilestoneID)
	if err != nil {
		return nil, fmt.Errorf("fetch existing record: %w", err)
	}

	pi, err := s.stripe.CreatePaymentIntent(ctx, portservice.CreatePaymentIntentInput{
		AmountCentimes: existing.ClientTotalAmount,
		Currency:       existing.Currency,
		ProposalID:     input.ProposalID.String(),
		MilestoneID:    input.MilestoneID.String(),
		ClientID:       input.ClientID.String(),
		ProviderID:     input.ProviderID.String(),
		TransferGroup:  input.ProposalID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("retrieve existing payment intent: %w", err)
	}

	if existing.StripePaymentIntentID == "" {
		existing.StripePaymentIntentID = pi.PaymentIntentID
		_ = s.records.Update(ctx, existing)
	}

	return &portservice.PaymentIntentOutput{
		ClientSecret:    pi.ClientSecret,
		PaymentRecordID: existing.ID,
		ProposalAmount:  existing.ProposalAmount,
		StripeFee:       existing.StripeFeeAmount,
		PlatformFee:     existing.PlatformFeeAmount,
		ClientTotal:     existing.ClientTotalAmount,
		ProviderPayout:  existing.ProviderPayout,
	}, nil
}

// MarkPaymentSucceeded marks a payment record as paid.
func (s *Service) MarkPaymentSucceeded(ctx context.Context, proposalID uuid.UUID) error {
	record, err := s.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("find record: %w", err)
	}
	if err := record.MarkPaid(); err != nil {
		// Idempotent — if already in non-pending state, treat as success.
		if errors.Is(err, domain.ErrPaymentNotPending) {
			return nil
		}
		return err
	}
	return s.records.Update(ctx, record)
}

// HandlePaymentSucceeded handles the payment_intent.succeeded webhook event.
// Returns the proposal_id so the proposal service can activate the mission.
func (s *Service) HandlePaymentSucceeded(ctx context.Context, paymentIntentID string) (uuid.UUID, error) {
	record, err := s.records.GetByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("find record: %w", err)
	}
	if err := record.MarkPaid(); err != nil {
		if errors.Is(err, domain.ErrPaymentNotPending) {
			return record.ProposalID, nil // idempotent
		}
		return uuid.Nil, err
	}
	if err := s.records.Update(ctx, record); err != nil {
		return uuid.Nil, fmt.Errorf("update record: %w", err)
	}
	return record.ProposalID, nil
}

// TransferToProvider releases EVERY pending payment record of a proposal
// to the provider's connected account. Used at macro completion and by
// the outbox worker where no specific milestone id is known.
//
// Iterates ListByProposalID (ordered oldest first) and delegates to
// TransferMilestone for each record that is still
// succeeded+TransferPending. Records in any other state are skipped
// silently so a repeat call after partial success is idempotent.
//
// For milestone-scoped releases (mid-project approve or auto-approve)
// callers MUST use TransferMilestone directly — calling this with a
// multi-milestone proposal where only ONE milestone is released would
// incorrectly double-transfer already-released jalons (they are skipped
// via the gate, but intent-wise the caller should be explicit).
func (s *Service) TransferToProvider(ctx context.Context, proposalID uuid.UUID) error {
	records, err := s.records.ListByProposalID(ctx, proposalID)
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
		if err := s.TransferMilestone(ctx, r.MilestoneID); err != nil {
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
func (s *Service) TransferMilestone(ctx context.Context, milestoneID uuid.UUID) error {
	record, err := s.records.GetByMilestoneID(ctx, milestoneID)
	if err != nil {
		return fmt.Errorf("find payment record: %w", err)
	}

	if record.Status != domain.RecordStatusSucceeded {
		return domain.ErrPaymentNotSucceeded
	}
	if record.TransferStatus != domain.TransferPending {
		return domain.ErrTransferAlreadyDone
	}

	stripeAccountID, _, err := s.orgs.GetStripeAccountByUserID(ctx, record.ProviderID)
	if err != nil || stripeAccountID == "" {
		return domain.ErrStripeAccountNotFound
	}

	transferID, err := s.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
		Amount:             record.ProviderPayout,
		Currency:           record.Currency,
		DestinationAccount: stripeAccountID,
		TransferGroup:      record.ProposalID.String(),
		// Idempotency scoped to the record id so multi-milestone proposals
		// don't collide on the same key (proposal-only key was buggy).
		IdempotencyKey: fmt.Sprintf("transfer_%s_%s", record.ID, stripeAccountID),
	})
	if err != nil {
		record.MarkTransferFailed()
		_ = s.records.Update(ctx, record)
		return fmt.Errorf("create stripe transfer: %w", err)
	}

	if err := record.MarkTransferred(transferID); err != nil {
		return fmt.Errorf("mark transferred: %w", err)
	}

	if err := s.records.Update(ctx, record); err != nil {
		return err
	}

	// Referral commission split — fire-and-forget into the referral feature.
	// A non-nil distributor means the referral feature is wired; nil means
	// we're running without it, skip silently. Errors here are logged and
	// swallowed so a flaky referral service never blocks the provider's
	// primary transfer.
	if s.referralDistributor != nil && record.MilestoneID != uuid.Nil {
		_, rErr := s.referralDistributor.DistributeIfApplicable(ctx, portservice.ReferralCommissionDistributorInput{
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
// CRITICAL invariant: the record's ProviderPayout MUST always reflect the
// admin / amiable / scheduler decision, even if the Stripe transfer cannot
// happen right now (e.g. the provider has not yet completed KYC or their
// Stripe account is not payouts-enabled). Otherwise RequestPayout later
// transfers the original proposal amount, over-paying the provider and
// leaving the platform short by the client's refunded portion.
//
// Three possible outcomes:
//   - amount == 0 (full refund to client): the record is marked completed
//     with zero payout, no Stripe call.
//   - amount > 0 and provider has a Stripe account: transfer is attempted.
//     If it succeeds, the record is marked completed. If Stripe rejects
//     (e.g. payouts not enabled yet), the new ProviderPayout is still
//     persisted with TransferPending so a later RequestPayout can retry
//     with the correct amount.
//   - amount > 0 and provider has NO Stripe account yet: the record is
//     persisted with the new ProviderPayout but TransferPending. When the
//     provider finishes KYC and calls RequestPayout, they receive the
//     split amount — not the original proposal amount.
func (s *Service) TransferPartialToProvider(ctx context.Context, proposalID uuid.UUID, amount int64) error {
	record, err := s.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("find payment record: %w", err)
	}
	if record.Status != domain.RecordStatusSucceeded {
		return domain.ErrPaymentNotSucceeded
	}

	// Full refund to client — nothing to transfer, mark the record as
	// resolved with zero payout so the wallet removes it from escrow.
	if amount == 0 {
		record.ApplyDisputeResolution(0, "")
		return s.records.Update(ctx, record)
	}

	// Provider has no Stripe account yet (KYC incomplete). Persist the new
	// payout but keep TransferPending so a post-KYC RequestPayout completes
	// the transfer with the correct (split) amount. This is a valid state,
	// not an error: the dispute flow continues normally.
	stripeAccountID, _, accErr := s.orgs.GetStripeAccountByUserID(ctx, record.ProviderID)
	if accErr != nil || stripeAccountID == "" {
		slog.Info("dispute: transfer deferred pending provider KYC",
			"proposal_id", proposalID,
			"old_payout", record.ProviderPayout,
			"new_payout", amount,
		)
		record.ProviderPayout = amount
		record.UpdatedAt = time.Now()
		return s.records.Update(ctx, record)
	}

	// Provider is KYC-ready — attempt the Stripe transfer.
	transferID, err := s.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
		Amount:             amount,
		Currency:           record.Currency,
		DestinationAccount: stripeAccountID,
		TransferGroup:      proposalID.String(),
		IdempotencyKey:     fmt.Sprintf("dispute_transfer_%s_%d", proposalID, amount),
	})
	if err != nil {
		// Stripe rejected (e.g. payouts not enabled yet). Persist the new
		// ProviderPayout so RequestPayout retries with the correct amount,
		// then surface the error so the dispute flow can log it.
		slog.Warn("dispute: stripe transfer failed, payout deferred",
			"proposal_id", proposalID,
			"new_payout", amount,
			"error", err,
		)
		record.ProviderPayout = amount
		record.UpdatedAt = time.Now()
		if saveErr := s.records.Update(ctx, record); saveErr != nil {
			return fmt.Errorf("stripe partial transfer: %w (save failed: %v)", err, saveErr)
		}
		return fmt.Errorf("stripe partial transfer: %w", err)
	}

	// Transfer succeeded — mark completed with the new amount.
	record.ApplyDisputeResolution(amount, transferID)
	return s.records.Update(ctx, record)
}

// RefundToClient creates a partial or full refund on the original payment.
// If the provider portion is 0 (full refund), marks the record as refunded
// so the wallet excludes it from escrow.
func (s *Service) RefundToClient(ctx context.Context, proposalID uuid.UUID, amount int64) error {
	if amount <= 0 {
		return nil
	}
	record, err := s.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("find payment record: %w", err)
	}
	if record.StripePaymentIntentID == "" {
		return fmt.Errorf("no payment intent for refund")
	}

	_, err = s.stripe.CreateRefund(ctx, record.StripePaymentIntentID, amount)
	if err != nil {
		return fmt.Errorf("stripe refund: %w", err)
	}

	// Full refund (provider gets nothing) → mark the entire payment as refunded
	if record.ProviderPayout == 0 {
		record.MarkRefunded()
	}
	return s.records.Update(ctx, record)
}

// WalletOverview holds the provider's wallet state plus the apporteur's
// commission state when the viewer is a referrer. The two sides are
// independent — a user can have both a provider role (their own
// payouts) and be an apporteur (commissions on referrals they made).
// Frontend renders two sections when both are non-empty.
type WalletOverview struct {
	StripeAccountID   string         `json:"stripe_account_id"`
	ChargesEnabled    bool           `json:"charges_enabled"`
	PayoutsEnabled    bool           `json:"payouts_enabled"`
	EscrowAmount      int64          `json:"escrow_amount"`
	AvailableAmount   int64          `json:"available_amount"`
	TransferredAmount int64          `json:"transferred_amount"`
	Records           []WalletRecord `json:"records"`
	// Referral commission side — populated only when the viewer is an
	// apporteur with commissions. Zero-valued otherwise (UI hides the
	// section when pending+paid+clawed_back == 0).
	Commissions       CommissionWallet          `json:"commissions"`
	CommissionRecords []WalletCommissionRecord  `json:"commission_records"`
}

type WalletRecord struct {
	// ID is the payment_record row id — unique per (proposal, milestone)
	// pair. Exposed so the UI can use a stable React/Flutter key: a
	// proposal with N milestones produces N records that share the same
	// proposal_id, so proposal_id alone is NOT a valid key.
	ID             string `json:"id"`
	ProposalID     string `json:"proposal_id"`
	MilestoneID    string `json:"milestone_id,omitempty"`
	ProposalAmount int64  `json:"proposal_amount"`
	PlatformFee    int64  `json:"platform_fee"`
	ProviderPayout int64  `json:"provider_payout"`
	PaymentStatus  string `json:"payment_status"`
	TransferStatus string `json:"transfer_status"`
	MissionStatus  string `json:"mission_status"` // populated by wallet handler
	CreatedAt      string `json:"created_at"`
}

// CommissionWallet is the aggregate apporteur view: totals grouped by
// commission status. Mirrors the grammar of the provider-side cards
// (escrow / available / transferred) so the UI can reuse the same
// layout for both.
type CommissionWallet struct {
	PendingCents    int64  `json:"pending_cents"`
	PendingKYCCents int64  `json:"pending_kyc_cents"`
	PaidCents       int64  `json:"paid_cents"`
	ClawedBackCents int64  `json:"clawed_back_cents"`
	Currency        string `json:"currency"`
}

// WalletCommissionRecord is one row of the apporteur's commission
// history, ordered newest first by the service layer. Carries enough
// context (referral_id, proposal_id) for the UI to deep-link to the
// relevant referral / project.
type WalletCommissionRecord struct {
	ID               string `json:"id"`
	ReferralID       string `json:"referral_id,omitempty"`
	ProposalID       string `json:"proposal_id,omitempty"`
	MilestoneID      string `json:"milestone_id,omitempty"`
	GrossAmountCents int64  `json:"gross_amount_cents"`
	CommissionCents  int64  `json:"commission_cents"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	StripeTransferID string `json:"stripe_transfer_id,omitempty"`
	PaidAt           string `json:"paid_at,omitempty"`
	ClawedBackAt     string `json:"clawed_back_at,omitempty"`
	CreatedAt        string `json:"created_at"`
}

// GetWalletOverview returns the organization's wallet state. Every
// member of the same org sees the same wallet (Stripe Dashboard model).
// Since phase R5 the Stripe account + KYC bookkeeping live on the org.
func (s *Service) GetWalletOverview(ctx context.Context, userID, orgID uuid.UUID) (*WalletOverview, error) {
	stripeAccountID, _, _ := s.orgs.GetStripeAccount(ctx, orgID)
	_ = userID // kept for audit / future per-operator fields
	wallet := &WalletOverview{StripeAccountID: stripeAccountID}

	// Fetch account capabilities from Stripe so wallet shows if charges/payouts are active
	if stripeAccountID != "" && s.stripe != nil {
		acct, err := s.stripe.GetAccount(ctx, stripeAccountID)
		if err == nil && acct != nil {
			wallet.ChargesEnabled = acct.ChargesEnabled
			wallet.PayoutsEnabled = acct.PayoutsEnabled
		}
	}

	records, err := s.records.ListByOrganization(ctx, orgID)
	if err != nil {
		return wallet, nil
	}

	for _, r := range records {
		rec := WalletRecord{
			ID:             r.ID.String(),
			ProposalID:     r.ProposalID.String(),
			ProposalAmount: r.ProposalAmount,
			PlatformFee:    r.PlatformFeeAmount,
			ProviderPayout: r.ProviderPayout,
			PaymentStatus:  string(r.Status),
			TransferStatus: string(r.TransferStatus),
			CreatedAt:      r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if r.MilestoneID != uuid.Nil {
			rec.MilestoneID = r.MilestoneID.String()
		}
		wallet.Records = append(wallet.Records, rec)

		switch {
		case r.TransferStatus == domain.TransferCompleted:
			wallet.TransferredAmount += r.ProviderPayout
		case r.Status == domain.RecordStatusSucceeded && r.TransferStatus == domain.TransferPending:
			wallet.EscrowAmount += r.ProviderPayout
		}
	}

	wallet.AvailableAmount = wallet.EscrowAmount

	// Commission side — populated only when a referral wallet reader
	// is wired (the referral feature might not be active in every
	// deployment). Failures are swallowed so a broken referral read
	// never takes down the provider-side wallet.
	if s.referralWallet != nil {
		if sum, err := s.referralWallet.GetReferrerSummary(ctx, userID); err == nil {
			wallet.Commissions = CommissionWallet{
				PendingCents:    sum.PendingCents,
				PendingKYCCents: sum.PendingKYCCents,
				PaidCents:       sum.PaidCents,
				ClawedBackCents: sum.ClawedBackCents,
				Currency:        sum.Currency,
			}
		}
		if recent, err := s.referralWallet.RecentCommissions(ctx, userID, 50); err == nil {
			wallet.CommissionRecords = make([]WalletCommissionRecord, 0, len(recent))
			for _, r := range recent {
				rec := WalletCommissionRecord{
					ID:               r.ID.String(),
					GrossAmountCents: r.GrossAmountCents,
					CommissionCents:  r.CommissionCents,
					Currency:         r.Currency,
					Status:           r.Status,
					StripeTransferID: r.StripeTransferID,
					CreatedAt:        r.CreatedAt.Format("2006-01-02T15:04:05Z"),
				}
				if r.ReferralID != uuid.Nil {
					rec.ReferralID = r.ReferralID.String()
				}
				if r.ProposalID != uuid.Nil {
					rec.ProposalID = r.ProposalID.String()
				}
				if r.MilestoneID != uuid.Nil {
					rec.MilestoneID = r.MilestoneID.String()
				}
				if r.PaidAt != nil {
					rec.PaidAt = r.PaidAt.Format("2006-01-02T15:04:05Z")
				}
				if r.ClawedBackAt != nil {
					rec.ClawedBackAt = r.ClawedBackAt.Format("2006-01-02T15:04:05Z")
				}
				wallet.CommissionRecords = append(wallet.CommissionRecords, rec)
			}
		}
	}

	return wallet, nil
}

// PayoutResult is returned by RequestPayout.
type PayoutResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// RequestPayout triggers manual transfers for all pending payments
// belonging to the caller's organization.
//
// Only records whose proposal has reached "completed" are transferred —
// the wallet handler's AvailableAmount already gates the UI on the same
// rule, this enforces it server-side so clicking Retirer cannot pull
// funds still in escrow for an active / disputed mission.
func (s *Service) RequestPayout(ctx context.Context, userID, orgID uuid.UUID) (*PayoutResult, error) {
	_ = userID // audit hook
	stripeAccountID, _, err := s.orgs.GetStripeAccount(ctx, orgID)
	if err != nil || stripeAccountID == "" {
		if errors.Is(err, sql.ErrNoRows) || stripeAccountID == "" {
			return nil, domain.ErrStripeAccountNotFound
		}
		return nil, fmt.Errorf("lookup stripe account: %w", err)
	}

	records, err := s.records.ListByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}

	// Log the degraded-mode warning once per call (not per record) so
	// production logs a single line per payout instead of a flood.
	statusesWired := s.proposalStatuses != nil
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
			status, sErr := s.proposalStatuses.GetProposalStatus(ctx, r.ProposalID)
			if sErr != nil {
				slog.Error("payout: proposal status lookup failed, skipping",
					"proposal_id", r.ProposalID, "error", sErr)
				continue
			}
			if status != "completed" {
				continue
			}
		}

		transferID, tErr := s.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
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
			_ = s.records.Update(ctx, r)
			continue
		}

		if err := r.MarkTransferred(transferID); err != nil {
			slog.Error("mark transferred", "error", err)
			continue
		}
		_ = s.records.Update(ctx, r)
		transferred += r.ProviderPayout
	}

	if transferred == 0 {
		return &PayoutResult{Status: "nothing_to_transfer", Message: "No funds available for transfer"}, nil
	}

	return &PayoutResult{
		Status:  "transferred",
		Message: fmt.Sprintf("Transferred %d centimes to your account", transferred),
	}, nil
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
func (s *Service) RetryFailedTransfer(ctx context.Context, userID, orgID, recordID uuid.UUID) (*PayoutResult, error) {
	record, err := s.records.GetByID(ctx, recordID)
	if err != nil {
		return nil, fmt.Errorf("find payment record: %w", err)
	}

	// Auth: the record's ProviderID must belong to the caller's org.
	// Admin-override is handled at the handler level via RequireRole.
	providerOrg, err := s.orgs.FindByUserID(ctx, record.ProviderID)
	if err != nil || providerOrg == nil {
		return nil, domain.ErrTransferNotRetriable
	}
	if providerOrg.ID != orgID {
		return nil, domain.ErrTransferNotRetriable
	}
	_ = userID // audit hook — user is already authenticated via middleware

	if record.Status != domain.RecordStatusSucceeded || record.TransferStatus != domain.TransferFailed {
		return nil, domain.ErrTransferNotRetriable
	}

	// Same gate as RequestPayout — never open a side-channel that
	// releases escrow funds for an active / disputed mission.
	if s.proposalStatuses != nil {
		status, sErr := s.proposalStatuses.GetProposalStatus(ctx, record.ProposalID)
		if sErr != nil {
			return nil, fmt.Errorf("lookup proposal status: %w", sErr)
		}
		if status != "completed" {
			return nil, domain.ErrTransferNotRetriable
		}
	}

	stripeAccountID, _, accErr := s.orgs.GetStripeAccountByUserID(ctx, record.ProviderID)
	if accErr != nil || stripeAccountID == "" {
		return nil, domain.ErrStripeAccountNotFound
	}

	// Reset status to Pending BEFORE the Stripe call so concurrent
	// reads of the wallet can't see the row stuck as failed. If the
	// retry itself fails, we re-mark Failed below.
	record.TransferStatus = domain.TransferPending
	record.UpdatedAt = time.Now()
	if uErr := s.records.Update(ctx, record); uErr != nil {
		return nil, fmt.Errorf("reset transfer status: %w", uErr)
	}

	transferID, tErr := s.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
		Amount:             record.ProviderPayout,
		Currency:           record.Currency,
		DestinationAccount: stripeAccountID,
		TransferGroup:      record.ProposalID.String(),
		IdempotencyKey:     fmt.Sprintf("transfer_%s_%s", record.ID, stripeAccountID),
	})
	if tErr != nil {
		slog.Error("retry transfer failed", "record_id", record.ID, "proposal_id", record.ProposalID, "error", tErr)
		record.MarkTransferFailed()
		_ = s.records.Update(ctx, record)
		return nil, fmt.Errorf("retry stripe transfer: %w", tErr)
	}

	if mErr := record.MarkTransferred(transferID); mErr != nil {
		return nil, fmt.Errorf("mark transferred: %w", mErr)
	}
	if uErr := s.records.Update(ctx, record); uErr != nil {
		return nil, fmt.Errorf("persist retried transfer: %w", uErr)
	}

	return &PayoutResult{
		Status:  "transferred",
		Message: fmt.Sprintf("Transferred %d centimes to your account", record.ProviderPayout),
	}, nil
}

// VerifyWebhook delegates webhook signature verification to the Stripe adapter.
func (s *Service) VerifyWebhook(payload []byte, signature string) (*portservice.StripeWebhookEvent, error) {
	if s.stripe == nil {
		return nil, errors.New("stripe not configured")
	}
	return s.stripe.ConstructWebhookEvent(payload, signature)
}

// GetPaymentRecord returns the payment record for a proposal.
func (s *Service) GetPaymentRecord(ctx context.Context, proposalID uuid.UUID) (*domain.PaymentRecord, error) {
	return s.records.GetByProposalID(ctx, proposalID)
}

// computePlatformFee looks up the provider's role and returns the flat fee
// from the billing schedule, waived to zero when the provider is a
// Premium subscriber. Returns an error if the provider cannot be resolved
// — creating a payment record without knowing which grid applies would
// skew either the platform (under-charge) or the provider (over-charge),
// so we fail fast on user lookup failure.
//
// Subscription reader failures, by contrast, do NOT fail the payment:
// the Redis-backed cache can degrade, the database can blip, and we must
// not block a live checkout over a cache miss. When the reader errors we
// log + fall back to the full grid fee (the conservative choice: the
// platform keeps its revenue, the user sees the normal fee). A genuinely
// subscribed user affected by this edge case will be refunded the
// milestone fee via support.
func (s *Service) computePlatformFee(ctx context.Context, providerID uuid.UUID, amountCents int64) (int64, error) {
	u, err := s.users.GetByID(ctx, providerID)
	if err != nil {
		return 0, fmt.Errorf("fetch provider: %w", err)
	}
	billingRole := billing.RoleFromUser(string(u.Role))
	fee := billing.Calculate(billingRole, amountCents).FeeCents

	if s.subscriptions != nil {
		active, subErr := s.subscriptions.IsActive(ctx, providerID)
		if subErr != nil {
			slog.Warn("payment: subscription lookup failed, applying full fee",
				"provider_id", providerID, "error", subErr)
			return fee, nil
		}
		if active {
			return 0, nil
		}
	}
	return fee, nil
}

// PreviewFee returns the fee schedule for the authenticated user alongside
// a permission flag that tells the UI whether the caller would actually pay
// the fee on a hypothetical proposal against `recipientID`. Used by the
// web/mobile proposal creation flow to render the live simulator.
//
// Subscription-aware: when the caller has an active Premium subscription,
// the FeeCents is zeroed (and NetCents equals AmountCents). The tier grid
// is still returned so the UI can explain "you would pay X without Premium"
// if it wants — the caller decides how to present the waiver visually.
//
// Visibility rule (single source of truth = proposal.DetermineRoles):
//   - recipientID nil: fallback to role-based default. Enterprise is ALWAYS
//     client so ViewerIsProvider=false. Provider is ALWAYS provider so
//     ViewerIsProvider=true. Agency defaults to true (proposal against an
//     enterprise is the common case); callers that need precise agency
//     resolution MUST pass recipientID.
//   - recipientID set: run DetermineRoles(caller, recipient) and set
//     ViewerIsProvider from the computed provider_id. Invalid combinations
//     (agency+agency, enterprise+enterprise) set the flag to false defensively
//     — the UI must never show fees when the backend cannot confirm the
//     caller is the prestataire.
func (s *Service) PreviewFee(ctx context.Context, userID uuid.UUID, amountCents int64, recipientID *uuid.UUID) (*FeePreviewResult, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("fetch user: %w", err)
	}
	billingRole := billing.RoleFromUser(string(u.Role))
	calc := billing.Calculate(billingRole, amountCents)

	viewerIsProvider := defaultViewerIsProvider(u.Role)
	if recipientID != nil {
		recipient, rErr := s.users.GetByID(ctx, *recipientID)
		if rErr != nil || recipient == nil {
			// Unknown recipient — fail closed rather than leak the fee to a
			// potentially client-side viewer. The UI hides the preview.
			viewerIsProvider = false
		} else {
			_, providerID, drErr := proposaldomain.DetermineRoles(
				userID, string(u.Role),
				*recipientID, string(recipient.Role),
			)
			if drErr != nil {
				viewerIsProvider = false
			} else {
				viewerIsProvider = providerID == userID
			}
		}
	}

	// Waive the fee for Premium subscribers. The tier grid (calc.Tiers
	// and calc.ActiveTierIndex) is kept intact so the UI can still show
	// "Premium → 0 €, normal price would be X" if it chooses.
	viewerIsSubscribed := false
	if s.subscriptions != nil && viewerIsProvider {
		active, sErr := s.subscriptions.IsActive(ctx, userID)
		if sErr != nil {
			slog.Warn("payment: subscription lookup in PreviewFee failed",
				"user_id", userID, "error", sErr)
		} else if active {
			viewerIsSubscribed = true
			calc.FeeCents = 0
			calc.NetCents = calc.AmountCents
		}
	}

	return &FeePreviewResult{
		Billing:            calc,
		ViewerIsProvider:   viewerIsProvider,
		ViewerIsSubscribed: viewerIsSubscribed,
	}, nil
}

// defaultViewerIsProvider is the role-only fallback when no recipient is
// known. Enterprise is ALWAYS the client; everyone else is (likely) the
// provider. Agency defaults to true for the happy path (agency pitching an
// enterprise); edge cases MUST be disambiguated by passing recipientID.
func defaultViewerIsProvider(role domainuser.Role) bool {
	return role != domainuser.RoleEnterprise
}

// CanProviderReceivePayouts reports whether the given provider organization
// has a Stripe Connect account that is ready to receive transfers (account
// id known AND payouts_enabled == true on the live Stripe account).
//
// Used as a pre-check by the proposal milestone-release path so we never
// flip a milestone to "released" + send a "milestone paid" notification
// when the underlying Stripe transfer would fail (no account, KYC
// pending, capability disabled, …).
//
// A nil error with a false bool means "the provider is not ready" — that
// is the expected, non-error happy path for ghost providers. A non-nil
// error means we could not even determine readiness (Stripe API down,
// org lookup failed) — the caller MUST treat this as not-ready and
// surface it the same way to avoid a partial release.
func (s *Service) CanProviderReceivePayouts(ctx context.Context, providerOrgID uuid.UUID) (bool, error) {
	accountID, _, err := s.orgs.GetStripeAccount(ctx, providerOrgID)
	if err != nil {
		return false, fmt.Errorf("get stripe account: %w", err)
	}
	if strings.TrimSpace(accountID) == "" {
		return false, nil
	}
	if s.stripe == nil {
		// No Stripe wired — degrade safely. Same posture as the existing
		// wallet-overview path which silently skips the GetAccount call.
		return false, nil
	}
	info, err := s.stripe.GetAccount(ctx, accountID)
	if err != nil {
		return false, fmt.Errorf("get stripe account capabilities: %w", err)
	}
	if info == nil {
		return false, nil
	}
	return info.PayoutsEnabled, nil
}

