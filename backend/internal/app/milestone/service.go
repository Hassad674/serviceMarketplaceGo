// Package milestone is the application-layer service for milestone
// operations. It orchestrates the milestone domain with the
// persistence, payment, notification, messaging, and scheduler ports.
//
// A milestone is a sub-aggregate of a proposal; this package is split
// out for code organisation but conceptually belongs to the same
// feature. The proposal app service injects this service as a
// collaborator (not via the feature-isolation wall — milestones cannot
// be removed without also removing proposals).
package milestone

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/milestone"
	"marketplace-backend/internal/port/repository"
)

// ServiceDeps is the constructor input for the milestone service.
// Every port is optional except the repository — a nil PaymentProcessor
// makes the service run in "simulate" mode (used by tests and local dev
// when Stripe is not configured).
type ServiceDeps struct {
	Milestones repository.MilestoneRepository
	// Payments is optional. When nil, Fund/Release skip the Stripe side
	// effects and only update the domain state. Production always wires
	// a real implementation.
	// Payments service.PaymentProcessor  // wired in phase 7 (outbox)
}

// Service exposes high-level milestone operations for the proposal
// app service and the phase-5 milestone handler. Every mutation goes
// through the optimistic-locked Update on the repository and returns
// a typed domain error on conflict.
type Service struct {
	repo repository.MilestoneRepository
}

// NewService builds a milestone service. Panics if Milestones is nil —
// it is the only non-optional dependency.
func NewService(deps ServiceDeps) *Service {
	if deps.Milestones == nil {
		panic("milestone.NewService: Milestones repository is required")
	}
	return &Service{repo: deps.Milestones}
}

// CreateBatch persists a set of milestones for a proposal in a single
// transaction. Callers must build the slice via milestone.NewMilestoneBatch
// so sequence/count invariants are enforced at the domain level before
// touching the DB.
func (s *Service) CreateBatch(ctx context.Context, milestones []*milestone.Milestone) error {
	return s.repo.CreateBatch(ctx, milestones)
}

// ListByProposal returns every milestone of a proposal, ordered by sequence.
func (s *Service) ListByProposal(ctx context.Context, proposalID uuid.UUID) ([]*milestone.Milestone, error) {
	return s.repo.ListByProposal(ctx, proposalID)
}

// GetByID fetches a milestone by its id without taking a lock.
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*milestone.Milestone, error) {
	return s.repo.GetByID(ctx, id)
}

// GetCurrentActive returns the lowest-sequence non-terminal milestone
// of a proposal — the one whose CTA is currently shown in the UI.
// Returns milestone.ErrMilestoneNotFound when all milestones are terminal.
func (s *Service) GetCurrentActive(ctx context.Context, proposalID uuid.UUID) (*milestone.Milestone, error) {
	return s.repo.GetCurrentActive(ctx, proposalID)
}

// Fund transitions a milestone from pending_funding to funded. Called
// after the client's payment intent has been captured and the escrow is
// held on the platform account.
//
// Enforces the strict sequential rule: Fund is only legal on the current
// active milestone. A request to fund a milestone whose sequence is
// higher than the current active one returns milestone.ErrInvalidStatus.
func (s *Service) Fund(ctx context.Context, milestoneID uuid.UUID) error {
	return s.withLocked(ctx, milestoneID, func(m *milestone.Milestone) error {
		if err := s.assertIsCurrentActive(ctx, m); err != nil {
			return err
		}
		return m.Fund()
	})
}

// Submit transitions a milestone from funded to submitted. Called by
// the provider when the deliverables are ready for client review.
func (s *Service) Submit(ctx context.Context, milestoneID uuid.UUID) error {
	return s.withLocked(ctx, milestoneID, func(m *milestone.Milestone) error {
		return m.Submit()
	})
}

// Approve transitions a milestone from submitted to approved. The
// immediate Release is NOT done here — the app layer can choose to
// call Release separately (which is what happens after an explicit
// client approval or auto-approval).
func (s *Service) Approve(ctx context.Context, milestoneID uuid.UUID) error {
	return s.withLocked(ctx, milestoneID, func(m *milestone.Milestone) error {
		return m.Approve()
	})
}

// ApproveAndRelease is the common happy-path: client approves the
// submitted work, escrow is released to the provider. Runs both
// transitions in a single locked update so there is no "approved but
// not released" window visible to concurrent readers.
func (s *Service) ApproveAndRelease(ctx context.Context, milestoneID uuid.UUID) error {
	return s.withLocked(ctx, milestoneID, func(m *milestone.Milestone) error {
		if err := m.Approve(); err != nil {
			return err
		}
		return m.Release()
	})
}

// Reject transitions a submitted milestone back to funded. Called by
// the client when the delivered work needs revisions — the provider
// can then Submit() again after applying corrections.
func (s *Service) Reject(ctx context.Context, milestoneID uuid.UUID) error {
	return s.withLocked(ctx, milestoneID, func(m *milestone.Milestone) error {
		return m.Reject()
	})
}

// Release transitions an approved milestone to released. Called
// separately from Approve in flows where the approve happens first
// (e.g. auto-approval by the scheduler) and the release comes later
// via an outbox event that triggers the Stripe transfer.
func (s *Service) Release(ctx context.Context, milestoneID uuid.UUID) error {
	return s.withLocked(ctx, milestoneID, func(m *milestone.Milestone) error {
		return m.Release()
	})
}

