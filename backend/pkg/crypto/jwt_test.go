package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/port/service"
)

// jwtClaims is a generic claim map used by signWithClaims. Tests need
// to forge specific malformed payloads so they cannot rely on the
// service's GenerateAccessToken which always produces well-formed UUIDs.
type jwtClaims map[string]any

// signWithClaims signs the given claims with the shared test secret so
// validateToken accepts the signature and continues to the claim
// validation path we want to exercise.
func signWithClaims(t *testing.T, c jwtClaims) string {
	t.Helper()
	mc := jwt.MapClaims(c)
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, mc)
	signed, err := tok.SignedString([]byte(testSecret))
	require.NoError(t, err)
	return signed
}

func base64URLEncode(s string) string {
	return base64.URLEncoding.EncodeToString([]byte(s))
}

const testSecret = "test-secret-key-for-unit-tests-32chars!"

func newTestJWTService() *JWTService {
	return NewJWTService(testSecret, 15*time.Minute, 7*24*time.Hour)
}

// accessInput builds a minimal AccessTokenInput for tests. Callers that
// need organization context construct the input inline.
func accessInput(userID uuid.UUID, role string) service.AccessTokenInput {
	return service.AccessTokenInput{UserID: userID, Role: role, IsAdmin: false}
}

func TestJWTService_GenerateAccessToken_ReturnsNonEmpty(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	token, err := svc.GenerateAccessToken(accessInput(userID, "agency"))

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTService_GenerateRefreshToken_ReturnsNonEmpty(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	token, err := svc.GenerateRefreshToken(userID)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestJWTService_GenerateAccessToken_DifferentUsersProduceDifferentTokens(t *testing.T) {
	svc := newTestJWTService()

	token1, err := svc.GenerateAccessToken(accessInput(uuid.New(), "agency"))
	require.NoError(t, err)

	token2, err := svc.GenerateAccessToken(accessInput(uuid.New(), "provider"))
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2)
}

func TestJWTService_ValidateAccessToken_ValidToken(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()
	role := "enterprise"

	token, err := svc.GenerateAccessToken(accessInput(userID, role))
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(token)

	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, role, claims.Role)
	assert.False(t, claims.ExpiresAt.IsZero())
	assert.True(t, claims.ExpiresAt.After(time.Now()))
}

func TestJWTService_ValidateAccessToken_HasJTI(t *testing.T) {
	// SEC-06: every JWT MUST embed a non-empty JTI claim so the
	// refresh-blacklist can use it as the rotation key. Two consecutive
	// tokens for the same user must carry distinct JTIs.
	svc := newTestJWTService()
	userID := uuid.New()

	tokenA, err := svc.GenerateAccessToken(accessInput(userID, "agency"))
	require.NoError(t, err)
	tokenB, err := svc.GenerateAccessToken(accessInput(userID, "agency"))
	require.NoError(t, err)

	claimsA, err := svc.ValidateAccessToken(tokenA)
	require.NoError(t, err)
	claimsB, err := svc.ValidateAccessToken(tokenB)
	require.NoError(t, err)

	assert.NotEmpty(t, claimsA.JTI, "access token must have a JTI claim")
	assert.NotEmpty(t, claimsB.JTI)
	assert.NotEqual(t, claimsA.JTI, claimsB.JTI,
		"distinct tokens must produce distinct JTIs")
}

func TestJWTService_ValidateRefreshToken_HasJTI(t *testing.T) {
	// SEC-06: refresh tokens carry the JTI used by RefreshToken to
	// blacklist after rotation.
	svc := newTestJWTService()
	userID := uuid.New()

	token, err := svc.GenerateRefreshToken(userID)
	require.NoError(t, err)

	claims, err := svc.ValidateRefreshToken(token)
	require.NoError(t, err)
	assert.NotEmpty(t, claims.JTI, "refresh token must have a JTI claim")
}

func TestJWTService_ValidateAccessToken_ExpiredToken(t *testing.T) {
	// Create a service with 0 expiry to generate already-expired tokens
	svc := NewJWTService(testSecret, -1*time.Second, 7*24*time.Hour)
	userID := uuid.New()

	token, err := svc.GenerateAccessToken(accessInput(userID, "agency"))
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "token")
}

