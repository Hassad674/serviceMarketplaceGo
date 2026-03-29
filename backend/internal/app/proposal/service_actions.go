package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/port/service"
)

func (s *Service) AcceptProposal(ctx context.Context, input AcceptProposalInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	if err := p.Accept(input.UserID); err != nil {
		return err
	}

	if err := s.proposals.Update(ctx, p); err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_accepted", metadata)

	// If the acceptor is the provider, send a payment request to the client
	if input.UserID == p.ProviderID {
		s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_payment_requested", metadata)
	}

	s.sendNotification(ctx, p.SenderID, "proposal_accepted", "Proposal accepted",
		"Your proposal has been accepted",
		buildNotificationData(p.ID, p.ConversationID, p.Title))

	return nil
}

func (s *Service) DeclineProposal(ctx context.Context, input DeclineProposalInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	if err := p.Decline(input.UserID); err != nil {
		return err
	}

	if err := s.proposals.Update(ctx, p); err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_declined", metadata)

	s.sendNotification(ctx, p.SenderID, "proposal_declined", "Proposal declined",
		"Your proposal has been declined",
		buildNotificationData(p.ID, p.ConversationID, p.Title))

	return nil
}

func (s *Service) ModifyProposal(ctx context.Context, input ModifyProposalInput) (*domain.Proposal, error) {
	original, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("get proposal: %w", err)
	}

	if !original.CanBeModifiedBy(input.UserID) {
		return nil, domain.ErrCannotModify
	}

	// Determine root proposal ID for version chain
	rootID := original.ID
	if original.ParentID != nil {
		rootID = *original.ParentID
	}

	// The modifier becomes the sender of the new version
	modified, err := domain.NewProposal(domain.NewProposalInput{
		ConversationID: original.ConversationID,
		SenderID:       input.UserID,
		RecipientID:    original.SenderID,
		Title:          input.Title,
		Description:    input.Description,
		Amount:         input.Amount,
		Deadline:       input.Deadline,
		ClientID:       original.ClientID,
		ProviderID:     original.ProviderID,
		ParentID:       &rootID,
		Version:        original.Version + 1,
	})
	if err != nil {
		return nil, err
	}

	docs := buildDocuments(modified.ID, input.Documents)

	if err := s.proposals.CreateWithDocuments(ctx, modified, docs); err != nil {
		return nil, fmt.Errorf("persist modified proposal: %w", err)
	}

	metadata := buildStatusMetadata(modified)
	s.sendProposalMessage(ctx, modified.ConversationID, input.UserID, "proposal_modified", metadata)

	// Notify the other party (the original sender receives the modification notice)
	modifierName := s.resolveUserName(ctx, input.UserID)
	s.sendNotification(ctx, original.SenderID, "proposal_modified", "Proposal modified",
		modifierName+" modified the proposal",
		buildNotificationData(modified.ID, modified.ConversationID, modified.Title))

	return modified, nil
}

// InitiatePayment creates a Stripe PaymentIntent or falls back to simulation.
// Returns nil output when simulation mode completes the payment immediately.
func (s *Service) InitiatePayment(ctx context.Context, input PayProposalInput) (*service.PaymentIntentOutput, error) {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("get proposal: %w", err)
	}

	if input.UserID != p.ClientID {
		return nil, domain.ErrNotAuthorized
	}

	if p.Status != domain.StatusAccepted {
		return nil, domain.ErrInvalidStatus
	}

	// Real Stripe mode
	if s.payments != nil {
		result, err := s.payments.CreatePaymentIntent(ctx, service.PaymentIntentInput{
			ProposalID:     p.ID,
			ClientID:       p.ClientID,
			ProviderID:     p.ProviderID,
			ProposalAmount: p.Amount,
		})
		if err != nil {
			return nil, fmt.Errorf("create payment intent: %w", err)
		}
		return result, nil
	}

	// Simulation fallback (dev mode)
	return nil, s.simulatePayment(ctx, p, input.UserID)
}

// simulatePayment immediately marks the proposal as paid+active (dev mode only).
func (s *Service) simulatePayment(ctx context.Context, p *domain.Proposal, userID uuid.UUID) error {
	if err := p.MarkPaid(); err != nil {
		return err
	}
	if err := p.MarkActive(); err != nil {
		return err
	}
	if err := s.proposals.Update(ctx, p); err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, userID, "proposal_paid", metadata)
	s.sendNotification(ctx, p.ProviderID, "proposal_paid", "Payment received",
		"A payment has been made for your proposal",
		buildNotificationData(p.ID, p.ConversationID, p.Title))
	return nil
}

// ConfirmPaymentAndActivate is called by the webhook handler after Stripe confirms payment.
func (s *Service) ConfirmPaymentAndActivate(ctx context.Context, proposalID uuid.UUID) error {
	p, err := s.proposals.GetByID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	// Idempotency: already paid/active
	if p.Status == domain.StatusPaid || p.Status == domain.StatusActive {
		return nil
	}

	if err := p.MarkPaid(); err != nil {
		return err
	}
	if err := p.MarkActive(); err != nil {
		return err
	}
	if err := s.proposals.Update(ctx, p); err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, p.ClientID, "proposal_paid", metadata)
	s.sendNotification(ctx, p.ProviderID, "proposal_paid", "Payment received",
		"A payment has been made for your proposal",
		buildNotificationData(p.ID, p.ConversationID, p.Title))
	return nil
}

