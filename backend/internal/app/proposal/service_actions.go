package proposal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
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

// InitiatePayment creates a Stripe PaymentIntent for the proposal's
// current active milestone (phase 4 refactor). The Stripe amount comes
// from the milestone.amount, not the proposal.amount — the proposal is
// now a header and the milestone is the concrete escrow unit.
//
// Falls back to simulation mode when no PaymentProcessor is configured.
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

	// A payment is legal only when the proposal is accepted (no
	// milestone funded yet) OR active (previous milestones released
	// and the next one is waiting for funding). Anything else — pending,
	// disputed, completed — rejects the call.
	if p.Status != domain.StatusAccepted && p.Status != domain.StatusActive {
		return nil, domain.ErrInvalidStatus
	}

	// Locate the milestone awaiting funding. The strict-sequential
	// rule means there is exactly one such milestone at any instant:
	// the lowest-sequence non-terminal one, which must be in
	// pending_funding for a fund call to be legal.
	current, err := s.milestones.GetCurrentActive(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("get current milestone: %w", err)
	}
	if current.Status != milestone.StatusPendingFunding {
		return nil, domain.ErrInvalidStatus
	}

	// Real Stripe mode: ask the payment processor for an intent on the
	// milestone's amount. The processor persists a PaymentRecord keyed
	// on milestone_id, so repeated calls on the same current milestone
	// re-use the same PaymentIntent (idempotency by milestone, not by
	// proposal — a proposal with N milestones legitimately owns N
	// payment records).
	if s.payments != nil {
		result, err := s.payments.CreatePaymentIntent(ctx, service.PaymentIntentInput{
			ProposalID:     p.ID,
			MilestoneID:    current.ID,
			ClientID:       p.ClientID,
			ProviderID:     p.ProviderID,
			ProposalAmount: current.Amount,
		})
		if err != nil {
			return nil, fmt.Errorf("create payment intent: %w", err)
		}
		return result, nil
	}

	// Simulation fallback (dev mode): fund the milestone immediately
	// and recompute the macro status.
	return nil, s.simulatePayment(ctx, p, current, input.UserID)
}

