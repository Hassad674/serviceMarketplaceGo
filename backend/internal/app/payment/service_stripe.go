package payment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	portservice "marketplace-backend/internal/port/service"
)

// CreatePaymentIntent implements service.PaymentProcessor.
func (s *Service) CreatePaymentIntent(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	// Check for existing record (idempotency)
	existing, err := s.records.GetByProposalID(ctx, input.ProposalID)
	if err == nil && existing != nil {
		if existing.StripePaymentIntentID != "" {
			return &portservice.PaymentIntentOutput{
				PaymentRecordID: existing.ID,
				ProposalAmount:  existing.ProposalAmount,
				StripeFee:       existing.StripeFeeAmount,
				PlatformFee:     existing.PlatformFeeAmount,
				ClientTotal:     existing.ClientTotalAmount,
				ProviderPayout:  existing.ProviderPayout,
			}, domain.ErrPaymentAlreadyExists
		}
	}

	stripeFee := domain.EstimateStripeFee(input.ProposalAmount)
	record := domain.NewPaymentRecord(input.ProposalID, input.ClientID, input.ProviderID, input.ProposalAmount, stripeFee)

	if err := s.records.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("persist payment record: %w", err)
	}

	pi, err := s.stripe.CreatePaymentIntent(ctx, portservice.CreatePaymentIntentInput{
		AmountCentimes: record.ClientTotalAmount,
		Currency:       record.Currency,
		ProposalID:     input.ProposalID.String(),
		ClientID:       input.ClientID.String(),
		ProviderID:     input.ProviderID.String(),
		TransferGroup:  input.ProposalID.String(),
	})
	if err != nil {
		record.MarkFailed()
		_ = s.records.Update(ctx, record)
		return nil, fmt.Errorf("create stripe payment intent: %w", err)
	}

	record.StripePaymentIntentID = pi.PaymentIntentID
	if err := s.records.Update(ctx, record); err != nil {
		return nil, fmt.Errorf("update record with PI ID: %w", err)
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

// HandlePaymentSucceeded implements service.PaymentProcessor.
func (s *Service) HandlePaymentSucceeded(ctx context.Context, paymentIntentID string) (uuid.UUID, error) {
	record, err := s.records.GetByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("find payment record: %w", err)
	}

	// Idempotency: already succeeded
	if record.Status == domain.RecordStatusSucceeded {
		return record.ProposalID, nil
	}

	if err := record.MarkPaid(); err != nil {
		return uuid.Nil, fmt.Errorf("mark paid: %w", err)
	}

	if err := s.records.Update(ctx, record); err != nil {
		return uuid.Nil, fmt.Errorf("update payment record: %w", err)
	}

	return record.ProposalID, nil
}

// TransferToProvider implements service.PaymentProcessor.
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

	providerInfo, err := s.payments.GetByUserID(ctx, record.ProviderID)
	if err != nil {
		return fmt.Errorf("get provider payment info: %w", err)
	}
	if providerInfo.StripeAccountID == "" {
		return domain.ErrStripeAccountNotFound
	}

	transferID, err := s.stripe.CreateTransfer(ctx, portservice.CreateTransferInput{
		Amount:             record.ProviderPayout,
		Currency:           record.Currency,
		DestinationAccount: providerInfo.StripeAccountID,
		TransferGroup:      proposalID.String(),
		IdempotencyKey:     "transfer_" + proposalID.String(),
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

// HandleAccountUpdated syncs Stripe connected account verification status.
func (s *Service) HandleAccountUpdated(ctx context.Context, accountID string) error {
	verified, err := s.stripe.GetAccountStatus(ctx, accountID)
	if err != nil {
		return fmt.Errorf("get account status: %w", err)
	}
	if !verified {
		return nil
	}

	// Find the payment info with this stripe account ID and mark verified
	// For now, we log it. A dedicated query would be more efficient.
	slog.Info("stripe account verified", "account_id", accountID, "verified", verified)
	return nil
}

// ensureStripeAccount creates a Stripe connected account if conditions are met.
func (s *Service) ensureStripeAccount(ctx context.Context, info *domain.PaymentInfo, tosIP string) {
	if s.stripe == nil || info.StripeAccountID != "" || !info.IsComplete() || tosIP == "" {
		return
	}

	accountID, err := s.stripe.CreateConnectedAccount(ctx, info, tosIP)
	if err != nil {
		slog.Error("failed to create stripe connected account", "user_id", info.UserID, "error", err)
		return
	}

	info.SetStripeAccount(accountID)
	if err := s.payments.UpdateStripeFields(ctx, info.UserID, accountID, false); err != nil {
		slog.Error("failed to persist stripe account id", "user_id", info.UserID, "error", err)
	}

	slog.Info("stripe connected account created", "user_id", info.UserID, "account_id", accountID)
}

// VerifyWebhook delegates webhook verification to the Stripe adapter.
func (s *Service) VerifyWebhook(payload []byte, signature string) (*portservice.StripeWebhookEvent, error) {
	if s.stripe == nil {
		return nil, fmt.Errorf("stripe not configured")
	}
	return s.stripe.ConstructWebhookEvent(payload, signature)
}

// StripeConfigured returns true when the Stripe service is available.
func (s *Service) StripeConfigured() bool {
	return s.stripe != nil
}

// GetPaymentRecord returns the payment record for a proposal.
func (s *Service) GetPaymentRecord(ctx context.Context, proposalID uuid.UUID) (*domain.PaymentRecord, error) {
	record, err := s.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get payment record: %w", err)
	}
	return record, nil
}
