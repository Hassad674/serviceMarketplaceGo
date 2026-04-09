package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	portservice "marketplace-backend/internal/port/service"
)

// CreatePaymentIntent creates a Stripe PaymentIntent for a proposal payment.
// If a payment_record already exists for this proposal (idempotent retry),
// Stripe returns the same PaymentIntent via the idempotency key.
func (s *Service) CreatePaymentIntent(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	if s.stripe == nil {
		return nil, errors.New("stripe not configured")
	}

	existing, err := s.records.GetByProposalID(ctx, input.ProposalID)
	if err == nil && existing != nil {
		return s.createPaymentIntentFromExisting(ctx, input)
	}

	stripeFee := domain.EstimateStripeFee(input.ProposalAmount)
	record := domain.NewPaymentRecord(
		input.ProposalID, input.ClientID, input.ProviderID,
		input.ProposalAmount, stripeFee,
	)

	pi, err := s.stripe.CreatePaymentIntent(ctx, portservice.CreatePaymentIntentInput{
		AmountCentimes: record.ClientTotalAmount,
		Currency:       "eur",
		ProposalID:     input.ProposalID.String(),
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
// existing record (idempotent via Stripe's idempotency key).
func (s *Service) createPaymentIntentFromExisting(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	existing, err := s.records.GetByProposalID(ctx, input.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("fetch existing record: %w", err)
	}

	pi, err := s.stripe.CreatePaymentIntent(ctx, portservice.CreatePaymentIntentInput{
		AmountCentimes: existing.ClientTotalAmount,
		Currency:       existing.Currency,
		ProposalID:     input.ProposalID.String(),
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

// TransferToProvider creates a Stripe transfer from the platform balance
// to the provider's connected account for a succeeded payment.
func (s *Service) TransferToProvider(ctx context.Context, proposalID uuid.UUID) error {
	record, err := s.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("find payment record: %w", err)
	}

	if record.Status != domain.RecordStatusSucceeded {
		return domain.ErrPaymentNotSucceeded
	}
	if record.TransferStatus != domain.TransferPending {
		return domain.ErrTransferAlreadyDone
	}

	stripeAccountID, _, err := s.users.GetStripeAccount(ctx, record.ProviderID)
	if err != nil || stripeAccountID == "" {
		return domain.ErrStripeAccountNotFound
	}

	transferID, err := s.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
		Amount:             record.ProviderPayout,
		Currency:           record.Currency,
		DestinationAccount: stripeAccountID,
		TransferGroup:      proposalID.String(),
		IdempotencyKey:     fmt.Sprintf("transfer_%s_%s", proposalID, stripeAccountID),
	})
	if err != nil {
		record.MarkTransferFailed()
		_ = s.records.Update(ctx, record)
		return fmt.Errorf("create stripe transfer: %w", err)
	}

	if err := record.MarkTransferred(transferID); err != nil {
		return fmt.Errorf("mark transferred: %w", err)
	}

	return s.records.Update(ctx, record)
}

// TransferPartialToProvider transfers a specific amount to the provider and
// updates the payment record to reflect the new ProviderPayout. Used for dispute
// resolutions: the wallet computation must show the actual amount, not the
// original proposal payout.
func (s *Service) TransferPartialToProvider(ctx context.Context, proposalID uuid.UUID, amount int64) error {
	record, err := s.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("find payment record: %w", err)
	}
	if record.Status != domain.RecordStatusSucceeded {
		return domain.ErrPaymentNotSucceeded
	}

	var transferID string
	if amount > 0 {
		stripeAccountID, _, accErr := s.users.GetStripeAccount(ctx, record.ProviderID)
		if accErr != nil || stripeAccountID == "" {
			return domain.ErrStripeAccountNotFound
		}

		transferID, err = s.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
			Amount:             amount,
			Currency:           record.Currency,
			DestinationAccount: stripeAccountID,
			TransferGroup:      proposalID.String(),
			IdempotencyKey:     fmt.Sprintf("dispute_transfer_%s_%d", proposalID, amount),
		})
		if err != nil {
			return fmt.Errorf("stripe partial transfer: %w", err)
		}
	}

	// Update the record so the wallet computation reflects the actual amount.
	// If amount == 0 (full refund), provider gets nothing — still mark as
	// resolved so the funds are no longer in escrow.
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

// WalletOverview holds the provider's wallet state.
type WalletOverview struct {
	StripeAccountID   string         `json:"stripe_account_id"`
	ChargesEnabled    bool           `json:"charges_enabled"`
	PayoutsEnabled    bool           `json:"payouts_enabled"`
	EscrowAmount      int64          `json:"escrow_amount"`
	AvailableAmount   int64          `json:"available_amount"`
	TransferredAmount int64          `json:"transferred_amount"`
	Records           []WalletRecord `json:"records"`
}

type WalletRecord struct {
	ProposalID     string `json:"proposal_id"`
	ProposalAmount int64  `json:"proposal_amount"`
	PlatformFee    int64  `json:"platform_fee"`
	ProviderPayout int64  `json:"provider_payout"`
	PaymentStatus  string `json:"payment_status"`
	TransferStatus string `json:"transfer_status"`
	MissionStatus  string `json:"mission_status"` // populated by wallet handler
	CreatedAt      string `json:"created_at"`
}

// GetWalletOverview returns the provider's wallet state.
func (s *Service) GetWalletOverview(ctx context.Context, userID uuid.UUID) (*WalletOverview, error) {
	stripeAccountID, _, _ := s.users.GetStripeAccount(ctx, userID)
	wallet := &WalletOverview{StripeAccountID: stripeAccountID}

	// Fetch account capabilities from Stripe so wallet shows if charges/payouts are active
	if stripeAccountID != "" && s.stripe != nil {
		acct, err := s.stripe.GetAccount(ctx, stripeAccountID)
		if err == nil && acct != nil {
			wallet.ChargesEnabled = acct.ChargesEnabled
			wallet.PayoutsEnabled = acct.PayoutsEnabled
		}
	}

	records, err := s.records.ListByProviderID(ctx, userID)
	if err != nil {
		return wallet, nil
	}

	for _, r := range records {
		wallet.Records = append(wallet.Records, WalletRecord{
			ProposalID:     r.ProposalID.String(),
			ProposalAmount: r.ProposalAmount,
			PlatformFee:    r.PlatformFeeAmount,
			ProviderPayout: r.ProviderPayout,
			PaymentStatus:  string(r.Status),
			TransferStatus: string(r.TransferStatus),
			CreatedAt:      r.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})

		switch {
		case r.TransferStatus == domain.TransferCompleted:
			wallet.TransferredAmount += r.ProviderPayout
		case r.Status == domain.RecordStatusSucceeded && r.TransferStatus == domain.TransferPending:
			wallet.EscrowAmount += r.ProviderPayout
		}
	}

	wallet.AvailableAmount = wallet.EscrowAmount
	return wallet, nil
}

// PayoutResult is returned by RequestPayout.
type PayoutResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// RequestPayout triggers manual transfers for all pending payments.
func (s *Service) RequestPayout(ctx context.Context, userID uuid.UUID) (*PayoutResult, error) {
	stripeAccountID, _, err := s.users.GetStripeAccount(ctx, userID)
	if err != nil || stripeAccountID == "" {
		if errors.Is(err, sql.ErrNoRows) || stripeAccountID == "" {
			return nil, domain.ErrStripeAccountNotFound
		}
		return nil, fmt.Errorf("lookup stripe account: %w", err)
	}

	records, err := s.records.ListByProviderID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list records: %w", err)
	}

	var transferred int64
	for _, r := range records {
		if r.Status != domain.RecordStatusSucceeded || r.TransferStatus != domain.TransferPending {
			continue
		}

		transferID, tErr := s.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
			Amount:             r.ProviderPayout,
			Currency:           r.Currency,
			DestinationAccount: stripeAccountID,
			TransferGroup:      r.ProposalID.String(),
			IdempotencyKey:     fmt.Sprintf("transfer_%s_%s", r.ProposalID, stripeAccountID),
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

