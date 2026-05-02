package proposal

// P8 commit 4 — defensive system-actor wrap on scheduler entry points.
//
// AutoApproveMilestone, FundReminderForMilestone, AutoCloseProposal
// MUST wrap their root context with system.WithSystemActor so the
// repository's legacy GetByID warn-if-not-system-actor guard from
// rls.go does not log false-positive warnings (and, once we flip to
// NOSUPERUSER NOBYPASSRLS, does not return rows-not-found by virtue
// of failing the policy USING expression).
//
// The brief mandates this wrap even though the worker.Run already
// wraps once: we want the protection to hold for ANY future caller
// (tests, ad-hoc CLI, an admin "force auto-approve" endpoint).

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/milestone"
	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/system"
)

// errSystemActorAuditHalt halts the scheduler entry point right after
// the first repo touch so we can inspect the captured tag without
// running the rest of the method.
var errSystemActorAuditHalt = errors.New("system actor audit: halt after first repo touch")

// TestAutoApproveMilestone_TagsContextSystemActor exercises the
// defensive wrap: we call AutoApproveMilestone with a vanilla
// context.Background() (i.e. NO system actor tag) and assert that
// the milestone repo's GetByID receives a ctx that IS tagged.
func TestAutoApproveMilestone_TagsContextSystemActor(t *testing.T) {
	var captured bool
	repo := &mockMilestoneRepo{
		getByIDFn: func(ctx context.Context, _ uuid.UUID) (*milestone.Milestone, error) {
			captured = system.IsSystemActor(ctx)
			return nil, errSystemActorAuditHalt
		},
	}
	svc := newTestService(&mockProposalRepo{}, nil, nil, nil)
	svc.milestones = repo

	err := svc.AutoApproveMilestone(context.Background(), uuid.New())
	require.Error(t, err) // halted by the audit-only mock, expected
	assert.True(t, captured,
		"AutoApproveMilestone must wrap ctx with system.WithSystemActor before any repo touch")
}

// TestAutoCloseProposal_TagsContextSystemActor mirrors the previous
// test for the auto-close path. AutoCloseProposal calls proposals.GetByID
// first, so we capture the tag there.
func TestAutoCloseProposal_TagsContextSystemActor(t *testing.T) {
	var captured bool
	repo := &mockProposalRepo{
		getByIDFn: func(ctx context.Context, _ uuid.UUID) (*domain.Proposal, error) {
			captured = system.IsSystemActor(ctx)
			return nil, errSystemActorAuditHalt
		},
	}
	svc := newTestService(repo, nil, nil, nil)

	err := svc.AutoCloseProposal(context.Background(), uuid.New())
	require.Error(t, err)
	assert.True(t, captured,
		"AutoCloseProposal must wrap ctx with system.WithSystemActor before any repo touch")
}

// TestFundReminderForMilestone_TagsContextSystemActor covers the
// fund-reminder path. It calls milestones.GetByID first.
func TestFundReminderForMilestone_TagsContextSystemActor(t *testing.T) {
	var captured bool
	repo := &mockMilestoneRepo{
		getByIDFn: func(ctx context.Context, _ uuid.UUID) (*milestone.Milestone, error) {
			captured = system.IsSystemActor(ctx)
			return nil, errSystemActorAuditHalt
		},
	}
	svc := newTestService(&mockProposalRepo{}, nil, nil, nil)
	svc.milestones = repo

	err := svc.FundReminderForMilestone(context.Background(), uuid.New())
	require.Error(t, err)
	assert.True(t, captured,
		"FundReminderForMilestone must wrap ctx with system.WithSystemActor before any repo touch")
}
