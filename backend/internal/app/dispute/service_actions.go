package dispute

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	disputedomain "marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/domain/message"
	milestonedomain "marketplace-backend/internal/domain/milestone"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/system"
)

// loadDisputeForActor reads a dispute under the appropriate
// tenant gate for the calling context. See loadProposalForActor
// (proposal/service_actions.go) for the rationale; the same
// system-actor / user-driven boundary applies here.
//
// User-facing dispute handlers run with an authenticated org
// context populated by the auth middleware → GetByIDForOrg, which
// enforces RLS on the disputes table (USING client_org OR
// provider_org = current_setting('app.current_org_id')).
//
// System-actor callers (the dispute scheduler, the dev-only
// force-escalate endpoint) tag their context with
// system.WithSystemActor and take the legacy GetByID path —
// production bypass-RLS pool selection happens inside the
// adapter, not here.
func (s *Service) loadDisputeForActor(ctx context.Context, id uuid.UUID) (*disputedomain.Dispute, error) {
	if system.IsSystemActor(ctx) {
		return s.disputes.GetByID(ctx, id)
	}
	orgID := middleware.MustGetOrgID(ctx)
	return s.disputes.GetByIDForOrg(ctx, id, orgID)
}

// loadProposalForActor mirrors loadDisputeForActor for the
// proposal repository — the dispute service touches proposals
// transitively (e.g. RestoreFromDispute) and the same
// system-actor / user-driven split applies.
func (s *Service) loadProposalForActor(ctx context.Context, id uuid.UUID) (*proposaldomain.Proposal, error) {
	if system.IsSystemActor(ctx) {
		return s.proposals.GetByID(ctx, id)
	}
	orgID := middleware.MustGetOrgID(ctx)
	return s.proposals.GetByIDForOrg(ctx, id, orgID)
}

// markMilestoneDisputed transitions the milestone to disputed status
// inside an optimistic-locked update. Mirrors the proposal service's
// withMilestoneLock pattern so the dispute service stays decoupled
// from the proposal app service while still enforcing the same
// concurrency guarantees.
func (s *Service) markMilestoneDisputed(ctx context.Context, milestoneID, disputeID uuid.UUID) error {
	m, err := s.milestones.GetByIDWithVersion(ctx, milestoneID)
	if err != nil {
		return err
	}
	if err := m.OpenDispute(disputeID); err != nil {
		return err
	}
	return s.milestones.Update(ctx, m)
}

// restoreMilestoneFromDispute is the symmetric helper called from
// dispute resolution / cancellation paths. The target status is
// chosen by the dispute service based on the resolution type
// (funded for partial refund, released for full release, refunded
// for full refund).
func (s *Service) restoreMilestoneFromDispute(ctx context.Context, milestoneID uuid.UUID, target milestonedomain.MilestoneStatus) error {
	m, err := s.milestones.GetByIDWithVersion(ctx, milestoneID)
	if err != nil {
		return err
	}
	if err := m.RestoreFromDispute(target); err != nil {
		return err
	}
	return s.milestones.Update(ctx, m)
}

// ---------------------------------------------------------------------------
// Input types
// ---------------------------------------------------------------------------

type OpenDisputeInput struct {
	ProposalID      uuid.UUID
	InitiatorID     uuid.UUID
	Reason          string
	Description     string // Detailed description for admin mediation (private)
	MessageToParty  string // Short message to the other party (visible in conversation)
	RequestedAmount int64
	Attachments     []AttachmentInput
}

type AttachmentInput struct {
	Filename string
	URL      string
	Size     int64
	MimeType string
}

type CounterProposeInput struct {
	DisputeID      uuid.UUID
	ProposerID     uuid.UUID
	AmountClient   int64
	AmountProvider int64
	Message        string
	Attachments    []AttachmentInput
}

type RespondToCounterInput struct {
	DisputeID         uuid.UUID
	CounterProposalID uuid.UUID
	UserID            uuid.UUID
	Accept            bool
}

type CancelDisputeInput struct {
	DisputeID uuid.UUID
	UserID    uuid.UUID
}

