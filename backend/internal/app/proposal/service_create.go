package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/port/service"
)

func (s *Service) CreateProposal(ctx context.Context, input CreateProposalInput) (*domain.Proposal, error) {
	if input.SenderID == input.RecipientID {
		return nil, domain.ErrSameUser
	}

	sender, err := s.users.GetByID(ctx, input.SenderID)
	if err != nil {
		return nil, fmt.Errorf("get sender: %w", err)
	}

	recipient, err := s.users.GetByID(ctx, input.RecipientID)
	if err != nil {
		return nil, fmt.Errorf("get recipient: %w", err)
	}

	clientID, providerID, err := domain.DetermineRoles(
		input.SenderID, string(sender.Role),
		input.RecipientID, string(recipient.Role),
	)
	if err != nil {
		return nil, err
	}

	p, err := domain.NewProposal(domain.NewProposalInput{
		ConversationID: input.ConversationID,
		SenderID:       input.SenderID,
		RecipientID:    input.RecipientID,
		Title:          input.Title,
		Description:    input.Description,
		Amount:         input.Amount,
		Deadline:       input.Deadline,
		ClientID:       clientID,
		ProviderID:     providerID,
		Version:        1,
	})
	if err != nil {
		return nil, err
	}

	docs := buildDocuments(p.ID, input.Documents)

	if err := s.proposals.CreateWithDocuments(ctx, p, docs); err != nil {
		return nil, fmt.Errorf("persist proposal: %w", err)
	}

	metadata := buildProposalMetadata(p, sender.DisplayName, len(docs))
	s.sendProposalMessage(ctx, p.ConversationID, p.SenderID, "proposal_sent", metadata)

	s.sendNotification(ctx, input.RecipientID, "proposal_received", "New proposal",
		sender.DisplayName+" sent you a proposal",
		buildNotificationData(p.ID, p.ConversationID, p.Title))

	return p, nil
}

func buildDocuments(proposalID uuid.UUID, inputs []DocumentInput) []*domain.ProposalDocument {
	docs := make([]*domain.ProposalDocument, len(inputs))
	now := time.Now()
	for i, d := range inputs {
		docs[i] = &domain.ProposalDocument{
			ID:         uuid.New(),
			ProposalID: proposalID,
			Filename:   d.Filename,
			URL:        d.URL,
			Size:       d.Size,
			MimeType:   d.MimeType,
			CreatedAt:  now,
		}
	}
	return docs
}

func buildProposalMetadata(p *domain.Proposal, senderName string, docsCount int) json.RawMessage {
	m := map[string]any{
		"proposal_id":              p.ID.String(),
		"proposal_title":           p.Title,
		"proposal_amount":          p.Amount,
		"proposal_status":          string(p.Status),
		"proposal_documents_count": docsCount,
		"proposal_sender_name":     senderName,
		"proposal_version":         p.Version,
		"proposal_client_id":       p.ClientID.String(),
		"proposal_provider_id":     p.ProviderID.String(),
	}
	if p.ParentID != nil {
		m["proposal_parent_id"] = p.ParentID.String()
	} else {
		m["proposal_parent_id"] = nil
	}
	if p.Deadline != nil {
		m["proposal_deadline"] = p.Deadline.Format(time.RFC3339)
	}
	data, _ := json.Marshal(m)
	return data
}

func (s *Service) sendProposalMessage(ctx context.Context, convID, senderID uuid.UUID, msgType string, metadata json.RawMessage) {
	_ = s.messages.SendSystemMessage(ctx, service.SystemMessageInput{
		ConversationID: convID,
		SenderID:       senderID,
		Content:        "",
		Type:           msgType,
		Metadata:       metadata,
	})
}

func (s *Service) sendNotification(ctx context.Context, userID uuid.UUID, notifType, title, body string, data json.RawMessage) {
	if s.notifications == nil {
		return
	}
	_ = s.notifications.Send(ctx, service.NotificationInput{
		UserID: userID,
		Type:   notifType,
		Title:  title,
		Body:   body,
		Data:   data,
	})
}

func buildNotificationData(proposalID, conversationID uuid.UUID, proposalTitle string) json.RawMessage {
	data, _ := json.Marshal(map[string]string{
		"proposal_id":     proposalID.String(),
		"conversation_id": conversationID.String(),
		"proposal_title":  proposalTitle,
	})
	return data
}

func (s *Service) resolveUserName(ctx context.Context, userID uuid.UUID) string {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return "Someone"
	}
	return u.DisplayName
}
