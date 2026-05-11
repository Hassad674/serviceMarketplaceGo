package postgres_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// stubUserRepo is a controllable repository.UserRepository the
// decorator wraps. Only the methods the decorator interacts with
// (BumpSessionVersion + anything that might fall through embedding)
// have meaningful implementations; the rest are no-ops so we still
// satisfy the full interface and the wrapper compiles cleanly.
//
// The stub uses a callable BumpFn so each test can inject a custom
// success/failure scenario without rebuilding a new struct.
type stubUserRepo struct {
	bumpCalls   []uuid.UUID
	BumpFn      func(ctx context.Context, userID uuid.UUID) (int, error)
	getByIDFn   func(ctx context.Context, id uuid.UUID) (*user.User, error)
	getByIDHits []uuid.UUID
}

func (s *stubUserRepo) Create(_ context.Context, _ *user.User) error           { return nil }
func (s *stubUserRepo) GetByEmail(_ context.Context, _ string) (*user.User, error) {
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	s.getByIDHits = append(s.getByIDHits, id)
	if s.getByIDFn != nil {
		return s.getByIDFn(ctx, id)
	}
	return nil, user.ErrUserNotFound
}
func (s *stubUserRepo) Update(_ context.Context, _ *user.User) error { return nil }
func (s *stubUserRepo) Delete(_ context.Context, _ uuid.UUID) error  { return nil }
func (s *stubUserRepo) ExistsByEmail(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (s *stubUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}
func (s *stubUserRepo) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}
func (s *stubUserRepo) CountByRole(_ context.Context) (map[string]int, error)   { return nil, nil }
func (s *stubUserRepo) CountByStatus(_ context.Context) (map[string]int, error) { return nil, nil }
func (s *stubUserRepo) RecentSignups(_ context.Context, _ int) ([]*user.User, error) {
	return nil, nil
}
func (s *stubUserRepo) BumpSessionVersion(ctx context.Context, userID uuid.UUID) (int, error) {
	s.bumpCalls = append(s.bumpCalls, userID)
	if s.BumpFn != nil {
		return s.BumpFn(ctx, userID)
	}
	return 1, nil
}
func (s *stubUserRepo) GetSessionVersion(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (s *stubUserRepo) UpdateEmailNotificationsEnabled(_ context.Context, _ uuid.UUID, _ bool) error {
	return nil
}
func (s *stubUserRepo) TouchLastActive(_ context.Context, _ uuid.UUID) error { return nil }

// stubInvalidator counts Invalidate calls and lets each test inject
// a failure to validate the wrapper's error-handling policy
// (failures must NEVER bubble up to fail the bump, only be logged).
type stubInvalidator struct {
	mu          sync.Mutex
	invalidated []uuid.UUID
	err         error
}

func (s *stubInvalidator) Invalidate(_ context.Context, userID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invalidated = append(s.invalidated, userID)
	return s.err
}

func (s *stubInvalidator) calls() []uuid.UUID {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]uuid.UUID, len(s.invalidated))
	copy(out, s.invalidated)
	return out
}

// ---------------------------------------------------------------------------
// QW-HARDENING fix #1: BumpSessionVersion success → cache.Invalidate
// must be called with the same userID. Without this test, a refactor
// that bypasses the wrapper would regress the cache eviction
// silently.
// ---------------------------------------------------------------------------

func TestInvalidatingUserRepository_BumpSuccess_TriggersInvalidate(t *testing.T) {
	t.Parallel()
	inner := &stubUserRepo{BumpFn: func(_ context.Context, _ uuid.UUID) (int, error) {
		return 42, nil
	}}
	inv := &stubInvalidator{}
	repo := postgres.NewInvalidatingUserRepository(inner, inv)

	uid := uuid.New()
	got, err := repo.BumpSessionVersion(context.Background(), uid)
	require.NoError(t, err)
	assert.Equal(t, 42, got, "wrapper must return the inner's new version")

	require.Len(t, inner.bumpCalls, 1, "inner must be called exactly once")
	assert.Equal(t, uid, inner.bumpCalls[0])

	require.Len(t, inv.calls(), 1, "invalidator must be called exactly once after success")
	assert.Equal(t, uid, inv.calls()[0], "invalidator must receive the same user id")
}

// ---------------------------------------------------------------------------
// QW-HARDENING regression test: a FAILED bump must NOT invalidate the
// cache. The DB write did not commit, so the cached version is still
// consistent — evicting would cause a useless extra round-trip on the
// next request.
// ---------------------------------------------------------------------------

func TestInvalidatingUserRepository_BumpError_SkipsInvalidate(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("transient postgres error")
	inner := &stubUserRepo{BumpFn: func(_ context.Context, _ uuid.UUID) (int, error) {
		return 0, wantErr
	}}
	inv := &stubInvalidator{}
	repo := postgres.NewInvalidatingUserRepository(inner, inv)

	got, err := repo.BumpSessionVersion(context.Background(), uuid.New())
	assert.ErrorIs(t, err, wantErr)
	assert.Equal(t, 0, got)

	require.Len(t, inner.bumpCalls, 1)
	assert.Empty(t, inv.calls(),
		"invalidator MUST NOT be called when inner bump fails — cache is still consistent")
}