type RespondToCancellationInput struct {
	DisputeID uuid.UUID
	UserID    uuid.UUID
	Accept    bool
}

// CancelDisputeResult indicates what happened when a cancellation was attempted.
// When Cancelled is true, the dispute was terminated directly (the respondent
// had not yet engaged). When Cancelled is false and Requested is true, a
// cancellation request was created and now waits for the respondent's consent.
type CancelDisputeResult struct {
	Cancelled bool
	Requested bool
}

type AdminResolveInput struct {
	DisputeID      uuid.UUID
	AdminID        uuid.UUID
	AmountClient   int64
	AmountProvider int64
	Note           string
}

// ---------------------------------------------------------------------------
// OpenDispute
// ---------------------------------------------------------------------------

func (s *Service) OpenDispute(ctx context.Context, in OpenDisputeInput) (*disputedomain.Dispute, error) {
	p, err := s.loadProposalForActor(ctx, in.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("get proposal: %w", err)
	}
	if p.Status != proposaldomain.StatusActive && p.Status != proposaldomain.StatusCompletionRequested {
		return nil, disputedomain.ErrProposalNotDisputable
	}

	existing, err := s.disputes.GetByProposalID(ctx, in.ProposalID)
	if err != nil {
		return nil, fmt.Errorf("check existing dispute: %w", err)
	}
	if existing != nil {
		return nil, disputedomain.ErrAlreadyDisputed
	}

	// Phase 8: a dispute is scoped to a single milestone — the one
	// currently in flight. Resolve it server-side from the proposal
	// so the API surface stays simple (only proposal_id required).
	// Reject if the milestone is not in funded or submitted state:
	// pending_funding milestones have no escrow to dispute, and
	// terminal milestones (released, cancelled, refunded) are
	// immutable.
	current, err := s.milestones.GetCurrentActive(ctx, p.ID)
	if err != nil {
		return nil, fmt.Errorf("get current milestone: %w", err)
	}
	if current.Status != milestonedomain.StatusFunded && current.Status != milestonedomain.StatusSubmitted {
		return nil, disputedomain.ErrProposalNotDisputable
	}

	respondentID := p.ProviderID
	if in.InitiatorID == p.ProviderID {
		respondentID = p.ClientID
	}

	// Resolve both parties' current organizations so the dispute is
	// visible to every operator of either org, not just the user who
	// opened it (Stripe Dashboard shared workspace).
	clientUser, err := s.users.GetByID(ctx, p.ClientID)
	if err != nil {
		return nil, fmt.Errorf("lookup client user: %w", err)
	}
	providerUser, err := s.users.GetByID(ctx, p.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("lookup provider user: %w", err)
	}
	if clientUser.OrganizationID == nil || providerUser.OrganizationID == nil {
		return nil, fmt.Errorf("open dispute: participants must belong to an organization")
	}

	// Phase 8: clamp the requested amount to the milestone amount —
	// a dispute can never claim more than the escrow it targets.
	if in.RequestedAmount > current.Amount {
		return nil, disputedomain.ErrInvalidAmount
	}

	d, err := disputedomain.NewDispute(disputedomain.NewDisputeInput{
		ProposalID:             p.ID,
		MilestoneID:            current.ID,
		ConversationID:         p.ConversationID,
		InitiatorID:            in.InitiatorID,
		RespondentID:           respondentID,
		ClientID:               p.ClientID,
		ProviderID:             p.ProviderID,
		ClientOrganizationID:   *clientUser.OrganizationID,
		ProviderOrganizationID: *providerUser.OrganizationID,
		Reason:                 disputedomain.Reason(in.Reason),
		Description:            in.Description,
		RequestedAmount:        in.RequestedAmount,
		// ProposalAmount carries the milestone amount post-phase-8
		// (the field name is preserved for backward compatibility
		// with existing SQL queries and resolution split logic).
		ProposalAmount: current.Amount,
	})
	if err != nil {
		return nil, err
	}

	if err := s.disputes.Create(ctx, d); err != nil {
		return nil, fmt.Errorf("persist dispute: %w", err)
	}

	for _, att := range in.Attachments {
		e := &disputedomain.Evidence{
			ID:         uuid.New(),
			DisputeID:  d.ID,
			UploaderID: in.InitiatorID,
			Filename:   att.Filename,
			URL:        att.URL,
			Size:       att.Size,
			MimeType:   att.MimeType,
		}
		if err := s.disputes.CreateEvidence(ctx, e); err != nil {
			slog.Warn("dispute: failed to save evidence", "error", err)
		}
	}

	// Phase 8: also mark the milestone disputed in the same flow, so
	// the macro state (proposal-level) and the milestone-level state
	// stay coherent. The milestone repo applies optimistic locking.
	if err := s.markMilestoneDisputed(ctx, current.ID, d.ID); err != nil {
		return nil, fmt.Errorf("mark milestone disputed: %w", err)
	}

	if err := p.MarkDisputed(d.ID); err != nil {
		return nil, fmt.Errorf("mark proposal disputed: %w", err)
	}
	if err := s.proposals.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update proposal: %w", err)
	}

	// Auto-create the first proposal from the initiator's requested amount.
	// This gives the other party something actionable immediately (accept/reject/counter).
	// All amounts are scoped to the disputed milestone (post-phase-8).
	var clientAmt, providerAmt int64
	if in.InitiatorID == p.ClientID {
		clientAmt = in.RequestedAmount
		providerAmt = current.Amount - in.RequestedAmount
	} else {
		providerAmt = in.RequestedAmount
		clientAmt = current.Amount - in.RequestedAmount
	}
	cp, cpErr := disputedomain.NewCounterProposal(disputedomain.NewCounterProposalInput{
		DisputeID:      d.ID,
		ProposerID:     in.InitiatorID,
		AmountClient:   clientAmt,
		AmountProvider: providerAmt,
		ProposalAmount: current.Amount,
		Message:        in.MessageToParty,
	})
	if cpErr == nil {
		_ = s.disputes.CreateCounterProposal(ctx, cp)
	}

	// Single system message: "Litige ouvert" with the initial proposal embedded
	s.sendSystemMessage(ctx, d.ConversationID, in.InitiatorID,
		message.MessageTypeDisputeOpened, buildOpenedWithProposalMetadata(d, cp, in.MessageToParty))

	s.sendNotification(ctx, respondentID, "dispute_opened",
		"Litige ouvert", "Un litige a ete ouvert sur votre mission.", d.ID)

	return d, nil
}

