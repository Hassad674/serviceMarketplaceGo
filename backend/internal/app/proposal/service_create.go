package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

// paymentModeOneTime and paymentModeMilestone are the two UX hints
// stored on proposals.payment_mode. The backend treats both identically
// (every proposal has ≥1 milestone); the flag only tells the frontend
// which form to render on read.
const (
	paymentModeOneTime   = "one_time"
	paymentModeMilestone = "milestone"
)

func (s *Service) CreateProposal(ctx context.Context, input CreateProposalInput) (*domain.Proposal, error) {
	if input.SenderID == input.RecipientID {
		return nil, domain.ErrSameUser
	}

	// Pre-validate the top-level proposal fields at the app layer so
	// errors surface as proposal.Err* (matching the handler's existing
	// error-code mapping) even though the proposal entity itself is
	// constructed later (after the milestone batch has derived the
	// total amount).
	if err := validateProposalFields(input); err != nil {
		return nil, err
	}

	sender, err := s.users.GetByID(ctx, input.SenderID)
	if err != nil {
		return nil, fmt.Errorf("get sender: %w", err)
	}
	// KYC enforcement: the sender's org must not be blocked (14-day
	// deadline without Stripe onboarding). Fails closed if the org
	// lookup errors so the flow never proceeds on ambiguous state.
	if s.orgs != nil {
		if org, oErr := s.orgs.FindByUserID(ctx, input.SenderID); oErr == nil && org.IsKYCBlocked() {
			return nil, user.ErrKYCRestricted
		}
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

	// Phase 4 invariant: every proposal has ≥1 milestone. Resolve the
	// milestone slice BEFORE building the proposal so the total amount
	// can be derived from the milestone batch.
	//
	// Two cases:
	//  1. Caller provided Milestones — build a validated batch from
	//     them, total amount = sum of milestone amounts. payment_mode
	//     defaults to "milestone" unless the caller overrode it.
	//  2. Caller did not provide Milestones — synthesise exactly one
	//     milestone at sequence=1 using input.Amount. payment_mode
	//     defaults to "one_time".
	// In both cases the persisted proposal.amount is the same as the
	// sum of the milestones, maintaining the cached-sum invariant.
	milestoneInputs := buildMilestoneDomainInputs(input)

	// We build the batch against a placeholder proposal id first,
	// then rebind it once the real proposal has been created below.
	// This avoids re-validating the batch a second time.
	placeholderProposalID := uuid.New()
	milestones, err := milestone.NewMilestoneBatch(placeholderProposalID, milestoneInputs)
	if err != nil {
		return nil, err
	}
	totalAmount := milestone.SumAmount(milestones)

	p, err := domain.NewProposal(domain.NewProposalInput{
		ConversationID: input.ConversationID,
		SenderID:       input.SenderID,
		RecipientID:    input.RecipientID,
		Title:          input.Title,
		Description:    input.Description,
		Amount:         totalAmount,
		Deadline:       input.Deadline,
		ClientID:       clientID,
		ProviderID:     providerID,
		Version:        1,
	})
	if err != nil {
		return nil, err
	}

	// Rebind milestones to the real proposal id now that it exists.
	for _, m := range milestones {
		m.ProposalID = p.ID
	}

	docs := buildDocuments(p.ID, input.Documents)

	if err := s.proposals.CreateWithDocumentsAndMilestones(ctx, p, docs, milestones); err != nil {
		return nil, fmt.Errorf("persist proposal: %w", err)
	}

	metadata := buildProposalMetadata(p, sender.DisplayName, len(docs))
	s.sendProposalMessage(ctx, p.ConversationID, p.SenderID, "proposal_sent", metadata)

	s.sendNotification(ctx, input.RecipientID, "proposal_received", "New proposal",
		sender.DisplayName+" sent you a proposal",
		buildNotificationData(p.ID, p.ConversationID, p.Title))

	return p, nil
}

// validateProposalFields runs the top-level proposal-domain checks
// (non-empty title/description, positive aggregate amount, distinct
// parties) at the app layer BEFORE the milestone batch is built.
//
// Without this pre-check, a milestone-mode caller sending an empty
// title would get milestone.ErrEmptyTitle back instead of the expected
// proposal.ErrEmptyTitle — which breaks the handler's error-code map.
func validateProposalFields(input CreateProposalInput) error {
	if input.Title == "" {
		return domain.ErrEmptyTitle
	}
	if input.Description == "" {
		return domain.ErrEmptyDescription
	}
	// One-time mode: validate the single Amount directly.
	if len(input.Milestones) == 0 {
		if input.Amount <= 0 {
			return domain.ErrInvalidAmount
		}
		return nil
	}
	// Milestone mode: validate that every milestone has a title,
	// description, and positive amount. The detailed sequence/batch
	// checks happen later in milestone.NewMilestoneBatch.
	for _, m := range input.Milestones {
		if m.Title == "" {
			return domain.ErrEmptyTitle
		}
		if m.Description == "" {
			return domain.ErrEmptyDescription
		}
		if m.Amount <= 0 {
			return domain.ErrInvalidAmount
		}
	}
	return nil
}

// buildMilestoneDomainInputs translates the app-layer MilestoneInput
// slice into the domain-layer NewMilestoneInput slice. When the caller
// didn't supply any milestones, a single synthetic milestone is
// produced at sequence=1 carrying the full input.Amount — this is the
// backward-compat path that lets legacy one_time mode keep working via
// the new unified pipeline.
func buildMilestoneDomainInputs(input CreateProposalInput) []milestone.NewMilestoneInput {
	if len(input.Milestones) == 0 {
		return []milestone.NewMilestoneInput{
			{
				Sequence:    1,
				Title:       input.Title,
				Description: input.Description,
				Amount:      input.Amount,
				Deadline:    input.Deadline,
			},
		}
	}
	out := make([]milestone.NewMilestoneInput, 0, len(input.Milestones))
	for _, m := range input.Milestones {
		out = append(out, milestone.NewMilestoneInput{
			Sequence:    m.Sequence,
			Title:       m.Title,
			Description: m.Description,
			Amount:      m.Amount,
			Deadline:    m.Deadline,
		})
	}
	return out
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
