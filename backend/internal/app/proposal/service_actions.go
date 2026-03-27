package proposal

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/proposal"
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

	return modified, nil
}

func (s *Service) SimulatePayment(ctx context.Context, input PayProposalInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	if input.UserID != p.ClientID {
		return domain.ErrNotAuthorized
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
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_paid", metadata)

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

func (s *Service) ListActiveProjects(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*domain.Proposal, string, error) {
	return s.proposals.ListActiveProjects(ctx, userID, cursorStr, limit)
}

func isParticipant(p *domain.Proposal, userID uuid.UUID) bool {
	return userID == p.SenderID ||
		userID == p.RecipientID ||
		userID == p.ClientID ||
		userID == p.ProviderID
}

func buildStatusMetadata(p *domain.Proposal) json.RawMessage {
	m := map[string]any{
		"proposal_id": p.ID.String(),
		"title":       p.Title,
		"amount":      p.Amount,
		"status":      string(p.Status),
	}
	data, _ := json.Marshal(m)
	return data
}
