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

	// Permissions is the resolved effective permission set (static
	// defaults + per-org overrides) captured at token issuance time.
	// Embedded in the JWT so mobile clients and the RequirePermission
	// middleware see the same customized set the session cookie carries
	// for web clients. Empty when the user has no org.
	Permissions []string

	// SessionVersion is copied from user.session_version at issuance
	// time. The auth middleware compares this against the current value
	// on every request — a mismatch means the user's effective
	// permissions have changed and any in-flight token must be rejected.
	// Defaults to 0 for fresh accounts.
	SessionVersion int
}

// TokenClaims is the decoded payload of a validated token.
// New optional fields can be added without breaking callers that only
// care about UserID/Role/IsAdmin.
type TokenClaims struct {
	UserID    uuid.UUID
	Role      string
	IsAdmin   bool
	ExpiresAt time.Time

	// JTI is the unique token id (UUID v4) embedded in every JWT. The
	// auth service uses it as the Redis blacklist key when a refresh
	// token is rotated or revoked (SEC-06). Always populated for
	// tokens minted by this codebase since Phase 1.
	JTI string

	// Organization context — nil / empty for solo users (Providers).
	OrganizationID *uuid.UUID
	OrgRole        string

	// Permissions is the list of effective permission keys embedded in
	// the token at issuance. Mirrors the session cookie semantics so
	// web and mobile clients behave identically against the
	// RequirePermission middleware.
	Permissions []string

	// SessionVersion from the decoded access token. Auth middleware
	// compares this against the current users.session_version and
	// rejects the request if they differ.
	SessionVersion int

	// B.8 — refresh-token family lineage. Empty / zero on legacy
	// tokens minted before B.8 ships; the auth service treats those
	// tokens as a fresh family on the next rotation (FamilyRootJTI is
	// reseeded to the current JTI).

	// FamilyRootJTI is the JTI of the first refresh token in this
	// rotation chain (the one minted at login). All descendants in the
	// chain carry the same value so the auth service can blacklist the
	// entire family in a single Redis lookup when reuse is detected.
	FamilyRootJTI string

	// ChainDepth is the number of rotations between this token and
	// the family root. The login-issued root has depth 0; the first
	// rotation produces depth 1, etc. The auth service rejects any
	// rotation that would push depth beyond the configured cap (see
	// auth.MaxRefreshChainDepth).
	ChainDepth int

	// FamilyRootIAT is the issued-at timestamp of the family root.
	// Carried forward unchanged through every rotation so the auth
	// service can enforce an absolute session lifetime independent of
	// per-token TTL: once `now - FamilyRootIAT >= MaxFamilyAge`, the
	// chain is rejected and the user must re-login.
	FamilyRootIAT time.Time
}

// RefreshTokenInput groups the parameters needed to mint a refresh
// token. Using a struct (rather than overloading GenerateRefreshToken
// with positional params) keeps the call site readable and lets the
// auth service copy lineage fields forward across rotations without
// growing a 7-arg signature.
//
// FamilyRootJTI / ChainDepth / FamilyRootIAT are zero on the very
// first token of a chain (login). The adapter seeds them from the
// new token's own JTI / 0 / time.Now() in that case so the issued
// token is always self-consistent (every refresh token is the root
// of its own family, even if the family contains only one member).
type RefreshTokenInput struct {
	UserID uuid.UUID

	// FamilyRootJTI is the JTI of the family root. Empty on first
	// issuance — the adapter substitutes the new token's own JTI.
	FamilyRootJTI string

	// ChainDepth is the depth this new token will carry. Callers pass
	// the parent token's depth + 1 on rotation, or 0 on first issuance.
	ChainDepth int

	// FamilyRootIAT is the issued-at of the family root. Zero on first
	// issuance — the adapter substitutes time.Now().
	FamilyRootIAT time.Time
}

type TokenService interface {
	GenerateAccessToken(input AccessTokenInput) (string, error)
	// GenerateRefreshToken mints a refresh token starting a fresh
	// family (depth 0, FamilyRootJTI = self). Kept for backward
	// compatibility with login / register / invitation acceptance
	// callers that have no parent token to copy lineage from.
	GenerateRefreshToken(userID uuid.UUID) (string, error)
	// GenerateRefreshTokenWithLineage mints a refresh token that
	// continues an existing family (used by rotation). The adapter
	// fills in the new JTI itself and copies FamilyRootJTI /
	// ChainDepth / FamilyRootIAT from the input.
	GenerateRefreshTokenWithLineage(input RefreshTokenInput) (string, error)
	ValidateAccessToken(token string) (*TokenClaims, error)
	ValidateRefreshToken(token string) (*TokenClaims, error)
}
