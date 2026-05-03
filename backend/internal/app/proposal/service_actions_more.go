package proposal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
	domain "marketplace-backend/internal/domain/proposal"
)

// CompleteProposal approves AND releases the proposal's current active
// milestone in a single locked transition. If the released milestone
// was the LAST one of the proposal, the macro status transitions to
// completed — at which point the credit-bonus fraud check runs and
// the KYC first-earning timestamp is recorded (user decisions F4/F5).
// Otherwise the proposal drops back to "active" (the next milestone
// now becomes the current one, awaiting funding).
func (s *Service) CompleteProposal(ctx context.Context, input CompleteProposalInput) error {
	p, err := s.proposals.GetByIDForOrg(ctx, input.ProposalID, input.OrgID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	// Client-only directional check at org granularity — only the
	// client side can confirm that the mission is done (because they
	// release the escrowed funds).
	if err := s.requireOrgIsSide(ctx, p.ClientID, input.OrgID, domain.ErrNotClient); err != nil {
		return err
	}

	current, err := s.milestones.GetCurrentActive(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("get current milestone: %w", err)
	}
	if current.Status != milestone.StatusSubmitted {
		return domain.ErrInvalidStatus
	}

	// KYC readiness probe — informational only. We do NOT block the
	// approve when payouts aren't enabled (blocking made the dev/test
	// workflow unusable, every fresh provider lacks KYC, and even in
	// prod a held-up client UX is worse than a delayed transfer the
	// provider can resolve by completing onboarding). The probe just
	// adjusts the notification copy: "paid" when truly ready, "held
	// pending Stripe onboarding" when the provider hasn't finished
	// KYC yet. On a probe error we stay optimistic — the actual
	// TransferMilestone call below is still authoritative.
	providerReadyForPayouts := true
	if ready, kerr := s.providerCanReceivePayouts(ctx, p.ProviderID); kerr != nil {
		slog.Warn("approve milestone: provider payouts probe failed, continuing optimistically",
			"proposal_id", p.ID, "provider_id", p.ProviderID, "error", kerr)
	} else {
		providerReadyForPayouts = ready
	}

	if err := s.withMilestoneLock(ctx, current.ID, func(m *milestone.Milestone) error {
		if err := m.Approve(); err != nil {
			return err
		}
		return m.Release()
	}); err != nil {
		return fmt.Errorf("approve+release milestone: %w", err)
	}

	if err := s.recomputeMacroStatus(ctx, p); err != nil {
		return fmt.Errorf("recompute macro status: %w", err)
	}

	// If the macro status is now completed, this was the LAST milestone
	// of the proposal: run the end-of-project side effects (shared with
	// AutoApproveMilestone via runEndOfProjectEffects).
	// Otherwise we're mid-project: emit the "milestone released" signal
	// AND a "payment requested" prompt for the next milestone so the
	// client gets a clickable CTA in the conversation (the same one
	// they saw when the proposal was first accepted), then schedule the
	// fund-reminder + auto-close timers so a ghosting client triggers a
	// graceful auto-close.
	if p.Status == domain.StatusCompleted {
		// Same eligibility rule as the mid-project branch: auto-transfer
		// the freshly-released milestone iff the provider has consent.
		if s.providerEligibleForAutoTransfer(ctx, p.ProviderID) {
			if err := s.payments.TransferMilestone(ctx, current.ID); err != nil {
				slog.Error("end-of-project auto-transfer failed; record stays TransferPending for manual retry",
					"proposal_id", p.ID, "milestone_id", current.ID, "error", err)
			}
		}
		s.runEndOfProjectEffects(ctx, p)
	} else {
		// Mid-project release: auto-transfer ONLY when the provider has
		// previously completed a manual payout (consent stamp). New
		// providers stay TransferPending so they pull funds explicitly
		// from the wallet on the first time — that click serves as the
		// "Stripe payouts actually work for me" proof and flips the
		// consent flag for every subsequent release.
		if s.providerEligibleForAutoTransfer(ctx, p.ProviderID) {
			if err := s.payments.TransferMilestone(ctx, current.ID); err != nil {
				slog.Error("mid-project auto-transfer failed; record stays TransferPending for manual retry",
					"proposal_id", p.ID, "milestone_id", current.ID, "error", err)
			}
		}
		metadata := buildStatusMetadata(p)
		s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "milestone_released", metadata)
		s.sendNotification(ctx, p.ClientID, "milestone_released", "Milestone released",
			"A milestone was released to the provider. Please fund the next milestone to continue.",
			buildNotificationData(p.ID, p.ConversationID, p.Title))
		// Honest message to the provider: only claim "paid" when the
		// Stripe transfer can actually go through. If KYC isn't ready,
		// flag the funds as held until onboarding completes — the
		// scheduler will retry the transfer once payouts are enabled.
		providerBody := "Your milestone has been approved and paid. Work on the next milestone can start once the client funds it."
		if !providerReadyForPayouts {
			providerBody = "Your milestone has been approved. Funds are held pending your Stripe onboarding — finish your payment setup to receive the payout."
		}
		s.sendNotification(ctx, p.ProviderID, "milestone_released", "Milestone released",
			providerBody,
			buildNotificationData(p.ID, p.ConversationID, p.Title))

		if next, nextErr := s.milestones.GetCurrentActive(ctx, p.ID); nextErr == nil && next.Status == milestone.StatusPendingFunding {
			// Re-use the existing proposal_payment_requested message
			// type so the client sees the same "Pay now" CTA in the
			// conversation that appeared after the initial accept.
			// The payment page (web + mobile) already handles the
			// "active + pending-funding milestone" case.
			s.sendProposalMessage(ctx, p.ConversationID, input.UserID,
				"proposal_payment_requested", metadata)
			s.scheduleMilestoneFundReminder(ctx, next.ID)
			s.scheduleProposalAutoClose(ctx, p.ID)
		}
	}

	return nil
}

