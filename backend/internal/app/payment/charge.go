package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// ChargeService owns the PaymentIntent lifecycle: creation,
// confirmation against Stripe (SEC-02 / BUG-01), webhook handling, and
// signature verification.
//
// SRP rationale: the charge service mutates payment records ONLY in the
// "client → platform escrow" direction. It does not touch the transfer
// state machine (PayoutService) or the read-side wallet shape
// (WalletService). It depends on:
//
//   - records: GetByMilestoneID/GetByProposalID/GetByPaymentIntentID/Create/Update
//   - stripe:  CreatePaymentIntent / GetPaymentIntent / ConstructWebhookEvent
//   - feeCalculator: a tiny port that delegates to WalletService.computePlatformFee
//     so the same fee logic is shared without leaking the wallet's full surface
//
// Notably, the charge service does NOT need: orgs (Stripe account
// resolution, KYC fields), notifications, the referral hooks, or the
// proposal-status reader. Those are payout-side concerns.
type ChargeService struct {
	records repository.PaymentRecordRepository
	stripe  portservice.StripeService

	// feeCalculator is the seam that lets ChargeService delegate
	// platform-fee calculation to WalletService without inheriting its
	// dependency surface. The parent Service wires this to the wallet
	// service's computePlatformFee method.
	feeCalculator platformFeeCalculator
}

// platformFeeCalculator is the narrow port the charge service relies on
// to compute the platform fee at PaymentIntent creation. WalletService
// satisfies it natively; tests can pass a tiny fake.
type platformFeeCalculator interface {
	computePlatformFee(ctx context.Context, providerID uuid.UUID, amountCents int64) (int64, error)
}

// ChargeServiceDeps groups every dependency NewChargeService needs.
type ChargeServiceDeps struct {
	Records       repository.PaymentRecordRepository
	Stripe        portservice.StripeService
	FeeCalculator platformFeeCalculator
}

// NewChargeService wires the PaymentIntent lifecycle sub-service.
func NewChargeService(deps ChargeServiceDeps) *ChargeService {
	return &ChargeService{
		records:       deps.Records,
		stripe:        deps.Stripe,
		feeCalculator: deps.FeeCalculator,
	}
}

