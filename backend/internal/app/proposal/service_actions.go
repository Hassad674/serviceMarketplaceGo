package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

func (s *Service) AcceptProposal(ctx context.Context, input AcceptProposalInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	// Org-level authorization: the caller must belong to the recipient's
	// organization. Only the recipient side can accept a proposal —
	// directionality is preserved even though the user-level check has
	// been replaced with an org-level one (any operator of the recipient
	// org can accept on behalf of the team).
	if err := s.requireOrgIsSide(ctx, p.RecipientID, input.OrgID, domain.ErrNotAuthorized); err != nil {
		return err
	}

	// KYC enforcement: if the acceptor's org is blocked (14 days
	// elapsed without Stripe onboarding), they cannot accept proposals.
	if s.orgs != nil {
		if org, oErr := s.orgs.FindByUserID(ctx, input.UserID); oErr == nil && org.IsKYCBlocked() {
			return user.ErrKYCRestricted
		}
	}

	// Pass the canonical recipient id to the domain method so its own
	// user-level invariant still holds. The real authorization has
	// already been performed at the org level above.
	if err := p.Accept(p.RecipientID); err != nil {
		return err
	}

	if err := s.proposals.Update(ctx, p); err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_accepted", metadata)

	// If the recipient side is the provider, send a payment request to
	// the client side. We key on the proposal's own ProviderID — not on
	// the acting operator — so the message is posted whenever the
	// provider org accepted, regardless of which team member clicked.
	if p.RecipientID == p.ProviderID {
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

	// Recipient-only directional check at org granularity.
	if err := s.requireOrgIsSide(ctx, p.RecipientID, input.OrgID, domain.ErrNotAuthorized); err != nil {
		return err
	}

	if err := p.Decline(p.RecipientID); err != nil {
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

	// Only the recipient SIDE can counter — any operator of the
	// recipient's org can create the counter-version. Status must still
	// be pending, which CanBeModifiedBy enforces via p.RecipientID.
	if err := s.requireOrgIsSide(ctx, original.RecipientID, input.OrgID, domain.ErrCannotModify); err != nil {
		return nil, err
	}
	if !original.CanBeModifiedBy(original.RecipientID) {
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

	// Payment is a strictly CLIENT-side action. The caller must belong
	// to the client's organization. The recipient side (provider org)
	// must NOT be able to pay on behalf of the client, even though they
	// are a party to the proposal — this directional check is the only
	// thing standing between a buggy operator and double-charging.
	if err := s.requireOrgIsSide(ctx, p.ClientID, input.OrgID, domain.ErrNotAuthorized); err != nil {
		return nil, err
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

	// Award bonus credits with fraud detection
	s.awardBonusWithFraudCheck(ctx, p)

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

	// Award bonus credits with fraud detection
	s.awardBonusWithFraudCheck(ctx, p)

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

// AuthorizeClientOrg returns nil if the caller's org owns the client
// side of the proposal, or domain.ErrNotAuthorized otherwise. Used by
// the ConfirmPayment handler, which still executes several payment
// service calls in sequence after authorization — hence this method
// is exposed as a standalone gate instead of being bundled with the
// status transition.
func (s *Service) AuthorizeClientOrg(ctx context.Context, proposalID, orgID uuid.UUID) error {
	p, err := s.proposals.GetByID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}
	return s.requireOrgIsSide(ctx, p.ClientID, orgID, domain.ErrNotAuthorized)
}

func (s *Service) RequestCompletion(ctx context.Context, input RequestCompletionInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	// Provider-only directional check at org granularity.
	if err := s.requireOrgIsSide(ctx, p.ProviderID, input.OrgID, domain.ErrNotProvider); err != nil {
		return err
	}

	if err := p.RequestCompletion(p.ProviderID); err != nil {
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

	// Client-only directional check at org granularity — only the
	// client side can confirm that the mission is done (because they
	// release the escrowed funds).
	if err := s.requireOrgIsSide(ctx, p.ClientID, input.OrgID, domain.ErrNotClient); err != nil {
		return err
	}

	if err := p.ConfirmCompletion(p.ClientID); err != nil {
		return err
	}

	if err := s.proposals.Update(ctx, p); err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	metadata := s.buildCompletedMetadata(ctx, p)
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_completed", metadata)

	// Double-blind reviews (since phase R18): a SINGLE evaluation_request
	// system message is posted in the shared conversation. Both parties
	// see it and can click the CTA — the frontend derives the viewer's
	// side from the client/provider organization ids carried in metadata
	// and opens the right review variant. Neither side will see the
	// other's review until both have submitted or 14 days have elapsed.
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "evaluation_request", metadata)

	s.sendNotification(ctx, p.ProviderID, "proposal_completed", "Mission completed",
		"Your mission has been marked as complete. Leave a review for the client before the 14-day window closes.",
		buildNotificationData(p.ID, p.ConversationID, p.Title))
	s.sendNotification(ctx, p.ClientID, "proposal_completed", "Mission completed",
		"The mission is marked as complete. Leave a review for the provider before the 14-day window closes.",
		buildNotificationData(p.ID, p.ConversationID, p.Title))

	// Transfer funds to provider (non-blocking — log errors but don't fail completion)
	if s.payments != nil {
		if err := s.payments.TransferToProvider(ctx, p.ID); err != nil {
			slog.Error("failed to transfer to provider", "proposal_id", p.ID, "error", err)
		}
	}

	// Record first earning for KYC enforcement — triggers the 14-day
	// countdown on the provider's organization (the merchant of record
	// since phase R5). Idempotent: only writes when the org row still
	// has a NULL kyc_first_earning_at.
	if s.orgs != nil && s.users != nil {
		providerUser, lookupErr := s.users.GetByID(ctx, p.ProviderID)
		if lookupErr == nil && providerUser.OrganizationID != nil {
			if err := s.orgs.SetKYCFirstEarning(ctx, *providerUser.OrganizationID, time.Now()); err != nil {
				slog.Warn("kyc: failed to record first earning",
					"provider_id", p.ProviderID,
					"org_id", providerUser.OrganizationID,
					"error", err)
			}
		}
	}

	return nil
}

func (s *Service) RejectCompletion(ctx context.Context, input RejectCompletionInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	// Client-only directional check at org granularity.
	if err := s.requireOrgIsSide(ctx, p.ClientID, input.OrgID, domain.ErrNotClient); err != nil {
		return err
	}

	if err := p.RejectCompletion(p.ClientID); err != nil {
		return err
	}

	if err := s.proposals.Update(ctx, p); err != nil {
		return fmt.Errorf("update proposal: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_completion_rejected", metadata)

	return nil
}

// GetProposal fetches a proposal along with its documents and verifies
// that the calling organization (not just the user) has access. Either
// the client-side org or the provider-side org is authorized — any
// operator inside those orgs can read the proposal. userID is kept in
// the signature for audit/logging consistency but is no longer used
// for the authorization check itself.
func (s *Service) GetProposal(ctx context.Context, userID, orgID, proposalID uuid.UUID) (*domain.Proposal, []*domain.ProposalDocument, error) {
	_ = userID // reserved for future audit logging; auth now uses orgID
	p, err := s.proposals.GetByID(ctx, proposalID)
	if err != nil {
		return nil, nil, fmt.Errorf("get proposal: %w", err)
	}

	authorized, err := s.proposals.IsOrgAuthorizedForProposal(ctx, proposalID, orgID)
	if err != nil {
		return nil, nil, fmt.Errorf("authorize proposal for org: %w", err)
	}
	if !authorized {
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

// ListActiveProjectsByOrganization returns the non-completed/active
// proposals where the caller's organization is either side.
func (s *Service) ListActiveProjectsByOrganization(ctx context.Context, orgID uuid.UUID, cursorStr string, limit int) ([]*domain.Proposal, string, error) {
	return s.proposals.ListActiveProjectsByOrganization(ctx, orgID, cursorStr, limit)
}

// buildCompletedMetadata returns the status metadata for a completed
// proposal, enriched with the client and provider ORGANIZATION ids.
//
// The base metadata only carries the user-level ClientID/ProviderID
// because that's how the proposal entity has stored participants since
// day one. Frontends running on the post-phase-4 team/org model need
// the organization ids to derive which side of a double-blind review
// the current viewer is on — comparing org id vs org id, so that any
// operator in the team can legitimately review on behalf of their org.
//
// We resolve both orgs via the user repository and add
// proposal_client_organization_id + proposal_provider_organization_id
// to the metadata. If a lookup fails (missing user, user without org)
// the field is simply omitted — the frontend falls back to hiding the
// CTA for that viewer, which is the safe default.
func (s *Service) buildCompletedMetadata(ctx context.Context, p *domain.Proposal) json.RawMessage {
	base := buildStatusMetadata(p)
	if s.users == nil {
		return base
	}

	var m map[string]any
	if err := json.Unmarshal(base, &m); err != nil || m == nil {
		return base
	}

	if clientUser, err := s.users.GetByID(ctx, p.ClientID); err == nil && clientUser.OrganizationID != nil {
		m["proposal_client_organization_id"] = clientUser.OrganizationID.String()
	}
	if providerUser, err := s.users.GetByID(ctx, p.ProviderID); err == nil && providerUser.OrganizationID != nil {
		m["proposal_provider_organization_id"] = providerUser.OrganizationID.String()
	}

	enriched, err := json.Marshal(m)
	if err != nil {
		return base
	}
	return enriched
}

// requireOrgIsSide resolves the user identified by sideUserID (which
// represents one specific directional side of the proposal — sender,
// recipient, client, or provider) and checks whether their
// organization matches the caller's org. Returns notAllowedErr if the
// side is not associated with the caller's org, preserving whichever
// sentinel error the calling method wants to surface (ErrNotAuthorized,
// ErrCannotModify, ErrNotClient, ErrNotProvider).
//
// This is how directional checks ("only the recipient side can
// accept", "only the client side can pay") are enforced at the org
// level while still letting any operator within the winning org act
// on behalf of the team.
func (s *Service) requireOrgIsSide(
	ctx context.Context,
	sideUserID uuid.UUID,
	callerOrgID uuid.UUID,
	notAllowedErr error,
) error {
	if s.users == nil {
		return notAllowedErr
	}
	sideUser, err := s.users.GetByID(ctx, sideUserID)
	if err != nil {
		return fmt.Errorf("resolve side user: %w", err)
	}
	if sideUser.OrganizationID == nil || *sideUser.OrganizationID != callerOrgID {
		return notAllowedErr
	}
	return nil
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