func TestJWTService_ValidateAccessToken_WrongSecret(t *testing.T) {
	svc1 := NewJWTService("secret-one-for-signing-tokens!!", 15*time.Minute, 7*24*time.Hour)
	svc2 := NewJWTService("secret-two-different-from-one!!", 15*time.Minute, 7*24*time.Hour)

	token, err := svc1.GenerateAccessToken(accessInput(uuid.New(), "agency"))
	require.NoError(t, err)

	claims, err := svc2.ValidateAccessToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTService_ValidateAccessToken_InvalidString(t *testing.T) {
	svc := newTestJWTService()

	claims, err := svc.ValidateAccessToken("not.a.valid.token")

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTService_ValidateAccessToken_EmptyString(t *testing.T) {
	svc := newTestJWTService()

	claims, err := svc.ValidateAccessToken("")

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTService_ValidateRefreshToken_WithAccessToken_ReturnsError(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	accessToken, err := svc.GenerateAccessToken(accessInput(userID, "agency"))
	require.NoError(t, err)

	// Trying to validate an access token as a refresh token should fail
	claims, err := svc.ValidateRefreshToken(accessToken)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "token type")
}

func TestJWTService_ValidateAccessToken_WithRefreshToken_ReturnsError(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	refreshToken, err := svc.GenerateRefreshToken(userID)
	require.NoError(t, err)

	// Trying to validate a refresh token as an access token should fail
	claims, err := svc.ValidateAccessToken(refreshToken)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "token type")
}

func TestJWTService_ValidateRefreshToken_ValidToken(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	token, err := svc.GenerateRefreshToken(userID)
	require.NoError(t, err)

	claims, err := svc.ValidateRefreshToken(token)

	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Empty(t, claims.Role, "refresh token should not contain role")
	assert.True(t, claims.ExpiresAt.After(time.Now()))
}

func TestJWTService_ValidateRefreshToken_ExpiredToken(t *testing.T) {
	svc := NewJWTService(testSecret, 15*time.Minute, -1*time.Second)
	userID := uuid.New()

	token, err := svc.GenerateRefreshToken(userID)
	require.NoError(t, err)

	claims, err := svc.ValidateRefreshToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTService_AccessToken_ContainsCorrectUserIDAndRole(t *testing.T) {
	svc := newTestJWTService()

	tests := []struct {
		name string
		role string
	}{
		{"agency role", "agency"},
		{"enterprise role", "enterprise"},
		{"provider role", "provider"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID := uuid.New()

			token, err := svc.GenerateAccessToken(accessInput(userID, tt.role))
			require.NoError(t, err)

			claims, err := svc.ValidateAccessToken(token)
			require.NoError(t, err)

			assert.Equal(t, userID, claims.UserID)
			assert.Equal(t, tt.role, claims.Role)
		})
	}
}

func TestJWTService_AccessToken_ExpiresWithinExpectedWindow(t *testing.T) {
	accessExpiry := 15 * time.Minute
	svc := NewJWTService(testSecret, accessExpiry, 7*24*time.Hour)
	userID := uuid.New()

	beforeGenerate := time.Now().Truncate(time.Second)
	token, err := svc.GenerateAccessToken(accessInput(userID, "agency"))
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(token)
	require.NoError(t, err)

	// JWT NumericDate has second precision, so truncate beforeGenerate to match
	expectedMin := beforeGenerate.Add(accessExpiry)
	expectedMax := beforeGenerate.Add(accessExpiry + 2*time.Second)

	assert.True(t, claims.ExpiresAt.After(expectedMin) || claims.ExpiresAt.Equal(expectedMin),
		"expiry %v should be at or after %v", claims.ExpiresAt, expectedMin)
	assert.True(t, claims.ExpiresAt.Before(expectedMax),
		"expiry %v should be before %v", claims.ExpiresAt, expectedMax)
}

func TestJWTService_RefreshToken_ExpiresWithinExpectedWindow(t *testing.T) {
	refreshExpiry := 7 * 24 * time.Hour
	svc := NewJWTService(testSecret, 15*time.Minute, refreshExpiry)
	userID := uuid.New()

	beforeGenerate := time.Now().Truncate(time.Second)
	token, err := svc.GenerateRefreshToken(userID)
	require.NoError(t, err)

	claims, err := svc.ValidateRefreshToken(token)
	require.NoError(t, err)

	// JWT NumericDate has second precision, so truncate beforeGenerate to match
	expectedMin := beforeGenerate.Add(refreshExpiry)
	expectedMax := beforeGenerate.Add(refreshExpiry + 2*time.Second)

	assert.True(t, claims.ExpiresAt.After(expectedMin) || claims.ExpiresAt.Equal(expectedMin),
		"expiry %v should be at or after %v", claims.ExpiresAt, expectedMin)
	assert.True(t, claims.ExpiresAt.Before(expectedMax),
		"expiry %v should be before %v", claims.ExpiresAt, expectedMax)
}

// TestJWTService_AccessToken_WithOrganizationContext verifies that
// organization fields survive the round-trip through the token.
func TestJWTService_AccessToken_WithOrganizationContext(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()
	orgID := uuid.New()

	token, err := svc.GenerateAccessToken(service.AccessTokenInput{
		UserID:         userID,
		Role:           "agency",
		IsAdmin:        false,
		OrganizationID: &orgID,
		OrgRole:        "owner",
	})
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims.OrganizationID)
	assert.Equal(t, orgID, *claims.OrganizationID)
	assert.Equal(t, "owner", claims.OrgRole)
}

