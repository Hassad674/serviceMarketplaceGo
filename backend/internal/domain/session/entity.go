// Package session models the server-side audit trail of authentication
// sessions (B.4). Every login (password / invitation / token bridge /
// admin impersonation) and every refresh produces one Session row that
// captures the JTI of the issued refresh token, the parent JTI it was
// rotated from, an anonymized fingerprint of the caller, and the
// lifecycle timestamps.
//
// The package is pure domain — zero dependencies beyond the Go
// standard library and uuid — so the auth service, the Postgres
// adapter, and the future "Sécurité" handlers can all consume the
// same value type without dragging in framework concerns.
package session

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// LoginMethod enumerates how a session came into existence. Mirrors
// the CHECK constraint on user_sessions.login_method (migration 147)
// so an invalid string never reaches the adapter.
type LoginMethod string

const (
	// LoginMethodPassword — interactive email + password flow.
	LoginMethodPassword LoginMethod = "password"
	// LoginMethodInvitation — accepted-invitation auto-login.
	LoginMethodInvitation LoginMethod = "invitation"
	// LoginMethodTokenBridge — short-lived bridge token used by the
	// web app to hand off cookies to the mobile / native client.
	LoginMethodTokenBridge LoginMethod = "token_bridge"
	// LoginMethodRefresh — session row created on a refresh rotation,
	// chained to its parent via ParentJTI.
	LoginMethodRefresh LoginMethod = "refresh"
	// LoginMethodAdminImpersonation — staff-initiated session, used
	// only by support tooling. Always carries an audit trail.
	LoginMethodAdminImpersonation LoginMethod = "admin_impersonation"
)

// IsValid reports whether m is one of the allowlisted methods. The
// SQL CHECK constraint is the load-bearing guard; this method exists
// so the app layer can fail-fast before the round-trip.
func (m LoginMethod) IsValid() bool {
	switch m {
	case LoginMethodPassword,
		LoginMethodInvitation,
		LoginMethodTokenBridge,
		LoginMethodRefresh,
		LoginMethodAdminImpersonation:
		return true
	default:
		return false
	}
}

// String returns the on-the-wire representation of the method. Kept
// as a method (not a direct type alias) so a future migration to a
// typed-enum SQL column does not ripple through the call sites.
func (m LoginMethod) String() string {
	return string(m)
}

// Sentinel domain errors. The adapter and the app service use
// errors.Is() to discriminate.
var (
	ErrInvalidLoginMethod = errors.New("session: invalid login method")
	ErrJTIRequired        = errors.New("session: jti required")
	ErrUserIDRequired     = errors.New("session: user_id required")
	ErrUserAgentRequired  = errors.New("session: user_agent_hash required")
	ErrIPRequired         = errors.New("session: ip_anonymized required")
	ErrExpiresAtPast      = errors.New("session: expires_at must be in the future")
	ErrNotFound           = errors.New("session: not found")
)

// Session is the value object persisted in user_sessions.
type Session struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	JTI           string
	ParentJTI     string // empty when this is the first session of a chain
	UserAgentHash string
	IPAnonymized  string // CIDR string form ("203.0.113.0/24") or bare IP — both fit Postgres INET.
	LoginMethod   LoginMethod
	CreatedAt     time.Time
	LastUsedAt    time.Time
	ExpiresAt     time.Time
	RevokedAt     *time.Time // nil when still active

	// SEC-SESSIONS / migration 150 — display-grade enrichment of the
	// row, written at session creation so the Sécurité page can
	// render a Malt-style row ("Ordinateur de bureau (Chrome) — Paris
	// — 11/05/2026 10:48:46") without parsing the UA at read time.
	//
	// All four fields are optional ('' is the documented "unknown"
	// value, matching the SQL DEFAULT ''). The forensic columns above
	// (UserAgentHash, IPAnonymized) stay the source of truth for
	// security workflows — these new columns are display-only.
	DeviceLabel string // "Ordinateur de bureau (Chrome)" / "iPhone (Safari)" / "Appareil inconnu"
	Browser     string // "Chrome" / "Safari" / "Firefox" / "Edge" / "Opera" — '' when unknown
	OS          string // "Windows" / "macOS" / "Linux" / "iOS" / "Android" — '' when unknown
	City        string // free-form, returned by the GeoIP adapter — '' when unknown
	CountryCode string // ISO 3166-1 alpha-2, uppercase — '' when unknown
}

// Active reports whether the session is currently usable: not
// revoked, not yet expired (relative to now). Centralising the
// invariant here keeps every consumer (Sécurité page, retention
// sweep, audit reports) reading from the same definition.
func (s Session) Active(now time.Time) bool {
	if s.RevokedAt != nil {
		return false
	}
	return s.ExpiresAt.After(now)
}

// NewInput groups the construction params. Using a struct keeps the
// constructor under the project's 4-parameter rule and lets future
// fields be added without breaking call sites.
type NewInput struct {
	UserID        uuid.UUID
	JTI           string
	ParentJTI     string
	UserAgentHash string
	IPAnonymized  string
	LoginMethod   LoginMethod
	ExpiresAt     time.Time

	// Display-grade enrichment (SEC-SESSIONS / migration 150). All
	// optional — the empty string is the explicit "unknown" value.
	DeviceLabel string
	Browser     string
	OS          string
	City        string
	CountryCode string
}

// New constructs and validates a Session value. The ID, CreatedAt,
// and LastUsedAt fields are filled by the constructor — callers
// that need deterministic timestamps for tests should override them
// after construction.
func New(in NewInput) (*Session, error) {
	if in.UserID == uuid.Nil {
		return nil, ErrUserIDRequired
	}
	if in.JTI == "" {
		return nil, ErrJTIRequired
	}
	if strings.TrimSpace(in.UserAgentHash) == "" {
		return nil, ErrUserAgentRequired
	}
	if strings.TrimSpace(in.IPAnonymized) == "" {
		return nil, ErrIPRequired
	}
	if !in.LoginMethod.IsValid() {
		return nil, ErrInvalidLoginMethod
	}
	now := time.Now().UTC()
	if !in.ExpiresAt.After(now) {
		return nil, ErrExpiresAtPast
	}
	return &Session{
		ID:            uuid.New(),
		UserID:        in.UserID,
		JTI:           in.JTI,
		ParentJTI:     in.ParentJTI,
		UserAgentHash: in.UserAgentHash,
		IPAnonymized:  in.IPAnonymized,
		LoginMethod:   in.LoginMethod,
		CreatedAt:     now,
		LastUsedAt:    now,
		ExpiresAt:     in.ExpiresAt,
		DeviceLabel:   in.DeviceLabel,
		Browser:       in.Browser,
		OS:            in.OS,
		City:          in.City,
		CountryCode:   strings.ToUpper(strings.TrimSpace(in.CountryCode)),
	}, nil
}
