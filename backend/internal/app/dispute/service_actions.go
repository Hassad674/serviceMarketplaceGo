package dispute

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	disputedomain "marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/domain/message"
	proposaldomain "marketplace-backend/internal/domain/proposal"
)

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
	p, err := s.proposals.GetByID(ctx, in.ProposalID)
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

	respondentID := p.ProviderID
	if in.InitiatorID == p.ProviderID {
		respondentID = p.ClientID
	}

	d, err := disputedomain.NewDispute(disputedomain.NewDisputeInput{
		ProposalID:      p.ID,
		ConversationID:  p.ConversationID,
		InitiatorID:     in.InitiatorID,
		RespondentID:    respondentID,
		ClientID:        p.ClientID,
		ProviderID:      p.ProviderID,
		Reason:          disputedomain.Reason(in.Reason),
		Description:     in.Description,
		RequestedAmount: in.RequestedAmount,
		ProposalAmount:  p.Amount,
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

	if err := p.MarkDisputed(d.ID); err != nil {
		return nil, fmt.Errorf("mark proposal disputed: %w", err)
	}
	if err := s.proposals.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("update proposal: %w", err)
	}

	// Auto-create the first proposal from the initiator's requested amount.
	// This gives the other party something actionable immediately (accept/reject/counter).
	var clientAmt, providerAmt int64
	if in.InitiatorID == p.ClientID {
		clientAmt = in.RequestedAmount
		providerAmt = p.Amount - in.RequestedAmount
	} else {
		providerAmt = in.RequestedAmount
		clientAmt = p.Amount - in.RequestedAmount
	}
	cp, cpErr := disputedomain.NewCounterProposal(disputedomain.NewCounterProposalInput{
		DisputeID:      d.ID,
		ProposerID:     in.InitiatorID,
		AmountClient:   clientAmt,
		AmountProvider: providerAmt,
		ProposalAmount: p.Amount,
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
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
	if err != nil {
		return nil, err
	}
	if d.Status != disputedomain.StatusOpen && d.Status != disputedomain.StatusNegotiation {
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
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
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
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
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
		p, err := s.proposals.GetByID(ctx, d.ProposalID)
		if err != nil {
			return CancelDisputeResult{}, fmt.Errorf("get proposal for restore: %w", err)
		}
		if err := p.RestoreFromDispute(proposaldomain.StatusActive); err != nil {
			slog.Warn("dispute: failed to restore proposal", "error", err)
		} else {
			_ = s.proposals.Update(ctx, p)
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

// ---------------------------------------------------------------------------
// RespondToCancellation
// ---------------------------------------------------------------------------

// RespondToCancellation processes the other party's decision on a pending
// cancellation request. Either party may have requested the cancellation;
// only the OTHER (not the requester) can respond.
//   - Accept: the dispute is cancelled and the proposal is restored.
//   - Refuse: the cancellation request is cleared and the dispute continues.
func (s *Service) RespondToCancellation(ctx context.Context, in RespondToCancellationInput) error {
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
	if err != nil {
		return err
	}

	// Capture the requester BEFORE the domain method clears it on refuse,
	// so we can notify the right person afterwards.
	var requesterID uuid.UUID
	if d.CancellationRequestedBy != nil {
		requesterID = *d.CancellationRequestedBy
	}

	if err := d.RespondToCancellationRequest(in.UserID, in.Accept); err != nil {
		return err
	}
	if err := s.disputes.Update(ctx, d); err != nil {
		return fmt.Errorf("update dispute: %w", err)
	}

	if in.Accept {
		p, err := s.proposals.GetByID(ctx, d.ProposalID)
		if err != nil {
			return fmt.Errorf("get proposal for restore: %w", err)
		}
		if err := p.RestoreFromDispute(proposaldomain.StatusActive); err != nil {
			slog.Warn("dispute: failed to restore proposal", "error", err)
		} else {
			_ = s.proposals.Update(ctx, p)
		}

		s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
			message.MessageTypeDisputeCancelled, buildCancelledMetadata(d))
		s.notifyBothParties(ctx, d, "dispute_cancelled",
			"Litige annule", "Les deux parties se sont mises d'accord pour annuler le litige.")
		return nil
	}

	s.sendSystemMessage(ctx, d.ConversationID, in.UserID,
		message.MessageTypeDisputeCancellationRefused,
		buildCancellationRefusedMetadata(d, in.UserID))
	if requesterID != uuid.Nil {
		s.sendNotification(ctx, requesterID, "dispute_cancellation_refused",
			"Annulation refusee",
			"L'autre partie a refuse votre demande d'annulation. Le litige continue.", d.ID)
	}

	return nil
}

// ---------------------------------------------------------------------------
// AdminResolve
// ---------------------------------------------------------------------------

func (s *Service) AdminResolve(ctx context.Context, in AdminResolveInput) error {
	d, err := s.disputes.GetByID(ctx, in.DisputeID)
	if err != nil {
		return err
	}
	if d.Status != disputedomain.StatusEscalated {
		return disputedomain.ErrInvalidStatus
	}

	if err := d.Resolve(disputedomain.ResolveInput{
		ResolvedBy:     in.AdminID,
		AmountClient:   in.AmountClient,
		AmountProvider: in.AmountProvider,
		Note:           in.Note,
	}); err != nil {
		return err
	}
	if err := s.disputes.Update(ctx, d); err != nil {
		return fmt.Errorf("update dispute: %w", err)
	}

	s.restoreProposalAndDistribute(ctx, d)

	s.sendSystemMessage(ctx, d.ConversationID, uuid.Nil,
		message.MessageTypeDisputeResolved, buildResolvedMetadata(d))
	s.notifyBothParties(ctx, d, "dispute_resolved",
		"Litige resolu", "L'equipe de mediation a rendu sa decision.")

	return nil
}

// ---------------------------------------------------------------------------
// Read methods
// ---------------------------------------------------------------------------

type DisputeDetail struct {
	Dispute          *disputedomain.Dispute
	Evidence         []*disputedomain.Evidence
	CounterProposals []*disputedomain.CounterProposal
}

func (s *Service) GetDispute(ctx context.Context, userID, disputeID uuid.UUID) (*DisputeDetail, error) {
	d, err := s.disputes.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	if !d.IsParticipant(userID) {
		return nil, disputedomain.ErrNotParticipant
	}
	return s.loadDetail(ctx, d)
}

func (s *Service) GetDisputeForAdmin(ctx context.Context, disputeID uuid.UUID) (*DisputeDetail, error) {
	d, err := s.disputes.GetByID(ctx, disputeID)
	if err != nil {
		return nil, err
	}
	return s.loadDetail(ctx, d)
}

func (s *Service) loadDetail(ctx context.Context, d *disputedomain.Dispute) (*DisputeDetail, error) {
	evidence, err := s.disputes.ListEvidence(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("load evidence: %w", err)
	}
	cps, err := s.disputes.ListCounterProposals(ctx, d.ID)
	if err != nil {
		return nil, fmt.Errorf("load counter-proposals: %w", err)
	}
	return &DisputeDetail{Dispute: d, Evidence: evidence, CounterProposals: cps}, nil
}

func (s *Service) ListMyDisputes(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*disputedomain.Dispute, string, error) {
	return s.disputes.ListByUserID(ctx, userID, cursor, limit)
}
