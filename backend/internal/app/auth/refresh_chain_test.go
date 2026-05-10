package auth

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

// Tests for B.8 — refresh-token rotation hardening.
//
// The behaviours covered here are NEW in B.8:
//   - Family invalidation on reuse detection (descendant JTIs blacklisted).
//   - Chain depth cap (MaxRefreshChainDepth) → 401.
//   - Family absolute age cap (MaxFamilyAge) → 401.
//   - Audit row content on theft + chain-limit rejection.
//
// They run alongside the SEC-06 / F.5 baseline tests in
// refresh_rotation_test.go.

// --- Family invalidation on reuse detection ---------------------------------

func TestAuthService_RefreshToken_ReuseInvalidatesEntireFamily(t *testing.T) {
	// B.8: when a blacklisted refresh token is replayed, every descendant
	// JTI tracked under the family root must be blacklisted too — so any
	// in-flight rotated token in the chain (the legitimate user's freshest
	// token, for instance) is also dead. Without this, only the replayed
	// token returns 401 — the user holding the freshest descendant keeps
	// rotating successfully, none the wiser.
	svc, blacklist, audits, u, tokens := newRotationService(t)

	familyRoot := "jti-root-1"
	descendant1 := "jti-child-1"
	descendant2 := "jti-child-2"

	// Register two descendants under the family.
	require.NoError(t, blacklist.AddFamilyMember(context.Background(), familyRoot, descendant1, time.Hour))
	require.NoError(t, blacklist.AddFamilyMember(context.Background(), familyRoot, descendant2, time.Hour))
	// Also pre-blacklist the replayed JTI to simulate a reuse attempt
	// (the previous rotation already blacklisted it).
	require.NoError(t, blacklist.Add(context.Background(), familyRoot, time.Hour))

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:        u.ID,
			JTI:           familyRoot,
			FamilyRootJTI: familyRoot,
			FamilyRootIAT: time.Now().Add(-time.Hour),
			ChainDepth:    0,
			ExpiresAt:     time.Now().Add(time.Hour),
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "replayed_token")
	assert.ErrorIs(t, err, user.ErrUnauthorized)
	assert.Nil(t, out)

	// Both descendants must now be on the per-jti blacklist.
	for _, jti := range []string{descendant1, descendant2} {
		has, herr := blacklist.Has(context.Background(), jti)
		require.NoError(t, herr)
		assert.True(t, has, "descendant %q must be blacklisted after family invalidation", jti)
	}
	// The family set itself must be gone.
	assert.Equal(t, 0, blacklist.FamilyCount(),
		"family set must be deleted after invalidation")

	// Audit row must carry family_root_jti + descendants_invalidated_count.
	var reuse *audit.Entry
	for _, e := range audits.Snapshot() {
		if e.Action == audit.ActionTokenReuseDetected {
			reuse = e
			break
		}
	}
	require.NotNil(t, reuse, "reuse must emit token_reuse_detected")
	assert.Equal(t, familyRoot, reuse.Metadata["family_root_jti"])
	assert.Equal(t, familyRoot, reuse.Metadata["compromised_jti"])
	// 2 descendants were tracked + the root was blacklisted; we count
	// descendants only (root counted via the per-jti blacklist before).
	count, ok := reuse.Metadata["descendants_invalidated_count"].(int)
	require.True(t, ok, "descendants_invalidated_count must be int")
	assert.Equal(t, 2, count)
}

func TestAuthService_RefreshToken_ReuseWithEmptyFamilyStillBumpsSession(t *testing.T) {
	// Edge case: a reuse on a legacy token (no FamilyRootJTI claim).
	// We cannot iterate descendants — the family set does not exist.
	// session_version must still be bumped so the access tokens die.
	svc, blacklist, _, u, tokens := newRotationService(t)
	users := svc.users.(*mockUserRepo)

	require.NoError(t, blacklist.Add(context.Background(), "jti-legacy", time.Hour))

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:    u.ID,
			JTI:       "jti-legacy",
			ExpiresAt: time.Now().Add(time.Hour),
			// FamilyRootJTI / IAT / ChainDepth all zero — legacy.
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "replayed")
	assert.ErrorIs(t, err, user.ErrUnauthorized)
	assert.Nil(t, out)
	assert.Len(t, users.snapshotBumpCalls(), 1,
		"legacy reuse must still bump session_version")
}

// --- Chain depth cap --------------------------------------------------------

