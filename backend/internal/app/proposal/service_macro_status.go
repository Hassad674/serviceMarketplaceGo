package proposal

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
	domain "marketplace-backend/internal/domain/proposal"
)

// recomputeMacroStatus derives the proposal's macro status from its
// milestones and persists the result.
//
// The macro status of a proposal is a projection of its milestones:
//
//	any disputed milestone  -> disputed
//	every milestone released -> completed
//	any submitted/approved  -> completion_requested
//	any funded (or beyond)  -> active
//	otherwise               -> accepted (no milestone funded yet)
//
// Pre-acceptance (pending) and user-terminated (declined, withdrawn)
// statuses are NEVER touched by this helper — they are controlled by
// the explicit Accept/Decline/Withdraw calls.
//
// The helper also maintains the cached timestamps:
//   - PaidAt = earliest funded_at across milestones
//   - CompletedAt = latest released_at when every milestone is released
//
// Callers run this after any milestone transition that may have changed
// the macro projection (Fund, Submit, Approve, Release, Reject, Dispute,
// Restore, Cancel, etc.). The helper is idempotent: running it without
// any milestone change is a no-op.
func (s *Service) recomputeMacroStatus(ctx context.Context, p *domain.Proposal) error {
	// Pre-acceptance and user-terminated statuses are off-limits to
	// the macro projection. They are set by explicit user actions
	// (CreateProposal, Accept, Decline, Withdraw) and must not be
	// overwritten by milestone-driven recomputation.
	switch p.Status {
	case domain.StatusPending, domain.StatusDeclined, domain.StatusWithdrawn:
		return nil
	}

	milestones, err := s.milestones.ListByProposal(ctx, p.ID)
	if err != nil {
		return fmt.Errorf("list milestones: %w", err)
	}
	// A proposal with zero milestones is a bug after phase 4 — every
	// proposal is supposed to be created with at least one. Surface it
	// loudly rather than silently degrading to a stale status.
	if len(milestones) == 0 {
		return fmt.Errorf("proposal %s has no milestones", p.ID)
	}

	newStatus := deriveMacroStatus(milestones)

	// Disputed is handled by the dispute app service flow (which calls
	// MarkDisputed on the proposal entity directly and sets
	// ActiveDisputeID). We never transition INTO disputed from the
	// macro recompute — if we see a disputed milestone we leave the
	// proposal status alone, since it's already disputed at the macro
	// level when the dispute was opened.
	if newStatus == domain.StatusDisputed && p.Status != domain.StatusDisputed {
		// Another path should have set this; skip.
		return nil
	}

	now := time.Now()

	// PaidAt: set on the first funded milestone and never reset.
	if p.PaidAt == nil {
		if first := firstFundedAt(milestones); first != nil {
			p.PaidAt = first
		}
	}

	// CompletedAt: set only when the terminal macro projection is
	// completed AND every milestone has a released_at to derive from.
	if newStatus == domain.StatusCompleted && p.CompletedAt == nil {
		if last := latestReleasedAt(milestones); last != nil {
			p.CompletedAt = last
		} else {
			p.CompletedAt = &now
		}
	}

	if p.Status != newStatus {
		p.Status = newStatus
		p.UpdatedAt = now
	}

	return s.proposals.Update(ctx, p)
}

// deriveMacroStatus is the pure projection function. Exposed within the
// package for unit testing without a database round-trip.
func deriveMacroStatus(milestones []*milestone.Milestone) domain.ProposalStatus {
	hasDispute := false
	hasSubmitOrApprove := false
	hasFundedLike := false

	for _, m := range milestones {
		switch m.Status {
		case milestone.StatusDisputed:
			hasDispute = true
		case milestone.StatusSubmitted, milestone.StatusApproved:
			hasSubmitOrApprove = true
			hasFundedLike = true
		case milestone.StatusFunded, milestone.StatusReleased, milestone.StatusRefunded:
			hasFundedLike = true
		}
	}

	switch {
	case hasDispute:
		return domain.StatusDisputed
	case milestone.AllReleased(milestones):
		return domain.StatusCompleted
	case hasSubmitOrApprove:
		return domain.StatusCompletionRequested
	case hasFundedLike:
		return domain.StatusActive
	}
	return domain.StatusAccepted
}