// ---------------------------------------------------------------------------
// CounterPropose
// ---------------------------------------------------------------------------

func (s *Service) CounterPropose(ctx context.Context, in CounterProposeInput) (*disputedomain.CounterProposal, error) {
	d, err := s.loadDisputeForActor(ctx, in.DisputeID)
	if err != nil {
		return nil, err
	}
	// Counter-proposals stay open through admin mediation: parties can still
	// reach an amicable agreement until the admin issues a final decision.
	if d.Status != disputedomain.StatusOpen &&
		d.Status != disputedomain.StatusNegotiation &&
		d.Status != disputedomain.StatusEscalated {
		return nil, disputedomain.ErrInvalidStatus
	}
	if !d.IsParticipant(in.ProposerID) {
		return nil, disputedomain.ErrNotParticipant
	}

	if err := s.disputes.SupersedeAllPending(ctx, d.ID); err != nil {
		return nil, fmt.Errorf("supersede pending: %w", err)
	}

	cp, err := disputedomain.NewCounterProposal(disputedomain.NewCounterProposalInput{
		DisputeID:      d.ID,
		ProposerID:     in.ProposerID,
		AmountClient:   in.AmountClient,
		AmountProvider: in.AmountProvider,
		ProposalAmount: d.ProposalAmount,
		Message:        in.Message,
	})
	if err != nil {
		return nil, err
	}

	if err := s.disputes.CreateCounterProposal(ctx, cp); err != nil {
		return nil, fmt.Errorf("persist counter-proposal: %w", err)
	}

	// Persist attachments linked to this counter-proposal
	for _, att := range in.Attachments {
		e := &disputedomain.Evidence{
			ID:                uuid.New(),
			DisputeID:         d.ID,
			CounterProposalID: &cp.ID,
			UploaderID:        in.ProposerID,
			Filename:          att.Filename,
			URL:               att.URL,
			Size:              att.Size,
			MimeType:          att.MimeType,
		}
		if err := s.disputes.CreateEvidence(ctx, e); err != nil {
			slog.Warn("dispute: failed to save counter-proposal evidence", "error", err)
		}
	}

	if d.Status == disputedomain.StatusOpen {
		_ = d.MarkNegotiation()
	}
	if in.ProposerID == d.RespondentID {
		d.RecordRespondentReply()
	}
	// A new counter-proposal signals active negotiation — any pending
	// cancellation request is implicitly withdrawn.
	d.ClearCancellationRequest()
	d.RecordActivity()
	if err := s.disputes.Update(ctx, d); err != nil {
		return nil, fmt.Errorf("update dispute: %w", err)
	}

	otherID := d.InitiatorID
	if in.ProposerID == d.InitiatorID {
		otherID = d.RespondentID
	}

	s.sendSystemMessage(ctx, d.ConversationID, in.ProposerID,
		message.MessageTypeDisputeCounterProposal, buildCounterMetadata(d, cp))
	s.sendNotification(ctx, otherID, "dispute_counter_proposal",
		"Nouvelle proposition", "Une contre-proposition a ete faite sur votre litige.", d.ID)

	return cp, nil
}