func TestAuthService_RefreshToken_ChainDepthExceededReturnsUnauthorized(t *testing.T) {
	// B.8: the 1001st rotation in a single chain is refused. The user is
	// forced to re-login. No new pair is issued; an audit row records
	// the rejection.
	svc, _, audits, u, tokens := newRotationService(t)

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:        u.ID,
			JTI:           "jti-deep",
			FamilyRootJTI: "jti-root",
			FamilyRootIAT: time.Now().Add(-time.Hour),
			ChainDepth:    MaxRefreshChainDepth, // already at cap → next rotation refused
			ExpiresAt:     time.Now().Add(time.Hour),
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "deep_token")
	assert.ErrorIs(t, err, user.ErrUnauthorized)
	assert.Nil(t, out)

	// Audit must record the rejection with reason = chain_depth_exceeded.
	var rejection *audit.Entry
	for _, e := range audits.Snapshot() {
		if e.Action == audit.ActionTokenRefresh && e.Metadata["outcome"] == "rejected" {
			rejection = e
			break
		}
	}
	require.NotNil(t, rejection, "chain-depth rejection must emit a rejected token_refresh row")
	assert.Equal(t, "chain_depth_exceeded", rejection.Metadata["reason"])
	assert.Equal(t, MaxRefreshChainDepth, rejection.Metadata["chain_depth"])
}

func TestAuthService_RefreshToken_ChainDepthBelowCapAccepted(t *testing.T) {
	// Sanity: a chain at depth 999 still works (we hit the cap at 1000
	// only). Off-by-one guard.
	svc, _, _, u, tokens := newRotationService(t)

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:        u.ID,
			JTI:           "jti-deep-1",
			FamilyRootJTI: "jti-root",
			FamilyRootIAT: time.Now().Add(-time.Hour),
			ChainDepth:    MaxRefreshChainDepth - 1,
			ExpiresAt:     time.Now().Add(time.Hour),
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "deep_token")
	require.NoError(t, err)
	require.NotNil(t, out)
}

// --- Family absolute-age cap -----------------------------------------------

func TestAuthService_RefreshToken_FamilyAgeExceededReturnsUnauthorized(t *testing.T) {
	// B.8: even with rotation, the absolute lifetime of the family is
	// capped at MaxFamilyAge from initial login. Beyond that, the user
	// must re-authenticate even if individual tokens are still fresh.
	svc, _, audits, u, tokens := newRotationService(t)

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:        u.ID,
			JTI:           "jti-old",
			FamilyRootJTI: "jti-root-old",
			FamilyRootIAT: time.Now().Add(-MaxFamilyAge - time.Hour), // 31d ago
			ChainDepth:    50,
			ExpiresAt:     time.Now().Add(time.Hour),
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "old_chain_token")
	assert.ErrorIs(t, err, user.ErrUnauthorized)
	assert.Nil(t, out)

	var rejection *audit.Entry
	for _, e := range audits.Snapshot() {
		if e.Action == audit.ActionTokenRefresh && e.Metadata["outcome"] == "rejected" {
			rejection = e
			break
		}
	}
	require.NotNil(t, rejection, "family-age rejection must emit a rejected token_refresh row")
	assert.Equal(t, "family_age_exceeded", rejection.Metadata["reason"])
}

func TestAuthService_RefreshToken_FamilyAgeBelowCapAccepted(t *testing.T) {
	// Sanity: a 29-day-old chain still works.
	svc, _, _, u, tokens := newRotationService(t)

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:        u.ID,
			JTI:           "jti-mid",
			FamilyRootJTI: "jti-root-mid",
			FamilyRootIAT: time.Now().Add(-29 * 24 * time.Hour),
			ChainDepth:    50,
			ExpiresAt:     time.Now().Add(time.Hour),
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "mid_chain_token")
	require.NoError(t, err)
	require.NotNil(t, out)
}

func TestAuthService_RefreshToken_LegacyTokenWithoutFamilyAccepted(t *testing.T) {
	// Backwards compat: tokens minted before B.8 carry no FamilyRootIAT.
	// They MUST still rotate (otherwise B.8 deploy logs every user out).
	// On the next rotation the JWT adapter re-roots them, so the cap
	// kicks in from there.
	svc, _, _, u, tokens := newRotationService(t)

	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		return &service.TokenClaims{
			UserID:    u.ID,
			JTI:       "jti-legacy",
			ExpiresAt: time.Now().Add(time.Hour),
			// No FamilyRootJTI / IAT — legacy.
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "legacy_token")
	require.NoError(t, err)
	require.NotNil(t, out)
}

// --- evaluateChainLimits unit tests ----------------------------------------

