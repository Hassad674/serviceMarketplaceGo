package service

import (
	"time"

	"github.com/google/uuid"
)

type TokenClaims struct {
	UserID    uuid.UUID
	Role      string
	ExpiresAt time.Time
}

type TokenService interface {
	GenerateAccessToken(userID uuid.UUID, role string) (string, error)
	GenerateRefreshToken(userID uuid.UUID) (string, error)
	ValidateAccessToken(token string) (*TokenClaims, error)
	ValidateRefreshToken(token string) (*TokenClaims, error)
}