// ---------------------------------------------------------------------------
// RespondToCounter
// ---------------------------------------------------------------------------

func (s *Service) RespondToCounter(ctx context.Context, in RespondToCounterInput) error {
	d, err := s.loadDisputeForActor(ctx, in.DisputeID)
	if err != nil {
		return err
	}

	cp, err := s.disputes.GetCounterProposalByID(ctx, in.CounterProposalID)
	if err != nil {
		return err
	}

	if in.Accept {
		if err := cp.Accept(in.UserID); err != nil {
			return err
		}
		if err := s.disputes.UpdateCounterProposal(ctx, cp); err != nil {
			return fmt.Errorf("update counter-proposal: %w", err)
		}

		if err := d.Resolve(disputedomain.ResolveInput{
			AmountClient:   cp.AmountClient,
			AmountProvider: cp.AmountProvider,
			Note:           "Accord amiable entre les parties.",
		}); err != nil {
			return fmt.Errorf("resolve dispute: %w", err)
		}
		if err := s.disputes.Update(ctx, d); err != nil {
			return fmt.Errorf("update dispute: %w", err)
		}

		s.restoreProposalAndDistribute(ctx, d)

		s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
			message.MessageTypeDisputeResolved, buildResolvedMetadata(d))
		s.notifyBothParties(ctx, d, "dispute_resolved",
			"Litige resolu", "Le litige a ete resolu par accord amiable.")
	} else {
		if err := cp.Reject(in.UserID); err != nil {
			return err
		}
		if err := s.disputes.UpdateCounterProposal(ctx, cp); err != nil {
			return fmt.Errorf("update counter-proposal: %w", err)
		}

		// Rejecting a proposal IS engagement with the dispute. Record the
		// respondent's first reply so that subsequent cancellation attempts
		// by the initiator go through the request flow (consent required).
		if in.UserID == d.RespondentID {
			d.RecordRespondentReply()
		}
		d.RecordActivity()
		if err := s.disputes.Update(ctx, d); err != nil {
			return fmt.Errorf("update dispute: %w", err)
		}

		s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
			message.MessageTypeDisputeCounterRejected, buildCounterMetadata(d, cp))
		// Notify the proposer that their amicable resolution proposal was
		// refused so they can react (make a new proposal, escalate, etc).
		s.sendNotification(ctx, cp.ProposerID, "dispute_counter_rejected",
			"Proposition refusee",
			"L'autre partie a refuse votre proposition de resolution du litige.", d.ID)
	}

	return nil
}

// ---------------------------------------------------------------------------
// CancelDispute
// ---------------------------------------------------------------------------