// Cancel transitions a pending_funding milestone to cancelled. Called
// when the proposal is closed at a boundary or auto-closes after the
// client fails to fund in time. Only pending_funding milestones can be
// cancelled — funded work goes through a dispute resolution path.
func (s *Service) Cancel(ctx context.Context, milestoneID uuid.UUID) error {
	return s.withLocked(ctx, milestoneID, func(m *milestone.Milestone) error {
		return m.Cancel()
	})
}

// CancelAllPendingFutureMilestones cancels every pending_funding
// milestone of a proposal in a single pass. Used by the proposal
// cancellation flow and the auto-close scheduler handler.
//
// Already-terminal milestones are left untouched. Active milestones
// (funded, submitted, approved, disputed) are NOT cancelled — those
// must go through their own resolution path.
func (s *Service) CancelAllPendingFutureMilestones(ctx context.Context, proposalID uuid.UUID) error {
	milestones, err := s.repo.ListByProposal(ctx, proposalID)
	if err != nil {
		return fmt.Errorf("list milestones: %w", err)
	}
	for _, m := range milestones {
		if m.Status != milestone.StatusPendingFunding {
			continue
		}
		if err := s.Cancel(ctx, m.ID); err != nil {
			// Swallow concurrent-update errors on a best-effort
			// sweep: the concurrent update means someone else is
			// already acting on this milestone, the next pass will
			// either find it cancelled or skip it.
			if errors.Is(err, milestone.ErrConcurrentUpdate) {
				continue
			}
			return fmt.Errorf("cancel milestone %s: %w", m.ID, err)
		}
	}
	return nil
}

// OpenDispute transitions a funded or submitted milestone into the
// disputed state and records the dispute id. Called by the dispute
// app service after it has persisted the dispute row.
func (s *Service) OpenDispute(ctx context.Context, milestoneID, disputeID uuid.UUID) error {
	return s.withLocked(ctx, milestoneID, func(m *milestone.Milestone) error {
		return m.OpenDispute(disputeID)
	})
}

// RestoreFromDispute transitions a disputed milestone to one of the
// legal targets (funded, released, refunded). Called by the dispute
// resolution service with the decided outcome.
func (s *Service) RestoreFromDispute(ctx context.Context, milestoneID uuid.UUID, target milestone.MilestoneStatus) error {
	return s.withLocked(ctx, milestoneID, func(m *milestone.Milestone) error {
		return m.RestoreFromDispute(target)
	})
}

// AddDeliverable registers a file attached to a milestone. Enforces
// the mutability rule: deliverables can only be added when the
// milestone is in pending_funding or funded. Once submitted (or any
// later state), deliverables are frozen as evidence.
func (s *Service) AddDeliverable(ctx context.Context, d *milestone.Deliverable) error {
	m, err := s.repo.GetByID(ctx, d.MilestoneID)
	if err != nil {
		return err
	}
	if !milestone.IsMutableStatus(m.Status) {
		return milestone.ErrDeliverableLocked
	}
	return s.repo.CreateDeliverable(ctx, d)
}

// ListDeliverables returns every deliverable attached to a milestone.
func (s *Service) ListDeliverables(ctx context.Context, milestoneID uuid.UUID) ([]*milestone.Deliverable, error) {
	return s.repo.ListDeliverables(ctx, milestoneID)
}

// DeleteDeliverable removes a deliverable by id. Enforces the same
// mutability rule as AddDeliverable: only allowed while the milestone
// is still mutable.
func (s *Service) DeleteDeliverable(ctx context.Context, milestoneID, deliverableID uuid.UUID) error {
	m, err := s.repo.GetByID(ctx, milestoneID)
	if err != nil {
		return err
	}
	if !milestone.IsMutableStatus(m.Status) {
		return milestone.ErrDeliverableLocked
	}
	return s.repo.DeleteDeliverable(ctx, deliverableID)
}

// withLocked is the common read-fetch-mutate-write pattern used by
// every transition method. It fetches the current version, runs the
// caller's mutation, and persists with an optimistic version check.
//
// "Locked" in the name is historical — there is no DB-level lock
// (BUG-11): protection comes from Update's optimistic version check
// (`WHERE id = $1 AND version = $2`). Callers should treat
// ErrConcurrentUpdate as a transient error and retry once from the
// top (including GetByIDWithVersion) — a second conflict in a tight
// loop indicates a hot contention point and is surfaced to the user.
func (s *Service) withLocked(ctx context.Context, milestoneID uuid.UUID, mutate func(*milestone.Milestone) error) error {
	m, err := s.repo.GetByIDWithVersion(ctx, milestoneID)
	if err != nil {
		return err
	}
	if err := mutate(m); err != nil {
		return err
	}
	return s.repo.Update(ctx, m)
}

// assertIsCurrentActive enforces the strict sequential rule: a
// transition out of pending_funding is only legal if this milestone
// is the current active one (lowest-sequence non-terminal).
//
// Called by Fund to prevent funding M3 while M2 is still pending.
// Submit/Approve/Release don't need this check: their preconditions
// on status already imply the milestone is active (funded/submitted).
func (s *Service) assertIsCurrentActive(ctx context.Context, m *milestone.Milestone) error {
	current, err := s.repo.GetCurrentActive(ctx, m.ProposalID)
	if err != nil {
		return err
	}
	if current.ID != m.ID {
		return milestone.ErrInvalidStatus
	}
	return nil
}
