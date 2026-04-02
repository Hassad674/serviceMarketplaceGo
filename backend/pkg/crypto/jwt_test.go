package crypto

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testSecret = "test-secret-key-for-unit-tests-32chars!"

func newTestJWTService() *JWTService {
	return NewJWTService(testSecret, 15*time.Minute, 7*24*time.Hour)
}

func TestJWTService_GenerateAccessToken_ReturnsNonEmpty(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	token, err := svc.GenerateAccessToken(userID, "agency", false)

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

	token1, err := svc.GenerateAccessToken(uuid.New(), "agency", false)
	require.NoError(t, err)

	token2, err := svc.GenerateAccessToken(uuid.New(), "provider", false)
	require.NoError(t, err)

	assert.NotEqual(t, token1, token2)
}

func TestJWTService_ValidateAccessToken_ValidToken(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()
	role := "enterprise"

	token, err := svc.GenerateAccessToken(userID, role, false)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(token)

	require.NoError(t, err)
	require.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, role, claims.Role)
	assert.False(t, claims.ExpiresAt.IsZero())
	assert.True(t, claims.ExpiresAt.After(time.Now()))
}

func TestJWTService_ValidateAccessToken_ExpiredToken(t *testing.T) {
	// Create a service with 0 expiry to generate already-expired tokens
	svc := NewJWTService(testSecret, -1*time.Second, 7*24*time.Hour)
	userID := uuid.New()

	token, err := svc.GenerateAccessToken(userID, "agency", false)
	require.NoError(t, err)

	claims, err := svc.ValidateAccessToken(token)

	assert.Error(t, err)
	assert.Nil(t, claims)
	assert.Contains(t, err.Error(), "token")
}

func TestJWTService_ValidateAccessToken_WrongSecret(t *testing.T) {
	svc1 := NewJWTService("secret-one-for-signing-tokens!!", 15*time.Minute, 7*24*time.Hour)
	svc2 := NewJWTService("secret-two-different-from-one!!", 15*time.Minute, 7*24*time.Hour)

	token, err := svc1.GenerateAccessToken(uuid.New(), "agency", false)
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

	accessToken, err := svc.GenerateAccessToken(userID, "agency", false)
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

			token, err := svc.GenerateAccessToken(userID, tt.role, false)
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
	token, err := svc.GenerateAccessToken(userID, "agency", false)
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
