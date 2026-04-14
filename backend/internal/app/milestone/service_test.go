package milestone

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/milestone"
)

// newMilestone is a local test helper that builds a fresh pending_funding
// milestone with a fixed proposal id for easier assertions.
func newMilestone(t *testing.T, sequence int) *domain.Milestone {
	t.Helper()
	m, err := domain.NewMilestone(domain.NewMilestoneInput{
		ProposalID:  uuid.New(),
		Sequence:    sequence,
		Title:       "Phase",
		Description: "desc",
		Amount:      10000,
	})
	if err != nil {
		t.Fatalf("NewMilestone: %v", err)
	}
	return m
}

func TestNewService_NilRepoPanics(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected NewService to panic on nil repo")
		}
	}()
	NewService(ServiceDeps{})
}

func TestService_Fund_Happy(t *testing.T) {
	m := newMilestone(t, 1)
	var updated *domain.Milestone

	repo := &mockRepo{
		getByIDForUpdateFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
		getCurrentActiveFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
		updateFn: func(_ context.Context, mm *domain.Milestone) error {
			updated = mm
			return nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	if err := svc.Fund(context.Background(), m.ID); err != nil {
		t.Fatalf("Fund: %v", err)
	}
	if updated == nil || updated.Status != domain.StatusFunded {
		t.Errorf("expected status funded, got %+v", updated)
	}
	if updated.FundedAt == nil {
		t.Error("FundedAt not set")
	}
}

func TestService_Fund_RejectsOutOfSequence(t *testing.T) {
	// current active = sequence 1, caller tries to fund milestone 2.
	m1 := newMilestone(t, 1)
	m2 := newMilestone(t, 2)
	m2.ProposalID = m1.ProposalID

	repo := &mockRepo{
		getByIDForUpdateFn: func(_ context.Context, id uuid.UUID) (*domain.Milestone, error) {
			if id == m2.ID {
				return m2, nil
			}
			return nil, domain.ErrMilestoneNotFound
		},
		getCurrentActiveFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m1, nil // m1 is still current; m2 cannot be funded
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	err := svc.Fund(context.Background(), m2.ID)
	if !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestService_Fund_PropagatesConcurrentUpdate(t *testing.T) {
	m := newMilestone(t, 1)
	repo := &mockRepo{
		getByIDForUpdateFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
		getCurrentActiveFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
		updateFn: func(_ context.Context, _ *domain.Milestone) error {
			return domain.ErrConcurrentUpdate
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	err := svc.Fund(context.Background(), m.ID)
	if !errors.Is(err, domain.ErrConcurrentUpdate) {
		t.Errorf("expected ErrConcurrentUpdate, got %v", err)
	}
}

func TestService_Submit_Happy(t *testing.T) {
	m := newMilestone(t, 1)
	_ = m.Fund() // pre-state: funded

	repo := &mockRepo{
		getByIDForUpdateFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	if err := svc.Submit(context.Background(), m.ID); err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if m.Status != domain.StatusSubmitted {
		t.Errorf("status = %q, want submitted", m.Status)
	}
}

func TestService_ApproveAndRelease_Happy(t *testing.T) {
	m := newMilestone(t, 1)
	_ = m.Fund()
	_ = m.Submit()

	repo := &mockRepo{
		getByIDForUpdateFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	if err := svc.ApproveAndRelease(context.Background(), m.ID); err != nil {
		t.Fatalf("ApproveAndRelease: %v", err)
	}
	if m.Status != domain.StatusReleased {
		t.Errorf("status = %q, want released", m.Status)
	}
	if m.ReleasedAt == nil {
		t.Error("ReleasedAt not set")
	}
}

func TestService_ApproveAndRelease_InvalidFrom(t *testing.T) {
	m := newMilestone(t, 1) // still pending_funding
	repo := &mockRepo{
		getByIDForUpdateFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	err := svc.ApproveAndRelease(context.Background(), m.ID)
	if !errors.Is(err, domain.ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestService_Reject_BackToFunded(t *testing.T) {
	m := newMilestone(t, 1)
	_ = m.Fund()
	_ = m.Submit()

	repo := &mockRepo{
		getByIDForUpdateFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	if err := svc.Reject(context.Background(), m.ID); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if m.Status != domain.StatusFunded {
		t.Errorf("status = %q, want funded", m.Status)
	}
	if m.SubmittedAt != nil {
		t.Error("Reject must clear SubmittedAt")
	}
}

func TestService_Cancel_Happy(t *testing.T) {
	m := newMilestone(t, 1)
	repo := &mockRepo{
		getByIDForUpdateFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	if err := svc.Cancel(context.Background(), m.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if m.Status != domain.StatusCancelled {
		t.Errorf("status = %q, want cancelled", m.Status)
	}
}

func TestService_CancelAllPendingFutureMilestones(t *testing.T) {
	proposalID := uuid.New()
	m1 := newMilestone(t, 1)
	m1.ProposalID = proposalID
	m1.Status = domain.StatusReleased // already terminal, must be left alone
	m2 := newMilestone(t, 2)
	m2.ProposalID = proposalID
	m2.Status = domain.StatusFunded // active (not pending), must be left alone
	m3 := newMilestone(t, 3)
	m3.ProposalID = proposalID
	// pending_funding -> will be cancelled
	m4 := newMilestone(t, 4)
	m4.ProposalID = proposalID
	// pending_funding -> will be cancelled

	all := []*domain.Milestone{m1, m2, m3, m4}
	byID := map[uuid.UUID]*domain.Milestone{
		m1.ID: m1, m2.ID: m2, m3.ID: m3, m4.ID: m4,
	}

	repo := &mockRepo{
		listByProposalFn: func(_ context.Context, _ uuid.UUID) ([]*domain.Milestone, error) {
			return all, nil
		},
		getByIDForUpdateFn: func(_ context.Context, id uuid.UUID) (*domain.Milestone, error) {
			return byID[id], nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	if err := svc.CancelAllPendingFutureMilestones(context.Background(), proposalID); err != nil {
		t.Fatalf("CancelAllPendingFutureMilestones: %v", err)
	}
	// m1 untouched (already released)
	if m1.Status != domain.StatusReleased {
		t.Errorf("m1 touched: %q", m1.Status)
	}
	// m2 untouched (funded, not pending)
	if m2.Status != domain.StatusFunded {
		t.Errorf("m2 touched: %q", m2.Status)
	}
	// m3 + m4 cancelled
	if m3.Status != domain.StatusCancelled {
		t.Errorf("m3 not cancelled: %q", m3.Status)
	}
	if m4.Status != domain.StatusCancelled {
		t.Errorf("m4 not cancelled: %q", m4.Status)
	}
}

func TestService_CancelAllPendingFutureMilestones_SwallowsConcurrent(t *testing.T) {
	proposalID := uuid.New()
	m1 := newMilestone(t, 1)
	m1.ProposalID = proposalID

	repo := &mockRepo{
		listByProposalFn: func(_ context.Context, _ uuid.UUID) ([]*domain.Milestone, error) {
			return []*domain.Milestone{m1}, nil
		},
		getByIDForUpdateFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m1, nil
		},
		updateFn: func(_ context.Context, _ *domain.Milestone) error {
			return domain.ErrConcurrentUpdate
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	// A concurrent update should not propagate — the sweep is
	// best-effort and the next pass will re-evaluate.
	if err := svc.CancelAllPendingFutureMilestones(context.Background(), proposalID); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestService_OpenDispute_Happy(t *testing.T) {
	m := newMilestone(t, 1)
	_ = m.Fund()

	repo := &mockRepo{
		getByIDForUpdateFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	disputeID := uuid.New()
	if err := svc.OpenDispute(context.Background(), m.ID, disputeID); err != nil {
		t.Fatalf("OpenDispute: %v", err)
	}
	if m.Status != domain.StatusDisputed {
		t.Errorf("status = %q, want disputed", m.Status)
	}
	if m.ActiveDisputeID == nil || *m.ActiveDisputeID != disputeID {
		t.Error("ActiveDisputeID not set")
	}
}

func TestService_RestoreFromDispute_AllTargets(t *testing.T) {
	for _, target := range []domain.MilestoneStatus{
		domain.StatusFunded, domain.StatusReleased, domain.StatusRefunded,
	} {
		t.Run(string(target), func(t *testing.T) {
			m := newMilestone(t, 1)
			_ = m.Fund()
			_ = m.OpenDispute(uuid.New())

			repo := &mockRepo{
				getByIDForUpdateFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
					return m, nil
				},
			}
			svc := NewService(ServiceDeps{Milestones: repo})

			if err := svc.RestoreFromDispute(context.Background(), m.ID, target); err != nil {
				t.Fatalf("restore: %v", err)
			}
			if m.Status != target {
				t.Errorf("status = %q, want %q", m.Status, target)
			}
		})
	}
}

func TestService_AddDeliverable_MutableStatus(t *testing.T) {
	m := newMilestone(t, 1) // pending_funding — mutable
	d, err := domain.NewDeliverable(domain.NewDeliverableInput{
		MilestoneID: m.ID,
		Filename:    "spec.pdf",
		URL:         "https://example.com/s.pdf",
		Size:        1024,
		MimeType:    "application/pdf",
		UploadedBy:  uuid.New(),
	})
	if err != nil {
		t.Fatalf("NewDeliverable: %v", err)
	}

	var stored *domain.Deliverable
	repo := &mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
		createDeliverableFn: func(_ context.Context, dd *domain.Deliverable) error {
			stored = dd
			return nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	if err := svc.AddDeliverable(context.Background(), d); err != nil {
		t.Fatalf("AddDeliverable: %v", err)
	}
	if stored == nil {
		t.Error("deliverable was not persisted")
	}
}

func TestService_AddDeliverable_LockedStatus(t *testing.T) {
	m := newMilestone(t, 1)
	m.Status = domain.StatusSubmitted // locked — cannot add

	d, _ := domain.NewDeliverable(domain.NewDeliverableInput{
		MilestoneID: m.ID,
		Filename:    "late.pdf",
		URL:         "https://example.com/l.pdf",
		Size:        1024,
		MimeType:    "application/pdf",
		UploadedBy:  uuid.New(),
	})

	repo := &mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	err := svc.AddDeliverable(context.Background(), d)
	if !errors.Is(err, domain.ErrDeliverableLocked) {
		t.Errorf("expected ErrDeliverableLocked, got %v", err)
	}
}

func TestService_DeleteDeliverable_LockedStatus(t *testing.T) {
	m := newMilestone(t, 1)
	m.Status = domain.StatusReleased // terminal — locked

	repo := &mockRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Milestone, error) {
			return m, nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	err := svc.DeleteDeliverable(context.Background(), m.ID, uuid.New())
	if !errors.Is(err, domain.ErrDeliverableLocked) {
		t.Errorf("expected ErrDeliverableLocked, got %v", err)
	}
}

func TestService_CreateBatch_DelegatesToRepo(t *testing.T) {
	var captured []*domain.Milestone
	repo := &mockRepo{
		createBatchFn: func(_ context.Context, ms []*domain.Milestone) error {
			captured = ms
			return nil
		},
	}
	svc := NewService(ServiceDeps{Milestones: repo})

	propID := uuid.New()
	batch, err := domain.NewMilestoneBatch(propID, []domain.NewMilestoneInput{
		{Sequence: 1, Title: "A", Description: "a", Amount: 1000},
		{Sequence: 2, Title: "B", Description: "b", Amount: 2000},
	})
	if err != nil {
		t.Fatalf("NewMilestoneBatch: %v", err)
	}

	if err := svc.CreateBatch(context.Background(), batch); err != nil {
		t.Fatalf("CreateBatch: %v", err)
	}
	if len(captured) != 2 {
		t.Errorf("captured %d, want 2", len(captured))
	}
}
