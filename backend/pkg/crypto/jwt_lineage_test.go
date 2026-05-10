package crypto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/port/service"
)

// Tests for B.8 — JWT adapter family lineage encoding.
//
// Coverage:
//   - Fresh family: GenerateRefreshToken seeds FamilyRootJTI = own JTI,
//     ChainDepth = 0, FamilyRootIAT = now.
//   - Rotation: GenerateRefreshTokenWithLineage copies inputs verbatim
//     (FamilyRootJTI, ChainDepth, FamilyRootIAT) and assigns a fresh JTI.
//   - Round-trip: ValidateRefreshToken returns the same lineage values.

func TestJWT_GenerateRefreshToken_SeedsFreshFamily(t *testing.T) {
	svc := NewJWTService("secret-b8-fresh", time.Hour, 24*time.Hour)
	uid := uuid.New()

	token, err := svc.GenerateRefreshToken(uid)
	require.NoError(t, err)

	claims, err := svc.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.NotEmpty(t, claims.JTI)
	assert.Equal(t, claims.JTI, claims.FamilyRootJTI,
		"fresh family must self-root: FamilyRootJTI == JTI")
	assert.Equal(t, 0, claims.ChainDepth, "fresh family starts at depth 0")
	assert.WithinDuration(t, time.Now(), claims.FamilyRootIAT, 5*time.Second,
		"FamilyRootIAT must be near now on fresh issuance")
}

func TestJWT_GenerateRefreshTokenWithLineage_CopiesInput(t *testing.T) {
	svc := NewJWTService("secret-b8-lineage", time.Hour, 24*time.Hour)
	uid := uuid.New()
	rootJTI := uuid.New().String()
	rootIAT := time.Now().Add(-3 * time.Hour).UTC().Truncate(time.Second)

	token, err := svc.GenerateRefreshTokenWithLineage(service.RefreshTokenInput{
		UserID:        uid,
		FamilyRootJTI: rootJTI,
		ChainDepth:    42,
		FamilyRootIAT: rootIAT,
	})
	require.NoError(t, err)

	claims, err := svc.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, rootJTI, claims.FamilyRootJTI)
	assert.Equal(t, 42, claims.ChainDepth)
	assert.Equal(t, rootIAT.Unix(), claims.FamilyRootIAT.Unix())
	assert.NotEqual(t, rootJTI, claims.JTI,
		"new token's JTI must be distinct from family root")
}

func TestJWT_GenerateRefreshTokenWithLineage_EmptyInputReseeds(t *testing.T) {
	// When the caller passes empty lineage, the adapter must reseed
	// FamilyRootJTI to the new JTI and FamilyRootIAT to now. This is
	// the path used by login / register / invitation acceptance.
	svc := NewJWTService("secret-b8-reseed", time.Hour, 24*time.Hour)
	uid := uuid.New()

	token, err := svc.GenerateRefreshTokenWithLineage(service.RefreshTokenInput{
		UserID: uid,
	})
	require.NoError(t, err)

	claims, err := svc.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.Equal(t, claims.JTI, claims.FamilyRootJTI)
	assert.Equal(t, 0, claims.ChainDepth)
	assert.WithinDuration(t, time.Now(), claims.FamilyRootIAT, 5*time.Second)
}

func TestJWT_AccessToken_HasNoLineageClaims(t *testing.T) {
	// Sanity: lineage claims are refresh-only. Access tokens must
	// not carry frj/cd/fiat (omitempty keeps them out of the payload).
	svc := NewJWTService("secret-b8-access", time.Hour, 24*time.Hour)
	uid := uuid.New()

	token, err := svc.GenerateAccessToken(service.AccessTokenInput{
		UserID: uid,
		Role:   "provider",
	})
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Empty(t, claims.FamilyRootJTI)
	assert.Equal(t, 0, claims.ChainDepth)
	assert.True(t, claims.FamilyRootIAT.IsZero())
}