func TestInvalidatingUserRepository_BumpUserNotFound_SkipsInvalidate(t *testing.T) {
	t.Parallel()
	inner := &stubUserRepo{BumpFn: func(_ context.Context, _ uuid.UUID) (int, error) {
		return 0, user.ErrUserNotFound
	}}
	inv := &stubInvalidator{}
	repo := postgres.NewInvalidatingUserRepository(inner, inv)

	_, err := repo.BumpSessionVersion(context.Background(), uuid.New())
	assert.ErrorIs(t, err, user.ErrUserNotFound)
	assert.Empty(t, inv.calls(),
		"user-not-found is not a bump success — must not invalidate")
}

// ---------------------------------------------------------------------------
// QW-HARDENING tolerance test: Invalidate FAILURES never fail the
// bump. The DB commit succeeded — the cache can heal on TTL. Failing
// here would be worse than the brief stale window.
// ---------------------------------------------------------------------------

func TestInvalidatingUserRepository_InvalidateError_DoesNotFailBump(t *testing.T) {
	t.Parallel()
	inner := &stubUserRepo{BumpFn: func(_ context.Context, _ uuid.UUID) (int, error) {
		return 7, nil
	}}
	inv := &stubInvalidator{err: errors.New("redis down")}
	repo := postgres.NewInvalidatingUserRepository(inner, inv)

	got, err := repo.BumpSessionVersion(context.Background(), uuid.New())
	require.NoError(t, err, "invalidate failure must not propagate")
	assert.Equal(t, 7, got, "wrapper must return the inner's new version even when DEL fails")
	assert.Len(t, inv.calls(), 1, "invalidator was still attempted")
}

// ---------------------------------------------------------------------------
// QW-HARDENING idempotence: invalidating a key that does not exist in
// the cache (e.g. the user's session_version was never queried) is not
// an error. The invalidator's Invalidate() contract guarantees this;
// the wrapper must not introduce extra friction.
// ---------------------------------------------------------------------------

func TestInvalidatingUserRepository_InvalidateMissingKey_NoError(t *testing.T) {
	t.Parallel()
	// Invalidator with nil err mimics "key not present, DEL returns 0
	// deleted but no error" — the production cache's Invalidate
	// behaviour.
	inner := &stubUserRepo{BumpFn: func(_ context.Context, _ uuid.UUID) (int, error) {
		return 1, nil
	}}
	inv := &stubInvalidator{}
	repo := postgres.NewInvalidatingUserRepository(inner, inv)

	_, err := repo.BumpSessionVersion(context.Background(), uuid.New())
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Constructor guards: nil inner or nil invalidator are programmer
// errors that must fail loud at wiring time, not silently regress the
// fix.
// ---------------------------------------------------------------------------

func TestInvalidatingUserRepository_NilInnerPanics(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		postgres.NewInvalidatingUserRepository(nil, &stubInvalidator{})
	})
}

func TestInvalidatingUserRepository_NilInvalidatorPanics(t *testing.T) {
	t.Parallel()
	assert.Panics(t, func() {
		postgres.NewInvalidatingUserRepository(&stubUserRepo{}, nil)
	})
}

// ---------------------------------------------------------------------------
// Embedding forwards every non-bump method without touching the
// invalidator. Anchors the Liskov contract — the wrapper must be
// substitutable for the raw repo from the caller's perspective.
// ---------------------------------------------------------------------------

func TestInvalidatingUserRepository_GetByID_ForwardsWithoutInvalidate(t *testing.T) {
	t.Parallel()
	inner := &stubUserRepo{}
	inv := &stubInvalidator{}
	repo := postgres.NewInvalidatingUserRepository(inner, inv)

	uid := uuid.New()
	_, _ = repo.GetByID(context.Background(), uid)

	assert.Equal(t, []uuid.UUID{uid}, inner.getByIDHits)
	assert.Empty(t, inv.calls(), "read methods MUST NOT trigger invalidation")
}

// ---------------------------------------------------------------------------
// Concurrency: many goroutines bumping the same user must each fire
// exactly one invalidate. The wrapper does NOT coalesce — every bump
// is an independent state mutation and must each fire its eviction.
// ---------------------------------------------------------------------------

func TestInvalidatingUserRepository_ConcurrentBumps_AllInvalidate(t *testing.T) {
	t.Parallel()
	inner := &stubUserRepo{BumpFn: func(_ context.Context, _ uuid.UUID) (int, error) {
		return 1, nil
	}}
	// The default stub mutates a slice; protect it.
	var mu sync.Mutex
	bumpRecorder := &stubUserRepo{BumpFn: func(_ context.Context, _ uuid.UUID) (int, error) {
		return 1, nil
	}}
	_ = bumpRecorder
	inv := &stubInvalidator{}
	repo := postgres.NewInvalidatingUserRepository(inner, inv)

	const N = 100
	uid := uuid.New()
	var wg sync.WaitGroup
	wg.Add(N)
	for range N {
		go func() {
			defer wg.Done()
			mu.Lock()
			_, _ = repo.BumpSessionVersion(context.Background(), uid)
			mu.Unlock()
		}()
	}
	wg.Wait()

	assert.Len(t, inner.bumpCalls, N, "every concurrent caller hits the inner bump")
	assert.Len(t, inv.calls(), N, "every concurrent bump fires its own invalidation")
}
