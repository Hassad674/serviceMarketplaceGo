package milestone

import (
	"context"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/milestone"
)

// mockRepo is a hand-written stub of the MilestoneRepository port,
// following the same pattern as the proposal app service tests (field
// functions + zero-value fallback).
type mockRepo struct {
	createBatchFn       func(ctx context.Context, milestones []*domain.Milestone) error
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*domain.Milestone, error)
	getByIDWithVersionFn  func(ctx context.Context, id uuid.UUID) (*domain.Milestone, error)
	listByProposalFn    func(ctx context.Context, proposalID uuid.UUID) ([]*domain.Milestone, error)
	getCurrentActiveFn  func(ctx context.Context, proposalID uuid.UUID) (*domain.Milestone, error)
	updateFn            func(ctx context.Context, m *domain.Milestone) error
	createDeliverableFn func(ctx context.Context, d *domain.Deliverable) error
	listDeliverablesFn  func(ctx context.Context, milestoneID uuid.UUID) ([]*domain.Deliverable, error)
	deleteDeliverableFn func(ctx context.Context, id uuid.UUID) error
	listByProposalsFn   func(ctx context.Context, proposalIDs []uuid.UUID) (map[uuid.UUID][]*domain.Milestone, error)
}

func (m *mockRepo) CreateBatch(ctx context.Context, milestones []*domain.Milestone) error {
	if m.createBatchFn != nil {
		return m.createBatchFn(ctx, milestones)
	}
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Milestone, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, domain.ErrMilestoneNotFound
}

// GetByIDForOrg delegates to GetByID so existing tests keep
// working after the org-aware caller migration.
func (m *mockRepo) GetByIDForOrg(ctx context.Context, id, _ uuid.UUID) (*domain.Milestone, error) {
	return m.GetByID(ctx, id)
}

func (m *mockRepo) GetByIDWithVersion(ctx context.Context, id uuid.UUID) (*domain.Milestone, error) {
	if m.getByIDWithVersionFn != nil {
		return m.getByIDWithVersionFn(ctx, id)
	}
	// Default: delegate to GetByID so happy-path tests don't need to
	// wire two separate functions.
	return m.GetByID(ctx, id)
}

func (m *mockRepo) ListByProposal(ctx context.Context, proposalID uuid.UUID) ([]*domain.Milestone, error) {
	if m.listByProposalFn != nil {
		return m.listByProposalFn(ctx, proposalID)
	}
	return nil, nil
}

func (m *mockRepo) GetCurrentActive(ctx context.Context, proposalID uuid.UUID) (*domain.Milestone, error) {
	if m.getCurrentActiveFn != nil {
		return m.getCurrentActiveFn(ctx, proposalID)
	}
	return nil, domain.ErrMilestoneNotFound
}

func (m *mockRepo) Update(ctx context.Context, mm *domain.Milestone) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, mm)
	}
	return nil
}

func (m *mockRepo) CreateDeliverable(ctx context.Context, d *domain.Deliverable) error {
	if m.createDeliverableFn != nil {
		return m.createDeliverableFn(ctx, d)
	}
	return nil
}

func (m *mockRepo) ListDeliverables(ctx context.Context, milestoneID uuid.UUID) ([]*domain.Deliverable, error) {
	if m.listDeliverablesFn != nil {
		return m.listDeliverablesFn(ctx, milestoneID)
	}
	return nil, nil
}

func (m *mockRepo) DeleteDeliverable(ctx context.Context, id uuid.UUID) error {
	if m.deleteDeliverableFn != nil {
		return m.deleteDeliverableFn(ctx, id)
	}
	return nil
}

func (m *mockRepo) ListByProposals(ctx context.Context, proposalIDs []uuid.UUID) (map[uuid.UUID][]*domain.Milestone, error) {
	if m.listByProposalsFn != nil {
		return m.listByProposalsFn(ctx, proposalIDs)
	}
	return map[uuid.UUID][]*domain.Milestone{}, nil
}