// RejectCompletion transitions a submitted milestone back to funded.
// The provider then has to re-address the deliverables and call
// RequestCompletion again — which restarts the auto-approval timer
// because SubmittedAt was cleared by milestone.Reject.
func (s *Service) RejectCompletion(ctx context.Context, input RejectCompletionInput) error {
	p, err := s.proposals.GetByIDForOrg(ctx, input.ProposalID, input.OrgID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	// Client-only directional check at org granularity.
	if err := s.requireOrgIsSide(ctx, p.ClientID, input.OrgID, domain.ErrNotClient); err != nil {
		return err
	}

	current, err := s.milestones.GetCurrentActive(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("get current milestone: %w", err)
	}
	if current.Status != milestone.StatusSubmitted {
		return domain.ErrInvalidStatus
	}

	if err := s.withMilestoneLock(ctx, current.ID, func(m *milestone.Milestone) error {
		return m.Reject()
	}); err != nil {
		return fmt.Errorf("reject milestone: %w", err)
	}

	if err := s.recomputeMacroStatus(ctx, p); err != nil {
		return fmt.Errorf("recompute macro status: %w", err)
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
	p, err := s.proposals.GetByIDForOrg(ctx, proposalID, orgID)
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

// ParticipantNames carries the display names for both sides of a
// proposal. Pre-populated by GetParticipantNamesBatch and keyed by
// proposal id in the returned map.
type ParticipantNames struct {
	ClientName   string
	ProviderName string
}

// GetParticipantNamesBatch resolves the display names of both sides
// (client + provider) for every proposal in a single batch query —
// closing the PERF-B-02 N+1: previously the caller looped over a page
// of N proposals and issued 2*N sequential users.GetByID lookups (~80–
// 200 ms p50 added on top of an already-loaded list endpoint).
//
// The function deduplicates user ids so we ask the DB for the unique
// set, then maps back to each proposal's pair. The map is keyed by
// proposal id; missing users (deleted accounts) collapse to "" so
// the caller renders the legacy fallback string.
//
// When usersBatch is nil (legacy test setups predating UsersBatch
// wiring), we degrade gracefully to per-id GetByID calls — preserving
// backwards-compatibility for every test that constructs the service
// without the new dependency. Production wiring always passes the
// concrete *postgres.UserRepository (which satisfies UserBatchReader)
// so the fast path is the default.
func (s *Service) GetParticipantNamesBatch(ctx context.Context, proposals []*domain.Proposal) map[uuid.UUID]ParticipantNames {
	out := make(map[uuid.UUID]ParticipantNames, len(proposals))
	if len(proposals) == 0 {
		return out
	}

	// Slow fallback: usersBatch unavailable.
	if s.usersBatch == nil {
		for _, p := range proposals {
			cn, pn := s.GetParticipantNames(ctx, p.ClientID, p.ProviderID)
			out[p.ID] = ParticipantNames{ClientName: cn, ProviderName: pn}
		}
		return out
	}

	// Collect every distinct user id we need to resolve.
	seen := make(map[uuid.UUID]struct{}, len(proposals)*2)
	ids := make([]uuid.UUID, 0, len(proposals)*2)
	for _, p := range proposals {
		if _, ok := seen[p.ClientID]; !ok {
			seen[p.ClientID] = struct{}{}
			ids = append(ids, p.ClientID)
		}
		if _, ok := seen[p.ProviderID]; !ok {
			seen[p.ProviderID] = struct{}{}
			ids = append(ids, p.ProviderID)
		}
	}

	users, err := s.usersBatch.GetByIDs(ctx, ids)
	if err != nil {
		// Don't break the list endpoint on a name lookup failure —
		// every proposal gets empty names, the frontend falls back to
		// "Unknown user" the same way it does for a deleted account.
		slog.Warn("get participant names batch failed", "error", err, "proposal_count", len(proposals))
		for _, p := range proposals {
			out[p.ID] = ParticipantNames{}
		}
		return out
	}

	byID := make(map[uuid.UUID]string, len(users))
	for _, u := range users {
		if u != nil {
			byID[u.ID] = u.DisplayName
		}
	}

	for _, p := range proposals {
		out[p.ID] = ParticipantNames{
			ClientName:   byID[p.ClientID],
			ProviderName: byID[p.ProviderID],
		}
	}
	return out
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

// providerCanReceivePayouts resolves the provider's organization (via
// the user repo) and asks the payment processor whether their Stripe
// Connect account is ready to receive transfers. Returns true on the
// happy path, false (with nil error) when the provider has no Stripe
// account or payouts are disabled, and a non-nil error when the check
// itself failed (in which case callers MUST treat the milestone as
// unreleasable to avoid a partial release).
//
// When the proposal service has no PaymentProcessor wired (legacy test
// setups, or a deployment without Stripe), the check is a no-op
// returning true — same posture as TransferMilestone, which is also a
// no-op when payments == nil. This keeps the existing "no payments
// configured" test paths working without changes.
func (s *Service) providerCanReceivePayouts(ctx context.Context, providerUserID uuid.UUID) (bool, error) {
	if s.payments == nil {
		return true, nil
	}
	if s.users == nil {
		// No way to resolve provider org → cannot guarantee readiness.
		// Fail closed: caller MUST treat this as not-ready.
		return false, nil
	}
	providerUser, err := s.users.GetByID(ctx, providerUserID)
	if err != nil {
		return false, fmt.Errorf("resolve provider user: %w", err)
	}
	if providerUser.OrganizationID == nil {
		return false, nil
	}
	ready, err := s.payments.CanProviderReceivePayouts(ctx, *providerUser.OrganizationID)
	if err != nil {
		return false, fmt.Errorf("payments: provider readiness check: %w", err)
	}
	return ready, nil
}

// providerEligibleForAutoTransfer reports whether the just-released
// milestone can be auto-transferred without waiting on the user to
// click "Retirer" in the wallet. Three conditions, ALL required:
//
//  1. The proposal service has a PaymentProcessor wired (production).
//  2. The provider's Stripe Connect account is ready (KYC + capabilities).
//  3. The provider's org has previously completed a successful manual
//     payout — the consent + the proof that Stripe payouts work for
//     them. Without this, a fresh provider whose KYC just landed but
//     has never received a payout still goes through the manual flow,
//     so a misconfiguration on their account never silently produces
//     a "released but not paid" milestone.
//
// Errors at any layer return false so callers default to the safer
// manual flow rather than auto-transferring on partial information.
func (s *Service) providerEligibleForAutoTransfer(ctx context.Context, providerUserID uuid.UUID) bool {
	if s.payments == nil || s.users == nil {
		return false
	}
	providerUser, err := s.users.GetByID(ctx, providerUserID)
	if err != nil || providerUser.OrganizationID == nil {
		return false
	}
	ready, err := s.payments.CanProviderReceivePayouts(ctx, *providerUser.OrganizationID)
	if err != nil || !ready {
		return false
	}
	consent, err := s.payments.HasAutoPayoutConsent(ctx, *providerUser.OrganizationID)
	if err != nil {
		return false
	}
	return consent
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