// simulatePayment immediately funds the given milestone (dev mode only)
// and recomputes the proposal's macro status. Called by InitiatePayment
// when no real payment processor is wired.
func (s *Service) simulatePayment(ctx context.Context, p *domain.Proposal, current *milestone.Milestone, userID uuid.UUID) error {
	if err := s.withMilestoneLock(ctx, current.ID, func(m *milestone.Milestone) error {
		return m.Fund()
	}); err != nil {
		return fmt.Errorf("fund milestone: %w", err)
	}
	if err := s.recomputeMacroStatus(ctx, p); err != nil {
		return fmt.Errorf("recompute macro status: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, userID, "proposal_paid", metadata)
	s.sendNotification(ctx, p.ProviderID, "proposal_paid", "Payment received",
		"A payment has been made for your proposal",
		buildNotificationData(p.ID, p.ConversationID, p.Title))
	return nil
}

// ConfirmPaymentAndActivate is called by the webhook handler (or the
// frontend fallback) after Stripe has confirmed a PaymentIntent. It
// transitions the current active milestone from pending_funding to
// funded and recomputes the proposal's macro status.
//
// Idempotent: if the milestone is already funded (or beyond), the call
// is a no-op and returns nil, so duplicate webhook deliveries don't
// double-fund. The dedicated stripe_webhook_events table (phase 7)
// adds a second layer of idempotency at the webhook boundary.
//
// The credit-bonus fraud check and the KYC first-earning timestamp are
// NOT fired here — per user decisions F4 and F5 they are triggered
// when the LAST milestone of a proposal is released (i.e. when the
// macro status transitions to completed), not at first funding.
func (s *Service) ConfirmPaymentAndActivate(ctx context.Context, proposalID uuid.UUID) error {
	p, err := s.proposals.GetByID(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	current, err := s.milestones.GetCurrentActive(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("get current milestone: %w", err)
	}
	// Idempotency: if the milestone is already beyond pending_funding
	// (funded, submitted, approved, released, etc.) the webhook has
	// already been processed and we have nothing to do.
	if current.Status != milestone.StatusPendingFunding {
		return nil
	}

	if err := s.withMilestoneLock(ctx, current.ID, func(m *milestone.Milestone) error {
		return m.Fund()
	}); err != nil {
		return fmt.Errorf("fund milestone: %w", err)
	}
	if err := s.recomputeMacroStatus(ctx, p); err != nil {
		return fmt.Errorf("recompute macro status: %w", err)
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

// ListMilestones returns every milestone of a proposal ordered by
// ascending sequence. Read-only — the handler calls it alongside
// GetProposal to materialise the milestone tracker in the response.
//
// Returns an empty slice when the milestones repository is not wired
// (legacy test setups that predate phase 4) so the response degrades
// gracefully to the one-time UX instead of panicking.
func (s *Service) ListMilestones(ctx context.Context, proposalID uuid.UUID) ([]*milestone.Milestone, error) {
	if s.milestones == nil {
		return nil, nil
	}
	return s.milestones.ListByProposal(ctx, proposalID)
}

// ListMilestonesForProposals batches the milestone lookup across many
// proposals in a single round trip — used by the project list endpoint
// to avoid N+1 queries when rendering each card with its current
// milestone CTA.
//
// Same nil-safety as ListMilestones for legacy test setups.
func (s *Service) ListMilestonesForProposals(ctx context.Context, proposalIDs []uuid.UUID) (map[uuid.UUID][]*milestone.Milestone, error) {
	if s.milestones == nil {
		return map[uuid.UUID][]*milestone.Milestone{}, nil
	}
	return s.milestones.ListByProposals(ctx, proposalIDs)
}

// CancelProposalInput is the input for the boundary cancel flow.
type CancelProposalInput struct {
	ProposalID uuid.UUID
	UserID     uuid.UUID
	OrgID      uuid.UUID
}

// CancelProposal performs a milestone-boundary cancellation: every
// pending_funding milestone is cancelled in place, leaving released
// milestones untouched. Either side (client or provider) can call it
// since neither party is committing additional money.
//
// Cancellation is only legal when there is NO funded/submitted/disputed
// milestone (i.e. nothing in flight). If a milestone is mid-execution
// the caller must use the dispute flow to negotiate an exit instead.
//
// After cancellation, the proposal macro status is recomputed:
//   - If at least one milestone was released → completed (project
//     ended early but with deliverables — the existing macro projection
//     handles this).
//   - If no milestones were released → declined (hard-stop).
//
// This method does not currently support partial-fund recovery — that
// path goes through the dispute service (phase 8).
func (s *Service) CancelProposal(ctx context.Context, input CancelProposalInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	// Either party may cancel at a milestone boundary — no money is
	// changing hands so the directional check is not needed. We just
	// require the caller to be on EITHER side of the proposal.
	if err := s.requireOrgIsParticipant(ctx, p, input.OrgID); err != nil {
		return err
	}

	milestones, err := s.milestones.ListByProposal(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("list milestones: %w", err)
	}

	// Forbid cancellation while a milestone is in flight: funded /
	// submitted / approved / disputed all hold escrow money. The
	// dispute flow handles those transitions instead.
	for _, m := range milestones {
		switch m.Status {
		case milestone.StatusFunded, milestone.StatusSubmitted, milestone.StatusApproved, milestone.StatusDisputed:
			return domain.ErrInvalidStatus
		}
	}

	// Cancel every pending_funding milestone via the optimistic-locked
	// path. Concurrent updates are swallowed per-item so a parallel
	// transition (e.g. another operator funding the milestone right
	// now) doesn't crash the sweep.
	for _, m := range milestones {
		if m.Status != milestone.StatusPendingFunding {
			continue
		}
		if err := s.withMilestoneLock(ctx, m.ID, func(mm *milestone.Milestone) error {
			return mm.Cancel()
		}); err != nil {
			if errors.Is(err, milestone.ErrConcurrentUpdate) {
				continue
			}
			return fmt.Errorf("cancel milestone %s: %w", m.ID, err)
		}
	}

	if err := s.recomputeMacroStatus(ctx, p); err != nil {
		return fmt.Errorf("recompute macro status: %w", err)
	}

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_cancelled", metadata)

	s.sendNotification(ctx, p.ClientID, "proposal_cancelled", "Project cancelled",
		"The project has been cancelled at a milestone boundary.",
		buildNotificationData(p.ID, p.ConversationID, p.Title))
	s.sendNotification(ctx, p.ProviderID, "proposal_cancelled", "Project cancelled",
		"The project has been cancelled at a milestone boundary.",
		buildNotificationData(p.ID, p.ConversationID, p.Title))

	return nil
}

// requireOrgIsParticipant returns nil if the caller's org owns either
// the client side OR the provider side of the proposal. Used by
// non-directional actions (cancel at boundary) where both parties may
// initiate the call.
func (s *Service) requireOrgIsParticipant(ctx context.Context, p *domain.Proposal, callerOrgID uuid.UUID) error {
	if err := s.requireOrgIsSide(ctx, p.ClientID, callerOrgID, domain.ErrNotAuthorized); err == nil {
		return nil
	}
	if err := s.requireOrgIsSide(ctx, p.ProviderID, callerOrgID, domain.ErrNotAuthorized); err == nil {
		return nil
	}
	return domain.ErrNotAuthorized
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

// RequestCompletion transitions the proposal's current active milestone
// from funded to submitted. The provider calls it when the milestone's
// deliverables are ready for client review — starting the auto-approval
// timer that the phase-6 scheduler will fire in 7 days (default).
func (s *Service) RequestCompletion(ctx context.Context, input RequestCompletionInput) error {
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
	if err != nil {
		return fmt.Errorf("get proposal: %w", err)
	}

	// Provider-only directional check at org granularity.
	if err := s.requireOrgIsSide(ctx, p.ProviderID, input.OrgID, domain.ErrNotProvider); err != nil {
		return err
	}

	current, err := s.milestones.GetCurrentActive(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("get current milestone: %w", err)
	}
	if current.Status != milestone.StatusFunded {
		return domain.ErrInvalidStatus
	}

	if err := s.withMilestoneLock(ctx, current.ID, func(m *milestone.Milestone) error {
		return m.Submit()
	}); err != nil {
		return fmt.Errorf("submit milestone: %w", err)
	}

	if err := s.recomputeMacroStatus(ctx, p); err != nil {
		return fmt.Errorf("recompute macro status: %w", err)
	}

	// Schedule auto-approval: if the client doesn't act within
	// autoApprovalDelay (default 7 days), the worker will pick this
	// event up and call AutoApproveMilestone, transitioning the
	// milestone all the way to released. Failure to schedule is
	// logged but doesn't block the submission — we'd rather miss
	// auto-approval than reject a legitimate provider call.
	s.scheduleMilestoneAutoApprove(ctx, current.ID)

	metadata := buildStatusMetadata(p)
	s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "proposal_completion_requested", metadata)

	s.sendNotification(ctx, p.ClientID, "completion_requested", "Completion requested",
		"The provider has requested to mark the mission as complete",
		buildNotificationData(p.ID, p.ConversationID, p.Title))

	return nil
}

// CompleteProposal approves AND releases the proposal's current active
// milestone in a single locked transition. If the released milestone
// was the LAST one of the proposal, the macro status transitions to
// completed — at which point the credit-bonus fraud check runs and
// the KYC first-earning timestamp is recorded (user decisions F4/F5).
// Otherwise the proposal drops back to "active" (the next milestone
// now becomes the current one, awaiting funding).
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

	current, err := s.milestones.GetCurrentActive(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("get current milestone: %w", err)
	}
	if current.Status != milestone.StatusSubmitted {
		return domain.ErrInvalidStatus
	}

	// Pre-check provider KYC BEFORE flipping any state. If the provider
	// has no Stripe Connect account, or payouts are not enabled, the
	// downstream TransferMilestone (mid-project) or runEndOfProjectEffects
	// (last milestone) Stripe transfer would fail silently — and the
	// "milestone has been approved and paid" notification would be a lie.
	// Fail-fast with a typed domain error so the handler can map it to
	// HTTP 412 and the client can prompt the provider to finish KYC.
	ready, kerr := s.providerCanReceivePayouts(ctx, p.ProviderID)
	if kerr != nil {
		return fmt.Errorf("check provider payouts: %w", kerr)
	}
	if !ready {
		return domain.ErrProviderKYCNotReady
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
		s.runEndOfProjectEffects(ctx, p)
	} else {
		// Mid-project release: transfer the just-released milestone's
		// escrow to the provider. Uses TransferMilestone(current.ID) so
		// multi-milestone proposals release the correct record (the
		// legacy TransferToProvider(proposal_id) would pick the most
		// recent record, which on a 3-jalon proposal is almost always
		// the wrong one and leaves jalons 1..N-1 stuck in escrow).
		if s.payments != nil {
			if err := s.payments.TransferMilestone(ctx, current.ID); err != nil {
				slog.Error("mid-project: failed to transfer milestone",
					"proposal_id", p.ID, "milestone_id", current.ID, "error", err)
			}
		}

		metadata := buildStatusMetadata(p)
		s.sendProposalMessage(ctx, p.ConversationID, input.UserID, "milestone_released", metadata)
		s.sendNotification(ctx, p.ClientID, "milestone_released", "Milestone released",
			"A milestone was released to the provider. Please fund the next milestone to continue.",
			buildNotificationData(p.ID, p.ConversationID, p.Title))
		s.sendNotification(ctx, p.ProviderID, "milestone_released", "Milestone released",
			"Your milestone has been approved and paid. Work on the next milestone can start once the client funds it.",
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
	p, err := s.proposals.GetByID(ctx, input.ProposalID)
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