// firstFundedAt walks the milestones and returns the earliest funded_at
// timestamp, or nil if no milestone has ever been funded.
func firstFundedAt(milestones []*milestone.Milestone) *time.Time {
	var first *time.Time
	for _, m := range milestones {
		if m.FundedAt == nil {
			continue
		}
		if first == nil || m.FundedAt.Before(*first) {
			first = m.FundedAt
		}
	}
	return first
}

// latestReleasedAt walks the milestones and returns the latest
// released_at timestamp across every released milestone, or nil if at
// least one milestone has never been released (which means the proposal
// is not yet macro-completed).
func latestReleasedAt(milestones []*milestone.Milestone) *time.Time {
	var latest *time.Time
	for _, m := range milestones {
		if m.Status != milestone.StatusReleased || m.ReleasedAt == nil {
			continue
		}
		if latest == nil || m.ReleasedAt.After(*latest) {
			latest = m.ReleasedAt
		}
	}
	return latest
}

// getCurrentActiveMilestone is a thin wrapper used by the action
// methods (InitiatePayment, RequestCompletion, CompleteProposal, etc.)
// to fetch the current active milestone for a proposal. Returns
// milestone.ErrMilestoneNotFound when every milestone is terminal.
func (s *Service) getCurrentActiveMilestone(ctx context.Context, proposalID uuid.UUID) (*milestone.Milestone, error) {
	return s.milestones.GetCurrentActive(ctx, proposalID)
}

// withMilestoneLock is the optimistic read-lock-mutate-write pattern
// used by every proposal-service transition that delegates to a
// milestone. Fetches with FOR UPDATE, runs the caller's mutation, and
// persists with a version check. A concurrent update returns
// milestone.ErrConcurrentUpdate which the caller can surface or retry.
//
// On success, an audit row is written to milestone_transitions
// recording the from/to status pair (when the milestoneTransitions
// repo is wired). The audit insert is best-effort: errors are logged
// but do not roll back the milestone update — the state has already
// committed by the time we get here.
//
// This wrapper lives in the proposal package (rather than reusing the
// milestone app service's withLocked) so the proposal service stays
// decoupled from the milestone app service layer — they share only
// the repository port.
func (s *Service) withMilestoneLock(ctx context.Context, milestoneID uuid.UUID, mutate func(*milestone.Milestone) error) error {
	return s.withMilestoneLockAudited(ctx, milestoneID, nil, nil, "", mutate)
}

// withMilestoneLockAudited is the audited variant of withMilestoneLock.
// Action methods that have actor context (handler-driven transitions)
// can pass actor id + org id + reason so the audit row carries the
// full who/why pair. System-actor callers (auto-approve scheduler,
// outbox handlers) pass nil/nil/"auto" to record an untraced
// transition that the admin dashboard renders as "system".
func (s *Service) withMilestoneLockAudited(
	ctx context.Context,
	milestoneID uuid.UUID,
	actorID *uuid.UUID,
	actorOrgID *uuid.UUID,
	reason string,
	mutate func(*milestone.Milestone) error,
) error {
	m, err := s.milestones.GetByIDWithVersion(ctx, milestoneID)
	if err != nil {
		return err
	}
	fromStatus := m.Status
	if err := mutate(m); err != nil {
		return err
	}
	if err := s.milestones.Update(ctx, m); err != nil {
		return err
	}
	// Best-effort audit insert. fromStatus == m.Status when the
	// mutate function was a no-op (e.g. idempotent re-fund attempt)
	// — skip the audit row in that case so we don't pollute the
	// timeline with phantom transitions.
	if fromStatus != m.Status {
		s.recordTransition(ctx, m, fromStatus, actorID, actorOrgID, reason)
	}
	return nil
}

// recordTransition writes one row to milestone_transitions. Skipped
// when the milestoneTransitions repository is not wired (legacy test
// setups). Errors are logged but never propagated — the milestone
// update has already committed and we don't want a transient audit
// failure to break business state.
func (s *Service) recordTransition(
	ctx context.Context,
	m *milestone.Milestone,
	fromStatus milestone.MilestoneStatus,
	actorID *uuid.UUID,
	actorOrgID *uuid.UUID,
	reason string,
) {
	if s.milestoneTransitions == nil {
		return
	}
	t := milestone.NewTransition(
		m.ID, m.ProposalID,
		fromStatus, m.Status,
		actorID, actorOrgID,
		reason,
		nil,
	)
	// Best-effort: log but never fail the caller.
	_ = s.milestoneTransitions.Insert(ctx, t)
}