func TestEvaluateChainLimits_TableDriven(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		claims   *service.TokenClaims
		wantRej  bool
		wantCode string
	}{
		{
			name:    "nil claims accepted",
			claims:  nil,
			wantRej: false,
		},
		{
			name: "zero depth and zero iat accepted",
			claims: &service.TokenClaims{
				ChainDepth: 0,
			},
			wantRej: false,
		},
		{
			name: "depth at cap rejected",
			claims: &service.TokenClaims{
				ChainDepth:    MaxRefreshChainDepth,
				FamilyRootIAT: now.Add(-time.Hour),
			},
			wantRej:  true,
			wantCode: "chain_depth_exceeded",
		},
		{
			name: "depth one below cap accepted",
			claims: &service.TokenClaims{
				ChainDepth:    MaxRefreshChainDepth - 1,
				FamilyRootIAT: now.Add(-time.Hour),
			},
			wantRej: false,
		},
		{
			name: "family older than cap rejected",
			claims: &service.TokenClaims{
				ChainDepth:    1,
				FamilyRootIAT: now.Add(-MaxFamilyAge - time.Second),
			},
			wantRej:  true,
			wantCode: "family_age_exceeded",
		},
		{
			name: "family exactly at cap rejected",
			claims: &service.TokenClaims{
				ChainDepth:    1,
				FamilyRootIAT: now.Add(-MaxFamilyAge),
			},
			wantRej:  true,
			wantCode: "family_age_exceeded",
		},
		{
			name: "depth wins when both caps tripped",
			claims: &service.TokenClaims{
				ChainDepth:    MaxRefreshChainDepth + 5,
				FamilyRootIAT: now.Add(-MaxFamilyAge - time.Hour),
			},
			wantRej:  true,
			wantCode: "chain_depth_exceeded",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := evaluateChainLimits(tc.claims, now)
			assert.Equal(t, tc.wantRej, d.rejected)
			if tc.wantRej {
				assert.Equal(t, tc.wantCode, d.reason)
			}
		})
	}
}

// --- Family-tracking on successful rotation --------------------------------

func TestAuthService_RefreshToken_RotationTracksNewJTIInFamily(t *testing.T) {
	// On every rotation, the new JTI must be appended to the family
	// set so a future reuse-detection can iterate descendants.
	svc, blacklist, _, u, tokens := newRotationService(t)

	familyRoot := "jti-fresh-root"
	familyIAT := time.Now().Add(-2 * time.Hour)
	parentJTI := "jti-parent"
	newJTI := "jti-fresh-child"

	// First call validates the parent (rotation context); second call
	// validates the just-issued new token to read its JTI for tracking.
	calls := 0
	tokens.validateRefreshFn = func(_ string) (*service.TokenClaims, error) {
		calls++
		if calls == 1 {
			return &service.TokenClaims{
				UserID:        u.ID,
				JTI:           parentJTI,
				FamilyRootJTI: familyRoot,
				FamilyRootIAT: familyIAT,
				ChainDepth:    3,
				ExpiresAt:     time.Now().Add(time.Hour),
			}, nil
		}
		return &service.TokenClaims{
			UserID:        u.ID,
			JTI:           newJTI,
			FamilyRootJTI: familyRoot,
			FamilyRootIAT: familyIAT,
			ChainDepth:    4,
			ExpiresAt:     time.Now().Add(time.Hour),
		}, nil
	}

	out, err := svc.RefreshToken(context.Background(), "parent_token")
	require.NoError(t, err)
	require.NotNil(t, out)

	members, err := blacklist.FamilyMembers(context.Background(), familyRoot)
	require.NoError(t, err)
	assert.Contains(t, members, newJTI,
		"new JTI must be tracked under the family root after rotation")
}

func TestAuthService_RefreshToken_LineagePropagatedOnRotation(t *testing.T) {
	// B.8 contract: the new refresh token must inherit FamilyRootJTI +
	// FamilyRootIAT verbatim and increment ChainDepth by 1.
	svc, _, _, u, tokens := newRotationService(t)

	parentRoot := "jti-root-77"
	parentIAT := time.Now().Add(-3 * time.Hour)

	captured := service.RefreshTokenInput{}
	tokens.generateRefreshLineageFn = func(input service.RefreshTokenInput) (string, error) {
		captured = input
		return "issued_token", nil
	}
	tokens.validateRefreshFn = func(token string) (*service.TokenClaims, error) {
		// First call (parent) returns the parent claims; the second
		// call (new-token validation for tracking) returns matching
		// lineage.
		if token == "issued_token" {
			return &service.TokenClaims{
				UserID:        u.ID,
				JTI:           uuid.New().String(),
				FamilyRootJTI: parentRoot,
				FamilyRootIAT: parentIAT,
				ChainDepth:    8,
				ExpiresAt:     time.Now().Add(time.Hour),
			}, nil
		}
		return &service.TokenClaims{
			UserID:        u.ID,
			JTI:           "jti-parent-77",
			FamilyRootJTI: parentRoot,
			FamilyRootIAT: parentIAT,
			ChainDepth:    7,
			ExpiresAt:     time.Now().Add(time.Hour),
		}, nil
	}

	_, err := svc.RefreshToken(context.Background(), "parent")
	require.NoError(t, err)

	assert.Equal(t, parentRoot, captured.FamilyRootJTI)
	assert.Equal(t, parentIAT.Unix(), captured.FamilyRootIAT.Unix())
	assert.Equal(t, 8, captured.ChainDepth, "depth must be parent depth + 1")
	assert.Equal(t, u.ID, captured.UserID)
}