// CancelDispute attempts to cancel a dispute on behalf of any participant.
//
// The domain decides between two paths based on the caller's role and the
// respondent's engagement state:
//   - Initiator + respondent has NOT yet replied → direct cancellation
//     (result.Cancelled=true), proposal restored to active.
//   - Initiator + respondent HAS replied, OR respondent (any time) →
//     cancellation REQUEST is created (result.Requested=true) and the OTHER
//     party is notified for consent.
//
// When a request is created, any pending counter-proposals are superseded.
// This is symmetric with CounterPropose, which clears pending cancellation
// requests: the latest intent always wins, so the two flows never coexist.
func (s *Service) CancelDispute(ctx context.Context, in CancelDisputeInput) (CancelDisputeResult, error) {
	d, err := s.loadDisputeForActor(ctx, in.DisputeID)
	if err != nil {
		return CancelDisputeResult{}, err
	}

	cancelled, err := d.Cancel(in.UserID)
	if err != nil {
		return CancelDisputeResult{}, err
	}

	// If a cancellation REQUEST was created, supersede any pending
	// counter-proposals BEFORE persisting the dispute. Doing it in this
	// order means a failure here leaves the dispute untouched in DB
	// (the user can simply retry) instead of leaving an inconsistent
	// state where the request is set but old CPs are still pending.
	if !cancelled {
		if err := s.disputes.SupersedeAllPending(ctx, d.ID); err != nil {
			return CancelDisputeResult{}, fmt.Errorf("supersede pending: %w", err)
		}
	}

	if err := s.disputes.Update(ctx, d); err != nil {
		return CancelDisputeResult{}, fmt.Errorf("update dispute: %w", err)
	}

	if cancelled {
		// Direct cancellation: restore the proposal to active.
		//
		// BUG-03: previously this branch swallowed the proposals.Update
		// error with `_ = s.proposals.Update(...)`. If the UPDATE failed
		// (DB blip, optimistic concurrency conflict), the dispute was
		// `cancelled` but the proposal stayed `disputed` — frozen
		// pair, no automatic recovery, user could not understand why.
		//
		// Recovery strategy: surface the error so the HTTP layer can
		// return 500 and the client can retry. The dispute itself was
		// already persisted as cancelled (s.disputes.Update above), so
		// retrying CancelDispute is a no-op on the dispute side and
		// will re-attempt the proposal restore — eventually consistent.
		// We log at ERROR level so the operations team sees the
		// inconsistency before the user retries.
		p, err := s.loadProposalForActor(ctx, d.ProposalID)
		if err != nil {
			return CancelDisputeResult{}, fmt.Errorf("get proposal for restore: %w", err)
		}
		if err := p.RestoreFromDispute(proposaldomain.StatusActive); err != nil {
			slog.Error("dispute cancel: domain restore rejected",
				"dispute_id", d.ID, "proposal_id", p.ID, "error", err)
			return CancelDisputeResult{}, fmt.Errorf("restore proposal from dispute: %w", err)
		}
		if err := s.proposals.Update(ctx, p); err != nil {
			// Dispute is cancelled in DB but proposal is still
			// disputed in DB — surface so the caller retries.
			slog.Error("dispute cancel: proposal update failed — pair may be inconsistent until retry",
				"dispute_id", d.ID, "proposal_id", p.ID, "error", err)
			return CancelDisputeResult{}, fmt.Errorf("update proposal after dispute cancel: %w", err)
		}

		s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
			message.MessageTypeDisputeCancelled, buildCancelledMetadata(d))
		s.sendNotification(ctx, d.RespondentID, "dispute_cancelled",
			"Litige annule", "Le litige a ete annule par l'initiateur.", d.ID)
		return CancelDisputeResult{Cancelled: true}, nil
	}

	// Cancellation request created — notify the OTHER participant (not the
	// requester) so they can accept or refuse.
	otherParty := d.RespondentID
	if in.UserID == d.RespondentID {
		otherParty = d.InitiatorID
	}
	s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
		message.MessageTypeDisputeCancellationRequested,
		buildCancellationRequestedMetadata(d, in.UserID))
	s.sendNotification(ctx, otherParty, "dispute_cancellation_requested",
		"Demande d'annulation",
		"L'autre partie demande l'annulation du litige. Votre accord est requis.", d.ID)

	return CancelDisputeResult{Requested: true}, nil
}