// CreatePaymentIntent creates a Stripe PaymentIntent for a milestone
// payment. Phase 4: the idempotency key is the milestone id, not the
// proposal id, so a proposal with N milestones can be funded N times
// (one PaymentIntent per milestone) without the second call reusing
// the first milestone's intent.
func (c *ChargeService) CreatePaymentIntent(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	if c.stripe == nil {
		return nil, errors.New("stripe not configured")
	}

	existing, err := c.records.GetByMilestoneID(ctx, input.MilestoneID)
	if err == nil && existing != nil {
		return c.createPaymentIntentFromExisting(ctx, input)
	}

	stripeFee := domain.EstimateStripeFee(input.ProposalAmount)

	// Platform fee is computed from the billing schedule using the
	// provider's role (agency pays the agency grid, everyone else pays
	// the freelance grid). The fee is frozen into the payment_record
	// row at creation time — future schedule changes never retro-modify
	// historical records.
	platformFee, err := c.feeCalculator.computePlatformFee(ctx, input.ProviderID, input.ProposalAmount)
	if err != nil {
		return nil, fmt.Errorf("compute platform fee: %w", err)
	}

	record := domain.NewPaymentRecord(
		input.ProposalID, input.MilestoneID, input.ClientID, input.ProviderID,
		input.ProposalAmount, stripeFee, platformFee,
	)

	pi, err := c.stripe.CreatePaymentIntent(ctx, portservice.CreatePaymentIntentInput{
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

	if err := c.records.Create(ctx, record); err != nil {
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
func (c *ChargeService) createPaymentIntentFromExisting(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	existing, err := c.records.GetByMilestoneID(ctx, input.MilestoneID)
	if err != nil {
		return nil, fmt.Errorf("fetch existing record: %w", err)
	}

	pi, err := c.stripe.CreatePaymentIntent(ctx, portservice.CreatePaymentIntentInput{
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
		// BUG-09: previously this branch did `_ = c.records.Update(...)`
		// which silently swallowed any DB error. The
		// StripePaymentIntentID is the link to the live Stripe charge —
		// if we lose it, future transfer/refund calls target a phantom
		// PI ID. Surface the error so the caller can retry instead of
		// returning a working ClientSecret backed by a broken record.
		existing.StripePaymentIntentID = pi.PaymentIntentID
		if err := c.records.Update(ctx, existing); err != nil {
			slog.Error("payment: failed to persist new PI id on existing record — record desynced from Stripe",
				"record_id", existing.ID,
				"proposal_id", input.ProposalID,
				"milestone_id", input.MilestoneID,
				"new_payment_intent_id", pi.PaymentIntentID,
				"error", err,
			)
			return nil, fmt.Errorf("persist new payment intent id: %w", err)
		}
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

// MarkPaymentSucceeded marks a payment record as paid AFTER verifying
// with Stripe that the underlying PaymentIntent has actually settled.
//
// Closes SEC-02 / BUG-01: previously, this method trusted the local
// record blindly — a client with DevTools could POST
// /proposals/{id}/confirm-payment and have the record flipped to
// `succeeded`, the proposal activated, and (on completion) funds
// transferred to the provider — all without any real Stripe charge
// having cleared.
//
// The check fetches the PaymentIntent and asserts pi.Status ==
// "succeeded" before delegating to record.MarkPaid(). Any other status
// returns domain.ErrPaymentNotConfirmed; missing PI ID or Stripe API
// error returns the wrapped error so the caller can decide whether to
// retry. Idempotency (record already in non-pending state) is
// preserved.
//
// TODO(SEC-13): Once audit logging is wired by Agent A, emit
// `payment_confirm_attempt_unverified` here when ErrPaymentNotConfirmed
// fires — that is the signal of a possible fraud attempt.
func (c *ChargeService) MarkPaymentSucceeded(ctx context.Context, proposalID uuid.UUID) error {
	record, err := c.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("find record: %w", err)
	}

	// Idempotent fast-path — already non-pending, nothing to do.
	if record.Status != domain.RecordStatusPending {
		return nil
	}

	// Stripe verification — the heart of the SEC-02 fix.
	if c.stripe == nil {
		return errors.New("stripe service not configured")
	}
	if record.StripePaymentIntentID == "" {
		slog.Warn("payment confirm: record has no PaymentIntent id",
			"proposal_id", proposalID, "record_id", record.ID)
		return domain.ErrPaymentNotConfirmed
	}
	pi, err := c.stripe.GetPaymentIntent(ctx, record.StripePaymentIntentID)
	if err != nil {
		return fmt.Errorf("verify payment intent: %w", err)
	}
	if pi == nil || pi.Status != "succeeded" {
		piStatus := ""
		if pi != nil {
			piStatus = pi.Status
		}
		slog.Warn("payment confirm: stripe says not succeeded",
			"proposal_id", proposalID,
			"record_id", record.ID,
			"payment_intent_id", record.StripePaymentIntentID,
			"stripe_status", piStatus)
		return domain.ErrPaymentNotConfirmed
	}

	if err := record.MarkPaid(); err != nil {
		// Idempotent — if already in non-pending state, treat as success.
		if errors.Is(err, domain.ErrPaymentNotPending) {
			return nil
		}
		return err
	}
	return c.records.Update(ctx, record)
}

// HandlePaymentSucceeded handles the payment_intent.succeeded webhook
// event. Returns the proposal_id so the proposal service can activate
// the mission.
func (c *ChargeService) HandlePaymentSucceeded(ctx context.Context, paymentIntentID string) (uuid.UUID, error) {
	record, err := c.records.GetByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("find record: %w", err)
	}
	if err := record.MarkPaid(); err != nil {
		if errors.Is(err, domain.ErrPaymentNotPending) {
			return record.ProposalID, nil // idempotent
		}
		return uuid.Nil, err
	}
	if err := c.records.Update(ctx, record); err != nil {
		return uuid.Nil, fmt.Errorf("update record: %w", err)
	}
	return record.ProposalID, nil
}

// VerifyWebhook delegates webhook signature verification to the Stripe
// adapter.
func (c *ChargeService) VerifyWebhook(payload []byte, signature string) (*portservice.StripeWebhookEvent, error) {
	if c.stripe == nil {
		return nil, errors.New("stripe not configured")
	}
	return c.stripe.ConstructWebhookEvent(payload, signature)
}
