package proposal

// PERF-B-02 — N+1 regression coverage for the participant-name lookup
// on the active-projects list.
//
// The test asserts the batch helper hits the user batch reader exactly
// once for any non-empty page of proposals, instead of issuing 2*N
// sequential GetByID calls (audit recorded ~80–200 ms p50 on top of
// the dashboard query). It also exercises the deduplication path
// (same client across multiple proposals), the empty-input fast path,
// the missing-user gracefulness, the error fallback, and the legacy
// nil-usersBatch fallback so coverage of GetParticipantNamesBatch
// stays at 100%.

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/domain/user"
)

// newProposal returns a minimal *domain.Proposal carrying just the ids
// the participant-name batch helper looks at. Avoids the full factory
// so the test is hermetic.
func newProposal(clientID, providerID uuid.UUID) *domain.Proposal {
	return &domain.Proposal{
		ID:         uuid.New(),
		ClientID:   clientID,
		ProviderID: providerID,
	}
}

func TestGetParticipantNamesBatch_SingleBatchCall(t *testing.T) {
	clientA := uuid.New()
	clientB := uuid.New()
	provA := uuid.New()
	provB := uuid.New()

	users := &mockUserRepo{
		getByIDsFn: func(_ context.Context, ids []uuid.UUID) ([]*user.User, error) {
			out := make([]*user.User, 0, len(ids))
			for _, id := range ids {
				out = append(out, &user.User{ID: id, DisplayName: "name-" + id.String()[:4]})
			}
			return out, nil
		},
	}

	svc := NewService(ServiceDeps{
		Users:      users,
		UsersBatch: users,
	})

	proposals := []*domain.Proposal{
		newProposal(clientA, provA),
		newProposal(clientB, provB),
		newProposal(clientA, provB), // dedup: clientA+provB already covered
	}

	got := svc.GetParticipantNamesBatch(context.Background(), proposals)

	require.Len(t, got, 3)
	for _, p := range proposals {
		names, ok := got[p.ID]
		require.Truef(t, ok, "proposal %s missing from result map", p.ID)
		assert.Equal(t, "name-"+p.ClientID.String()[:4], names.ClientName)
		assert.Equal(t, "name-"+p.ProviderID.String()[:4], names.ProviderName)
	}

	// Core assertion: regardless of N proposals the batch reader is
	// called exactly once. This is the N+1 regression guard.
	assert.Equal(t, 1, users.getByIDsCalls, "GetByIDs must be invoked exactly once for the page")
}

func TestGetParticipantNamesBatch_DeduplicatesUserIDs(t *testing.T) {
	sharedClient := uuid.New()
	provA := uuid.New()
	provB := uuid.New()
	provC := uuid.New()

	var capturedIDs []uuid.UUID
	users := &mockUserRepo{
		getByIDsFn: func(_ context.Context, ids []uuid.UUID) ([]*user.User, error) {
			capturedIDs = ids
			out := make([]*user.User, 0, len(ids))
			for _, id := range ids {
				out = append(out, &user.User{ID: id, DisplayName: id.String()})
			}
			return out, nil
		},
	}

	svc := NewService(ServiceDeps{Users: users, UsersBatch: users})
	proposals := []*domain.Proposal{
		newProposal(sharedClient, provA),
		newProposal(sharedClient, provB),
		newProposal(sharedClient, provC),
	}

	_ = svc.GetParticipantNamesBatch(context.Background(), proposals)

	// 1 shared client + 3 distinct providers = 4 unique ids — not 6.
	assert.Lenf(t, capturedIDs, 4,
		"GetByIDs should receive 4 unique user ids (1 shared client + 3 providers), got %d",
		len(capturedIDs),
	)
}

func TestGetParticipantNamesBatch_Empty(t *testing.T) {
	users := &mockUserRepo{}
	svc := NewService(ServiceDeps{Users: users, UsersBatch: users})

	got := svc.GetParticipantNamesBatch(context.Background(), nil)
	assert.Empty(t, got)
	assert.Equal(t, 0, users.getByIDsCalls,
		"empty input must not touch the DB")
}

func TestGetParticipantNamesBatch_MissingUsersDegradeGracefully(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New() // intentionally not returned by the batch

	users := &mockUserRepo{
		getByIDsFn: func(_ context.Context, _ []uuid.UUID) ([]*user.User, error) {
			// Return only the client — provider was deleted.
			return []*user.User{{ID: clientID, DisplayName: "client-name"}}, nil
		},
	}
	svc := NewService(ServiceDeps{Users: users, UsersBatch: users})

	p := newProposal(clientID, providerID)
	got := svc.GetParticipantNamesBatch(context.Background(), []*domain.Proposal{p})

	require.Contains(t, got, p.ID)
	assert.Equal(t, "client-name", got[p.ID].ClientName)
	assert.Empty(t, got[p.ID].ProviderName,
		"missing user must collapse to empty string, not crash")
}

