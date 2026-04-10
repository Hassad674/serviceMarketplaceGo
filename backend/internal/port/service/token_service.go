package service

import (
	"time"

	"github.com/google/uuid"
)

// AccessTokenInput groups the claims required to mint a new access token.
// Using a struct instead of positional params keeps the call site readable
// as we add optional claims (org context, session version, etc.) without
// violating the project's 4-parameter rule.
type AccessTokenInput struct {
	UserID  uuid.UUID
	Role    string
	IsAdmin bool

	// Organization context — set only for users who belong to an organization
	// (agencies, enterprises, or invited operators). Providers have both
	// OrganizationID = nil and OrgRole = "".
	OrganizationID *uuid.UUID
	OrgRole        string
}

// TokenClaims is the decoded payload of a validated token.
// New optional fields can be added without breaking callers that only
// care about UserID/Role/IsAdmin.
type TokenClaims struct {
	UserID    uuid.UUID
	Role      string
	IsAdmin   bool
	ExpiresAt time.Time

	// Organization context — nil / empty for solo users (Providers).
	OrganizationID *uuid.UUID
	OrgRole        string
}

type TokenService interface {
	GenerateAccessToken(input AccessTokenInput) (string, error)
	GenerateRefreshToken(userID uuid.UUID) (string, error)
	ValidateAccessToken(token string) (*TokenClaims, error)
	ValidateRefreshToken(token string) (*TokenClaims, error)
}
