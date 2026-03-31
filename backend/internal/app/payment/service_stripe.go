package payment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// CreatePaymentIntent implements service.PaymentProcessor.
func (s *Service) CreatePaymentIntent(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	existing, err := s.records.GetByProposalID(ctx, input.ProposalID)
	if err == nil && existing != nil {
		return s.retryExistingPaymentIntent(ctx, existing, input)
	}

	stripeFee := domain.EstimateStripeFee(input.ProposalAmount)
	record := domain.NewPaymentRecord(input.ProposalID, input.ClientID, input.ProviderID, input.ProposalAmount, stripeFee)

	if err := s.records.Create(ctx, record); err != nil {
		return s.createPaymentIntentFromExisting(ctx, input)
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

	return paymentIntentOutputFromRecord(pi.ClientSecret, record), nil
}

func (s *Service) retryExistingPaymentIntent(ctx context.Context, existing *domain.PaymentRecord, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
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
	return paymentIntentOutputFromRecord(pi.ClientSecret, existing), nil
}

func (s *Service) createPaymentIntentFromExisting(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	existing, err := s.records.GetByProposalID(ctx, input.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("fetch existing record after race: %w", err)
	}
	return s.retryExistingPaymentIntent(ctx, existing, input)
}

func paymentIntentOutputFromRecord(clientSecret string, r *domain.PaymentRecord) *portservice.PaymentIntentOutput {
	return &portservice.PaymentIntentOutput{
		ClientSecret:    clientSecret,
		PaymentRecordID: r.ID,
		ProposalAmount:  r.ProposalAmount,
		StripeFee:       r.StripeFeeAmount,
		PlatformFee:     r.PlatformFeeAmount,
		ClientTotal:     r.ClientTotalAmount,
		ProviderPayout:  r.ProviderPayout,
	}
}

// MarkPaymentSucceeded marks the payment record as succeeded by proposal ID.
func (s *Service) MarkPaymentSucceeded(ctx context.Context, proposalID uuid.UUID) error {
	record, err := s.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("find payment record: %w", err)
	}
	if record.Status == domain.RecordStatusSucceeded {
		return nil
	}
	if err := record.MarkPaid(); err != nil {
		return fmt.Errorf("mark paid: %w", err)
	}
	return s.records.Update(ctx, record)
}

// HandlePaymentSucceeded implements service.PaymentProcessor.
func (s *Service) HandlePaymentSucceeded(ctx context.Context, paymentIntentID string) (uuid.UUID, error) {
	record, err := s.records.GetByPaymentIntentID(ctx, paymentIntentID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("find payment record: %w", err)
	}
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
		IdempotencyKey:     fmt.Sprintf("transfer_%s_%s", proposalID, providerInfo.StripeAccountID),
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

// HandleAccountUpdated syncs Stripe connected account status.
// Called by the webhook handler when account.updated fires.
func (s *Service) HandleAccountUpdated(ctx context.Context, accountID string) error {
	acctInfo, err := s.stripe.GetFullAccount(ctx, accountID)
	if err != nil {
		return fmt.Errorf("get full account: %w", err)
	}

	info, err := s.payments.GetByStripeAccountID(ctx, accountID)
	if err != nil {
		slog.Warn("webhook: no user for stripe account", "account_id", accountID)
		return nil
	}

	verified := acctInfo.ChargesEnabled && acctInfo.PayoutsEnabled
	syncInput := repository.StripeSyncInput{
		ChargesEnabled: acctInfo.ChargesEnabled,
		PayoutsEnabled: acctInfo.PayoutsEnabled,
		StripeVerified: verified,
		BusinessType:   acctInfo.BusinessType,
		Country:        acctInfo.Country,
		DisplayName:    acctInfo.DisplayName,
	}

	if err := s.payments.UpdateStripeSyncFields(ctx, info.UserID, syncInput); err != nil {
		slog.Error("webhook: failed to sync stripe fields", "user_id", info.UserID, "error", err)
	}

	// Notify user if there are pending requirements
	if len(acctInfo.CurrentlyDue) > 0 {
		s.notifyNewRequirements(ctx, info.UserID, acctInfo.CurrentlyDue)
	}

	return nil
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
	MissionStatus  string `json:"mission_status"`
	CreatedAt      string `json:"created_at"`
}

// GetWalletOverview returns the provider's wallet state.
func (s *Service) GetWalletOverview(ctx context.Context, userID uuid.UUID) (*WalletOverview, error) {
	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil {
		return &WalletOverview{}, nil
	}

	wallet := &WalletOverview{
		StripeAccountID: info.StripeAccountID,
	}

	if s.stripe != nil && info.StripeAccountID != "" {
		verified, _ := s.stripe.GetAccountStatus(ctx, info.StripeAccountID)
		wallet.ChargesEnabled = verified
		wallet.PayoutsEnabled = verified
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

type PayoutResult struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// RequestPayout triggers a manual payout from the connected account.
func (s *Service) RequestPayout(ctx context.Context, userID uuid.UUID) (*PayoutResult, error) {
	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get payment info: %w", err)
	}
	if info.StripeAccountID == "" {
		return nil, domain.ErrStripeAccountNotFound
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
			DestinationAccount: info.StripeAccountID,
			TransferGroup:      r.ProposalID.String(),
			IdempotencyKey:     fmt.Sprintf("transfer_%s_%s", r.ProposalID, info.StripeAccountID),
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

// notifyNewRequirements sends a notification when Stripe requires new information.
func (s *Service) notifyNewRequirements(ctx context.Context, userID uuid.UUID, requirements []string) {
	if s.notifications == nil || len(requirements) == 0 {
		return
	}

	data, _ := json.Marshal(map[string]string{
		"type": "stripe_requirements",
		"url":  "/payment-info",
	})

	if err := s.notifications.Send(ctx, portservice.NotificationInput{
		UserID: userID,
		Type:   "stripe_requirements",
		Title:  "Action requise — Stripe",
		Body:   "Stripe demande des informations complémentaires pour activer votre compte.",
		Data:   data,
	}); err != nil {
		slog.Error("failed to send stripe requirements notification", "user_id", userID, "error", err)
	}
}
