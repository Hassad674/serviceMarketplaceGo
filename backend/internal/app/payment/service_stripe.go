package payment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
	portservice "marketplace-backend/internal/port/service"
)

// CreatePaymentIntent implements service.PaymentProcessor.
func (s *Service) CreatePaymentIntent(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	// Check for existing record (idempotency)
	existing, err := s.records.GetByProposalID(ctx, input.ProposalID)
	if err == nil && existing != nil {
		// Record exists — re-call Stripe with same idempotency key (returns same PI)
		pi, piErr := s.stripe.CreatePaymentIntent(ctx, portservice.CreatePaymentIntentInput{
			AmountCentimes: existing.ClientTotalAmount,
			Currency:       existing.Currency,
			ProposalID:     input.ProposalID.String(),
			ClientID:       input.ClientID.String(),
			ProviderID:     input.ProviderID.String(),
			TransferGroup:  input.ProposalID.String(),
		})
		if piErr != nil {
			return nil, fmt.Errorf("retrieve existing payment intent: %w", piErr)
		}
		// Update record with PI ID if it was missing (race condition recovery)
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

	stripeFee := domain.EstimateStripeFee(input.ProposalAmount)
	record := domain.NewPaymentRecord(input.ProposalID, input.ClientID, input.ProviderID, input.ProposalAmount, stripeFee)

	if err := s.records.Create(ctx, record); err != nil {
		// Race condition: another request just created it — fetch and return
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

// createPaymentIntentFromExisting handles the race condition where another request
// already created the record. Fetches the existing record and returns the PI.
func (s *Service) createPaymentIntentFromExisting(ctx context.Context, input portservice.PaymentIntentInput) (*portservice.PaymentIntentOutput, error) {
	existing, err := s.records.GetByProposalID(ctx, input.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("fetch existing record after race: %w", err)
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
		return nil, fmt.Errorf("create PI from existing: %w", err)
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

// MarkPaymentSucceeded marks the payment record as succeeded by proposal ID.
// Called by the confirm-payment handler as a fallback to the webhook.
func (s *Service) MarkPaymentSucceeded(ctx context.Context, proposalID uuid.UUID) error {
	record, err := s.records.GetByProposalID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("find payment record: %w", err)
	}
	if record.Status == domain.RecordStatusSucceeded {
		return nil // already succeeded
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

// HandleAccountUpdated syncs Stripe connected account verification status.
// Called by the webhook handler when account.updated fires.
func (s *Service) HandleAccountUpdated(ctx context.Context, accountID string) error {
	// Get full account status in one API call (verification + charges + payouts)
	fullStatus, err := s.stripe.GetAccountFullStatus(ctx, accountID)
	if err != nil {
		return fmt.Errorf("get account full status: %w", err)
	}

	verStatus := fullStatus.VerificationStatus
	verifiedFileID := fullStatus.VerifiedFileID

	// Find the user associated with this Stripe account
	info, err := s.payments.GetByStripeAccountID(ctx, accountID)
	if err != nil {
		slog.Warn("webhook: no user for stripe account", "account_id", accountID)
		return nil
	}

	// Update account-level verification
	if verStatus == "verified" {
		if err := s.payments.UpdateStripeFields(ctx, info.UserID, accountID, true); err != nil {
			slog.Error("webhook: failed to mark stripe verified", "user_id", info.UserID, "error", err)
		}
	}

	// Detect account status changes and notify
	chargesChanged := info.ChargesEnabled != fullStatus.ChargesEnabled
	payoutsChanged := info.PayoutsEnabled != fullStatus.PayoutsEnabled
	if chargesChanged || payoutsChanged {
		if err := s.payments.UpdateAccountStatus(ctx, info.UserID, fullStatus.ChargesEnabled, fullStatus.PayoutsEnabled); err != nil {
			slog.Error("webhook: failed to update account status", "user_id", info.UserID, "error", err)
		}
		s.notifyAccountStatusChange(ctx, info.UserID, fullStatus.ChargesEnabled, fullStatus.PayoutsEnabled)
	}

	// Update identity document statuses
	docs, err := s.documents.ListByUserID(ctx, info.UserID)
	if err != nil {
		return nil
	}

	for _, d := range docs {
		if d.Status != domain.DocStatusPending {
			continue
		}
		switch verStatus {
		case "verified":
			if d.StripeFileID == verifiedFileID || verifiedFileID == "" {
				_ = s.documents.UpdateStatus(ctx, d.ID, string(domain.DocStatusVerified), "")
				slog.Info("webhook: document verified", "doc_id", d.ID, "user_id", info.UserID)
			}
		case "unverified":
			_ = s.documents.UpdateStatus(ctx, d.ID, string(domain.DocStatusRejected), "verification failed")
			slog.Info("webhook: document rejected", "doc_id", d.ID, "user_id", info.UserID)
		case "pending":
			// Keep as pending — don't change status
		}
	}

	// Check for new requirements and notify (only for urgent requirements)
	reqs, reqErr := s.stripe.GetAccountRequirements(ctx, accountID)
	if reqErr == nil && (len(reqs.CurrentlyDue) > 0 || len(reqs.PastDue) > 0) {
		s.NotifyNewRequirements(ctx, info.UserID, reqs)
	}

	return nil
}

// statusCooldown prevents duplicate account status notifications.
var statusCooldown sync.Map

// notifyAccountStatusChange sends a notification when Stripe account charges/payouts status changes.
func (s *Service) notifyAccountStatusChange(ctx context.Context, userID uuid.UUID, charges, payouts bool) {
	if s.notifications == nil {
		return
	}
	// Cooldown: max 1 status notification per user per 5 minutes
	if lastSent, ok := statusCooldown.Load(userID); ok {
		if time.Since(lastSent.(time.Time)) < 5*time.Minute {
			return
		}
	}
	statusCooldown.Store(userID, time.Now())

	var title, body string
	switch {
	case charges && payouts:
		title = "Compte Stripe activ\u00e9"
		body = "Votre compte est maintenant actif. Vous pouvez recevoir des paiements."
	case !payouts:
		title = "Virements suspendus \u2014 Stripe"
		body = "Les virements vers votre compte ont \u00e9t\u00e9 suspendus. V\u00e9rifiez vos informations."
	case !charges:
		title = "Paiements suspendus \u2014 Stripe"
		body = "Votre capacit\u00e9 \u00e0 recevoir des paiements a \u00e9t\u00e9 suspendue par Stripe."
	}

	data, _ := json.Marshal(map[string]string{
		"type": "stripe_account_status",
		"url":  "/payment-info",
	})

	if err := s.notifications.Send(ctx, portservice.NotificationInput{
		UserID: userID,
		Type:   "stripe_account_status",
		Title:  title,
		Body:   body,
		Data:   data,
	}); err != nil {
		slog.Error("failed to send account status notification", "user_id", userID, "error", err)
	}
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

	// Check Stripe account status
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

	// Find all succeeded payments with pending transfers
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

// updateStripeAccount updates an existing Stripe account with new payment info.
// Returns an error if Stripe rejects the data so the caller can surface it.
func (s *Service) updateStripeAccount(ctx context.Context, info *domain.PaymentInfo, tosIP string, email string) error {
	if s.stripe == nil || info.StripeAccountID == "" || tosIP == "" {
		return nil
	}

	if err := s.stripe.UpdateConnectedAccount(ctx, info.StripeAccountID, info, tosIP, email); err != nil {
		slog.Error("failed to update stripe account", "user_id", info.UserID, "account_id", info.StripeAccountID, "error", err)
		return err
	}

	// Create or update persons for business accounts
	if info.IsBusiness {
		if !s.stripe.HasPersons(ctx, info.StripeAccountID) {
			s.createStripePersons(ctx, info, info.StripeAccountID, email)
		} else {
			// Update existing representative with latest data
			repInput := portservice.CreatePersonInput{
				FirstName: info.FirstName,
				LastName:  info.LastName,
				Email:     firstNonEmpty(info.Email, email),
				Phone:     firstNonEmpty(getPersonExtra(info.ExtraFields, "phone"), info.Phone),
				DOB:       info.DateOfBirth,
				Address:   info.Address,
				City:      info.City,
				PostalCode: info.PostalCode,
				Country:   info.Country,
				State:     getExtra(info.ExtraFields, "representative.address.state", "state"),
				Title:     firstNonEmpty(info.RoleInCompany, getExtraSuffix(info.ExtraFields, "relationship.title")),
			}
			if err := s.stripe.UpdateRepresentativePerson(ctx, info.StripeAccountID, repInput); err != nil {
				slog.Warn("failed to update representative person", "error", err)
			}
		}
	}

	slog.Info("stripe account updated", "user_id", info.UserID, "account_id", info.StripeAccountID)
	return nil
}

// ensureStripeAccount creates a Stripe connected account if conditions are met.
// Returns an error if Stripe rejects the data so the caller can surface it.
func (s *Service) ensureStripeAccount(ctx context.Context, info *domain.PaymentInfo, tosIP string, email string) error {
	if s.stripe == nil || info.StripeAccountID != "" || !info.IsComplete() || tosIP == "" {
		return nil
	}

	accountID, err := s.stripe.CreateConnectedAccount(ctx, info, tosIP, email)
	if err != nil {
		slog.Error("failed to create stripe connected account", "user_id", info.UserID, "error", err)
		return err
	}

	info.SetStripeAccount(accountID)
	if err := s.payments.UpdateStripeFields(ctx, info.UserID, accountID, false); err != nil {
		slog.Error("failed to persist stripe account id", "user_id", info.UserID, "error", err)
	}

	slog.Info("stripe connected account created", "user_id", info.UserID, "account_id", accountID)

	// Create Stripe persons for business accounts
	if info.IsBusiness {
		s.createStripePersons(ctx, info, accountID, email)
	}

	// Sync any documents uploaded before the account was created
	s.syncPendingDocuments(ctx, info.UserID, accountID)

	return nil
}

// firstNonEmpty returns the first non-empty string from the arguments.
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// getPersonExtra finds a Person field value from extra_fields.
// Looks for keys starting with "person_" or "representative." ending with the given field name.
func getPersonExtra(extra map[string]string, field string) string {
	suffix := "." + field
	for k, v := range extra {
		if (strings.HasPrefix(k, "person_") || strings.HasPrefix(k, "representative.")) && strings.HasSuffix(k, suffix) && v != "" {
			return v
		}
	}
	return ""
}

// getExtraSuffix finds the first extra_fields value whose key ends with the given suffix.
func getExtraSuffix(extra map[string]string, suffix string) string {
	for k, v := range extra {
		if strings.HasSuffix(k, suffix) && v != "" {
			return v
		}
	}
	return ""
}

// getExtra looks up a value in extra_fields by multiple possible keys.
func getExtra(extra map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := extra[k]; ok && v != "" {
			return v
		}
	}
	return ""
}

// createStripePersons creates the required Stripe persons for a company account.
func (s *Service) createStripePersons(ctx context.Context, info *domain.PaymentInfo, accountID, email string) {
	// Representative person (always required for company)
	repInput := portservice.CreatePersonInput{
		FirstName:        info.FirstName,
		LastName:         info.LastName,
		Email:            email,
		Phone:            info.Phone,
		DOB:              info.DateOfBirth,
		Address:          info.Address,
		City:             info.City,
		PostalCode:       info.PostalCode,
		State:            getExtra(info.ExtraFields, "representative.address.state", "state"),
		Country:          info.Country,
		Title:            firstNonEmpty(info.RoleInCompany, getExtraSuffix(info.ExtraFields, "relationship.title")),
		IDNumber:         getExtra(info.ExtraFields, "representative.id_number", "individual.id_number", "id_number"),
		SSNLast4:         getExtra(info.ExtraFields, "representative.ssn_last_4", "individual.ssn_last_4", "ssn_last_4"),
		IsRepresentative: true,
		IsDirector:       info.IsSelfDirector,
		IsExecutive:      info.IsSelfExecutive,
		IsOwner:          !info.NoMajorOwners && info.IsSelfRepresentative,
	}

	if _, err := s.stripe.CreatePerson(ctx, accountID, repInput); err != nil {
		slog.Error("failed to create representative person", "error", err)
	}

	// Additional persons from business_persons table
	persons, _ := s.persons.ListByUserID(ctx, info.UserID)
	for _, p := range persons {
		input := portservice.CreatePersonInput{
			FirstName: p.FirstName,
			LastName:  p.LastName,
			Email:     p.Email,
			Phone:     p.Phone,
			DOB:       p.DateOfBirth,
			Address:   p.Address,
			City:      p.City,
			PostalCode: p.PostalCode,
			Title:     p.Title,
		}
		switch p.Role {
		case domain.RoleDirector:
			input.IsDirector = true
		case domain.RoleOwner:
			input.IsOwner = true
		case domain.RoleExecutive:
			input.IsExecutive = true
		}

		personID, err := s.stripe.CreatePerson(ctx, accountID, input)
		if err != nil {
			slog.Error("failed to create business person", "role", p.Role, "error", err)
			continue
		}
		p.StripePersonID = personID
	}

	// Mark all provided
	if err := s.stripe.UpdateCompanyFlags(ctx, accountID, true, true, true); err != nil {
		slog.Error("failed to update company flags", "error", err)
	}
}

// syncPendingDocuments uploads pending documents to Stripe after account creation.
func (s *Service) syncPendingDocuments(ctx context.Context, userID uuid.UUID, accountID string) {
	docs, err := s.documents.ListByUserID(ctx, userID)
	if err != nil || len(docs) == 0 {
		return
	}

	for _, d := range docs {
		if d.StripeFileID != "" {
			continue
		}
		fileURL := s.storage.GetPublicURL(d.FileKey)
		resp, httpErr := http.Get(fileURL)
		if httpErr != nil || resp.StatusCode != 200 {
			slog.Error("sync: failed to download from R2", "doc_id", d.ID, "url", fileURL)
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		fileData, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			continue
		}
		stripeFileID, uploadErr := s.stripe.UploadIdentityFile(ctx, d.FileKey, bytes.NewReader(fileData), "identity_document")
		if uploadErr != nil {
			slog.Error("sync: failed to upload to stripe", "doc_id", d.ID, "error", uploadErr)
			continue
		}
		_ = s.documents.UpdateStripeFileID(ctx, d.ID, stripeFileID)
		slog.Info("sync: document uploaded to stripe", "doc_id", d.ID, "stripe_file_id", stripeFileID)
	}

	s.attachDocumentToAccount(ctx, userID, accountID)
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
