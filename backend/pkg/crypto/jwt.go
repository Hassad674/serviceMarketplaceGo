package crypto

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"marketplace-backend/internal/port/service"
)

type JWTService struct {
	secret        string
	accessExpiry  time.Duration
	refreshExpiry time.Duration
}

func NewJWTService(secret string, accessExpiry, refreshExpiry time.Duration) *JWTService {
	return &JWTService{
		secret:        secret,
		accessExpiry:  accessExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// customClaims holds everything we pack into an access token.
//
// Organization context (OrgID, OrgRole) is optional and omitted from the
// JSON when the user is a solo Provider. The omitempty tags keep the
// token payload small for users without an org.
type customClaims struct {
	UserID  string `json:"user_id"`
	Role    string `json:"role,omitempty"`
	IsAdmin bool   `json:"is_admin,omitempty"`
	Type    string `json:"type"`

	// Organization context for members of an organization. When absent,
	// the user is a solo Provider (or a not-yet-activated account).
	OrgID   string `json:"org_id,omitempty"`
	OrgRole string `json:"org_role,omitempty"`

	jwt.RegisteredClaims
}

func (s *JWTService) GenerateAccessToken(input service.AccessTokenInput) (string, error) {
	claims := customClaims{
		UserID:  input.UserID.String(),
		Role:    input.Role,
		IsAdmin: input.IsAdmin,
		Type:    "access",
		OrgRole: input.OrgRole,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}
	if input.OrganizationID != nil {
		claims.OrgID = input.OrganizationID.String()
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *JWTService) GenerateRefreshToken(userID uuid.UUID) (string, error) {
	claims := customClaims{
		UserID: userID.String(),
		Type:   "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.refreshExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

func (s *JWTService) ValidateAccessToken(tokenString string) (*service.TokenClaims, error) {
	return s.validateToken(tokenString, "access")
}

func (s *JWTService) ValidateRefreshToken(tokenString string) (*service.TokenClaims, error) {
	return s.validateToken(tokenString, "refresh")
}

func (s *JWTService) validateToken(tokenString string, expectedType string) (*service.TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &customClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*customClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	if claims.Type != expectedType {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", expectedType, claims.Type)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user id in token: %w", err)
	}

	result := &service.TokenClaims{
		UserID:    userID,
		Role:      claims.Role,
		IsAdmin:   claims.IsAdmin,
		ExpiresAt: claims.ExpiresAt.Time,
		OrgRole:   claims.OrgRole,
	}
	if claims.OrgID != "" {
		orgID, err := uuid.Parse(claims.OrgID)
		if err != nil {
			return nil, fmt.Errorf("invalid org id in token: %w", err)
		}
		result.OrganizationID = &orgID
	}
	return result, nil
}
