package dispute

// P8 commit 4 — defensive system-actor wrap on the dispute scheduler
// goroutine entry. Scheduler.Run MUST tag its root context with
// system.WithSystemActor before any repo call so the legacy
// non-tenant-aware repository code path passes the warn-if-not-
// system-actor guard from rls.go (and the future NOSUPERUSER
// NOBYPASSRLS policy USING expression).

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	disputedomain "marketplace-backend/internal/domain/dispute"
	"marketplace-backend/internal/system"
)

// TestSchedulerRun_TagsContextSystemActor uses the existing
// mockDisputeRepo to capture the ctx tag at the first repo call
// (ListPendingForScheduler) and asserts the defensive wrap inside
// Scheduler.Run kicked in even when the caller hands an un-tagged
// context.Background.
func TestSchedulerRun_TagsContextSystemActor(t *testing.T) {
	var captured bool
	repo := &mockDisputeRepo{
		listPendingFn: func(ctx context.Context) ([]*disputedomain.Dispute, error) {
			captured = system.IsSystemActor(ctx)
			return nil, nil
		},
	}
	sch := NewScheduler(SchedulerDeps{
		Disputes: repo,
	})

	// Pre-cancel the context so Run exits after the immediate-on-start
	// tick. tick() runs synchronously before the select, so the
	// captured flag is populated before Run returns.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sch.Run(ctx, 1*time.Hour) // interval doesn't matter, ctx is cancelled

	assert.True(t, captured,
		"Scheduler.Run must wrap ctx with system.WithSystemActor before the first repo touch")
}