// TestJWTService_AccessToken_WithoutOrganizationContext verifies that
// Provider-style tokens (no org context) decode with nil/empty org fields.
func TestJWTService_AccessToken_WithoutOrganizationContext(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	token, err := svc.GenerateAccessToken(accessInput(userID, "provider"))
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.Nil(t, claims.OrganizationID)
	assert.Empty(t, claims.OrgRole)
}

// SEC-06: validateToken must reject any non-HMAC signing algorithm so a
// malicious caller cannot use the alg=none / RS256 algorithm-confusion
// attack to forge tokens that the server accepts. We craft a token
// with the "none" alg and assert the validator rejects it.
func TestJWTService_ValidateAccessToken_RejectsNoneAlg(t *testing.T) {
	svc := newTestJWTService()

	// Craft a token signed with `none`. jwt-go refuses this by default
	// for ParseWithClaims so we build the JWT manually.
	header := `{"alg":"none","typ":"JWT"}`
	payload := `{"user_id":"00000000-0000-0000-0000-000000000001","type":"access","exp":99999999999}`
	encode := func(s string) string {
		return strings.TrimRight(base64URLEncode(s), "=")
	}
	tok := encode(header) + "." + encode(payload) + "."

	claims, err := svc.ValidateAccessToken(tok)
	require.Error(t, err)
	assert.Nil(t, claims)
}

// validateToken returns "invalid user id in token" when the user_id
// claim is malformed. Hand-craft a token signed with the test secret
// but with a non-UUID user_id.
func TestJWTService_ValidateAccessToken_InvalidUserID(t *testing.T) {
	tok := signWithClaims(t, jwtClaims{
		"user_id": "not-a-uuid",
		"type":    "access",
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	})

	svc := newTestJWTService()
	claims, err := svc.ValidateAccessToken(tok)
	require.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "user id")
}

// validateToken returns "invalid org id in token" when the org_id
// claim is present but malformed.
func TestJWTService_ValidateAccessToken_InvalidOrgID(t *testing.T) {
	tok := signWithClaims(t, jwtClaims{
		"user_id": uuid.NewString(),
		"type":    "access",
		"org_id":  "not-a-uuid",
		"exp":     time.Now().Add(time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	})

	svc := newTestJWTService()
	claims, err := svc.ValidateAccessToken(tok)
	require.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "org id")
}

// Round-trip with permissions + session_version verifies those fields
// are preserved end-to-end. Otherwise the auth middleware would lose
// per-org permission overrides on every refresh.
func TestJWTService_AccessToken_PermissionsAndSessionVersion(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	token, err := svc.GenerateAccessToken(service.AccessTokenInput{
		UserID:         userID,
		Role:           "agency",
		Permissions:    []string{"missions.read", "missions.write"},
		SessionVersion: 7,
	})
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(token)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"missions.read", "missions.write"}, claims.Permissions)
	assert.Equal(t, 7, claims.SessionVersion)
}