func TestGetParticipantNamesBatch_BatchErrorReturnsEmptyNames(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()

	users := &mockUserRepo{
		getByIDsFn: func(_ context.Context, _ []uuid.UUID) ([]*user.User, error) {
			return nil, errors.New("transient db hiccup")
		},
	}
	svc := NewService(ServiceDeps{Users: users, UsersBatch: users})

	p := newProposal(clientID, providerID)
	got := svc.GetParticipantNamesBatch(context.Background(), []*domain.Proposal{p})

	// On error every proposal still appears in the map, with empty
	// names — the list endpoint must keep rendering.
	require.Contains(t, got, p.ID)
	assert.Empty(t, got[p.ID].ClientName)
	assert.Empty(t, got[p.ID].ProviderName)
}

func TestGetParticipantNamesBatch_NilUsersBatchFallsBackToPerIDLookup(t *testing.T) {
	clientID := uuid.New()
	providerID := uuid.New()

	getByIDCalls := 0
	users := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			getByIDCalls++
			return &user.User{ID: id, DisplayName: "fallback-" + id.String()[:4]}, nil
		},
	}
	// Deliberately omit UsersBatch — exercises the legacy fallback so
	// pre-PERF-B-02 wiring still works.
	svc := NewService(ServiceDeps{Users: users})

	p := newProposal(clientID, providerID)
	got := svc.GetParticipantNamesBatch(context.Background(), []*domain.Proposal{p})

	require.Contains(t, got, p.ID)
	assert.NotEmpty(t, got[p.ID].ClientName)
	assert.NotEmpty(t, got[p.ID].ProviderName)
	// Fallback path uses 2 GetByID calls per proposal — the slow path
	// still works, but the fast path is the wired default.
	assert.Equal(t, 2, getByIDCalls)
	assert.Equal(t, 0, users.getByIDsCalls,
		"fallback path must NOT touch the batch reader")
}

// BenchmarkGetParticipantNamesBatch_BatchVsLoop captures the win on
// the realistic page-size of 20 proposals (the default list cap).
// Each mocked DB call sleeps for a small amount that models the
// production round-trip floor (~1ms cross-AZ Postgres). The fast path
// issues 1 batch call → 1ms; the slow path issues 2*N=40 per-id
// calls → 40ms. The cumulative win on a real wallet/dashboard hit is
// where PERF-B-02 actually pays off.
//
// Without the simulated latency, the benchmark would (mis)reward the
// loop for having fewer allocs in a CPU-only test — completely
// reversing the production reality where every extra round trip
// costs milliseconds. We use a small (200µs) latency to keep the
// benchmark fast while still giving the batch path a clear advantage.
func BenchmarkGetParticipantNamesBatch_BatchVsLoop(b *testing.B) {
	const (
		pageSize     = 20
		fakeRTT      = 200 * time.Microsecond // simulate Postgres RTT
	)

	proposals := make([]*domain.Proposal, pageSize)
	for i := 0; i < pageSize; i++ {
		proposals[i] = newProposal(uuid.New(), uuid.New())
	}

	users := &mockUserRepo{
		getByIDsFn: func(_ context.Context, ids []uuid.UUID) ([]*user.User, error) {
			time.Sleep(fakeRTT)
			out := make([]*user.User, 0, len(ids))
			for _, id := range ids {
				out = append(out, &user.User{ID: id, DisplayName: "user"})
			}
			return out, nil
		},
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			time.Sleep(fakeRTT)
			return &user.User{ID: id, DisplayName: "user"}, nil
		},
	}

	b.Run("Batch", func(b *testing.B) {
		svc := NewService(ServiceDeps{Users: users, UsersBatch: users})
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = svc.GetParticipantNamesBatch(ctx, proposals)
		}
	})

	b.Run("Loop", func(b *testing.B) {
		// Wire the service WITHOUT UsersBatch so it falls back to
		// per-id lookups — the pre-PERF-B-02 behaviour.
		svc := NewService(ServiceDeps{Users: users})
		ctx := context.Background()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = svc.GetParticipantNamesBatch(ctx, proposals)
		}
	})
}