// GetProposalByID returns a proposal without authorization checks.
// Used by the handler for ownership verification.
func (s *Service) GetProposalByID(ctx context.Context, id uuid.UUID) (*domain.Proposal, error) {
	return s.proposals.GetByID(ctx, id)
}

func (s *Service) RequestCompletion(ctx context.Context, input RequestCompletionInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	if err := p.RequestCompletion(input.UserID); err != nil {
		return err
	}

	if err := s.proposals.Update(ctx, p); err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_completion_requested", metadata)

	s.sendNotification(ctx, p.ClientID, "completion_requested", "Completion requested",
		"The provider has requested to mark the mission as complete",
		buildNotificationData(p.ID, p.ConversationID, p.Title))

	return nil
}

func (s *Service) CompleteProposal(ctx context.Context, input CompleteProposalInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	if err := p.ConfirmCompletion(input.UserID); err != nil {
		return err
	}

	if err := s.proposals.Update(ctx, p); err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_completed", metadata)

	// Send evaluation_request only to the client (the party who pays).
	// The provider never evaluates the client.
	// Enterprise evaluates Freelance/Agency, Agency evaluates Freelance.
	s.sendEvaluationRequest(ctx, p.ConversationID, p.ClientID, metadata)

	s.sendNotification(ctx, p.ProviderID, "proposal_completed", "Mission completed",
		"Your mission has been marked as complete",
		buildNotificationData(p.ID, p.ConversationID, p.Title))

	// Transfer funds to provider (non-blocking — log errors but don't fail completion)
	if s.payments != nil {
		if err := s.payments.TransferToProvider(ctx, p.ID); err != nil {
			slog.Error("failed to transfer to provider", "proposal_id", p.ID, "error", err)
		}
	}

	return nil
}

func (s *Service) RejectCompletion(ctx context.Context, input RejectCompletionInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	if err := p.RejectCompletion(input.UserID); err != nil {
		return err
	}

	if err := s.proposals.Update(ctx, p); err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_completion_rejected", metadata)

	return nil
}

func (s *Service) GetProposal(ctx context.Context, userID, proposalID uuid.UUID) (*domain.Proposal, []*domain.ProposalDocument, error) {
	p, err := s.proposals.GetByID(ctx, proposalID)
	if err != nil {
		return nil, nil, fmt.Errorf("get proposal: %w", err)
	}

	if !isParticipant(p, userID) {
		return nil, nil, domain.ErrNotAuthorized
	}

	docs, err := s.proposals.GetDocuments(ctx, proposalID)
	if err != nil {
		return nil, nil, fmt.Errorf("get documents: %w", err)
	}

	return p, docs, nil
}

func (s *Service) GetParticipantNames(ctx context.Context, clientID, providerID uuid.UUID) (clientName, providerName string) {
	if c, err := s.users.GetByID(ctx, clientID); err == nil {
		clientName = c.DisplayName
	}
	if p, err := s.users.GetByID(ctx, providerID); err == nil {
		providerName = p.DisplayName
	}
	return clientName, providerName
}

func (s *Service) ListActiveProjects(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*domain.Proposal, string, error) {
	return s.proposals.ListActiveProjects(ctx, userID, cursorStr, limit)
}

// sendEvaluationRequest sends an evaluation_request system message enriched
// with target_user_id so the frontend only renders it for the client.
func (s *Service) sendEvaluationRequest(ctx context.Context, convID, clientID uuid.UUID, baseMetadata json.RawMessage) {
	// Enrich metadata with target_user_id so frontends can filter visibility.
	var m map[string]any
	_ = json.Unmarshal(baseMetadata, &m)
	m["target_user_id"] = clientID.String()
	enriched, _ := json.Marshal(m)

	s.sendProposalMessage(ctx, convID, clientID, "evaluation_request", enriched)
}

func isParticipant(p *domain.Proposal, userID uuid.UUID) bool {
	return userID == p.SenderID ||
		userID == p.RecipientID ||
		userID == p.ClientID ||
		userID == p.ProviderID
}

func buildStatusMetadata(p *domain.Proposal) json.RawMessage {
	m := map[string]any{
		"proposal_id":              p.ID.String(),
		"proposal_title":           p.Title,
		"proposal_amount":          p.Amount,
		"proposal_status":          string(p.Status),
		"proposal_version":         p.Version,
		"proposal_client_id":       p.ClientID.String(),
		"proposal_provider_id":     p.ProviderID.String(),
		"proposal_sender_name":     "",
		"proposal_documents_count": 0,
	}
	if p.ParentID != nil {
		m["proposal_parent_id"] = p.ParentID.String()
	} else {
		m["proposal_parent_id"] = nil
	}
	if p.Deadline != nil {
		m["proposal_deadline"] = p.Deadline.Format(time.RFC3339)
	} else {
		m["proposal_deadline"] = nil
	}
	data, _ := json.Marshal(m)
	return data
}
