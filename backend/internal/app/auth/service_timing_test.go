package auth

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/user"
)

// F.5 S5 timing parity — the duplicate-email branch of Register() now
// runs a discard-bcrypt step so the wall-clock cost matches the create
// path. Without parity, an attacker could probe email existence by
// measuring response time: the duplicate path used to skip Hash()
// entirely (~10-50ms) versus the create path's ~250ms bcrypt step.
//
// We assert on the hasher invocation count, NOT on wall-clock time —
// timing assertions are flaky on shared CI runners. The structural
// invariant "Register calls Hash exactly once on every code path" is
// stronger than any statistical timing assertion: if Hash() is called,
// then both paths share the same dominant cost and the timing channel
// is closed regardless of bcrypt's exact realised cost on the CI host.

// countingHasher wraps the standard mockHasher with a goroutine-safe
// counter so tests can assert on Hash() invocations.
type countingHasher struct {
	hashCount    atomic.Int32
	compareCount atomic.Int32
}

func (h *countingHasher) Hash(password string) (string, error) {
	h.hashCount.Add(1)
	return "hashed_" + password, nil
}

func (h *countingHasher) Compare(hashed, password string) error {
	h.compareCount.Add(1)
	if hashed == "hashed_"+password {
		return nil
	}
	return user.ErrInvalidCredentials
}

// TestRegister_DuplicatePathRunsHasher ensures the F.5 S5 timing
// parity step is in place: the hasher is invoked even when the email
// is already registered, so the duplicate path's wall-clock matches
// the create path's bcrypt step.
func TestRegister_DuplicatePathRunsHasher(t *testing.T) {
	hasher := &countingHasher{}
	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}

	svc := newTestServiceWithHasher(userRepo, hasher)

	out, err := svc.Register(context.Background(), validRegisterInput())

	require.NoError(t, err)
	require.NotNil(t, out)
	require.True(t, out.SilentDuplicate, "duplicate path must return SilentDuplicate")

	// The structural assertion: even on the duplicate branch, Hash is
	// invoked exactly once. This is the timing parity guarantee.
	assert.Equal(t, int32(1), hasher.hashCount.Load(),
		"duplicate path must invoke Hash exactly once for timing parity (F.5 S5)")
}

// TestRegister_HashCountParity is the parity test: both the
// duplicate-email path AND the fresh-registration path must invoke
// Hash exactly once. Table-driven so the symmetry is obvious.
func TestRegister_HashCountParity(t *testing.T) {
	tests := []struct {
		name           string
		emailExists    bool
		wantSilentDup  bool
		wantHashCalls  int32
	}{
		{
			name:          "fresh email runs Hash once",
			emailExists:   false,
			wantSilentDup: false,
			wantHashCalls: 1,
		},
		{
			name:          "duplicate email also runs Hash once (timing parity)",
			emailExists:   true,
			wantSilentDup: true,
			wantHashCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := &countingHasher{}
			userRepo := &mockUserRepo{
				existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
					return tt.emailExists, nil
				},
			}

			svc := newTestServiceWithHasher(userRepo, hasher)

			out, err := svc.Register(context.Background(), validRegisterInput())

			require.NoError(t, err)
			require.NotNil(t, out)
			assert.Equal(t, tt.wantSilentDup, out.SilentDuplicate)
			assert.Equal(t, tt.wantHashCalls, hasher.hashCount.Load(),
				"hash invocation count must match parity expectation")
		})
	}
}

// TestRegister_DuplicatePathParityHashDoesNotPersist asserts the
// parity step's hash output is discarded — it must NOT leak into the
// returned AuthOutput, otherwise an attacker could observe its
// presence/absence to distinguish the two paths.
func TestRegister_DuplicatePathParityHashDoesNotPersist(t *testing.T) {
	hasher := &countingHasher{}
	userRepo := &mockUserRepo{
		existsByEmailFn: func(_ context.Context, _ string) (bool, error) {
			return true, nil
		},
	}

	svc := newTestServiceWithHasher(userRepo, hasher)

	out, err := svc.Register(context.Background(), validRegisterInput())

	require.NoError(t, err)
	require.NotNil(t, out)
	require.True(t, out.SilentDuplicate)

	// The parity step's hash must be discarded — the wire shape stays
	// indistinguishable from a fresh registration's neutral 202.
	assert.Nil(t, out.User, "duplicate must not leak user payload")
	assert.Empty(t, out.AccessToken, "duplicate must not leak access token")
	assert.Empty(t, out.RefreshToken, "duplicate must not leak refresh token")
}

// newTestServiceWithHasher builds a Service with a custom hasher,
// reusing the standard mocks for everything else. We keep this
// helper local to the timing tests so the broader test suite is not
// disturbed.
func newTestServiceWithHasher(userRepo *mockUserRepo, hasher *countingHasher) *Service {
	if userRepo == nil {
		userRepo = &mockUserRepo{}
	}
	resets := &mockPasswordResetRepo{}
	tokens := &mockTokenService{}
	emailSvc := &mockEmailService{}
	return NewService(userRepo, resets, hasher, tokens, emailSvc, "https://example.com")
}
