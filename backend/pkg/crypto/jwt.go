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

type customClaims struct {
	UserID  string `json:"user_id"`
	Role    string `json:"role,omitempty"`
	IsAdmin bool   `json:"is_admin,omitempty"`
	Type    string `json:"type"`
	jwt.RegisteredClaims
}

func (s *JWTService) GenerateAccessToken(userID uuid.UUID, role string, isAdmin bool) (string, error) {
	claims := customClaims{
		UserID:  userID.String(),
		Role:    role,
		IsAdmin: isAdmin,
		Type:    "access",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.accessExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        uuid.New().String(),
		},
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

	return &service.TokenClaims{
		UserID:    userID,
		Role:      claims.Role,
		IsAdmin:   claims.IsAdmin,
		ExpiresAt: claims.ExpiresAt.Time,
	}, nil
}
