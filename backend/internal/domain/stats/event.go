// Package stats holds the domain types for the per-org public stats
// feature: profile views, visibility time series, top search keywords,
// and the application-counts time series the enterprise dashboard
// renders.
//
// This package has zero persistence responsibilities and no external
// imports beyond the Go stdlib + uuid. The repository contracts live
// in port/repository, the adapter in adapter/postgres, and the use
// cases in app/stats.
package stats

import (
	"errors"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Persona identifies which split-profile surface emitted the view.
// The DB CHECK constraint mirrors this list.
type Persona string

const (
	PersonaFreelance Persona = "freelance"
	PersonaAgency    Persona = "agency"
	PersonaReferrer  Persona = "referrer"
)

// IsValid returns true when the persona matches one of the three
// allowed enum values.
func (p Persona) IsValid() bool {
	switch p {
	case PersonaFreelance, PersonaAgency, PersonaReferrer:
		return true
	default:
		return false
	}
}

// CameFrom identifies the navigation source that brought the visitor
// to the profile detail page.
type CameFrom string

const (
	CameFromSearch   CameFrom = "search"
	CameFromList     CameFrom = "list"
	CameFromDirect   CameFrom = "direct"
	CameFromReferral CameFrom = "referral"
	CameFromUnknown  CameFrom = "unknown"
)

// IsValid returns true when the source matches one of the five
// allowed enum values.
func (c CameFrom) IsValid() bool {
	switch c {
	case CameFromSearch, CameFromList, CameFromDirect, CameFromReferral, CameFromUnknown:
		return true
	default:
		return false
	}
}

// Sentinel domain errors. Never wrapped inside this package per the
// project's error-handling rules.
var (
	ErrInvalidPersona   = errors.New("stats: invalid persona")
	ErrInvalidCameFrom  = errors.New("stats: invalid came_from")
	ErrOrgIDRequired    = errors.New("stats: organization_id is required")
	ErrIPRequired       = errors.New("stats: viewer ip is required")
	ErrIPInvalid        = errors.New("stats: viewer ip is not a valid CIDR/IP")
	ErrUAHashRequired   = errors.New("stats: viewer user-agent hash is required")
	ErrSearchPosNonNeg  = errors.New("stats: search position must be >= 1")
	ErrPeriodInvalid    = errors.New("stats: period days must be one of 7/30/90/365")
)

// ViewEvent is one profile_view_events row. ViewerUserID is nil for
// anonymous visitors; SearchQuery + SearchPosition are populated only
// when CameFrom == CameFromSearch.
type ViewEvent struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	Persona            Persona
	ViewerUserID       *uuid.UUID
	ViewerIPAnonymized string
	ViewerUAHash       string
	CameFrom           CameFrom
	SearchQuery        *string
	SearchPosition     *int
	ReferrerURL        *string
	CreatedAt          time.Time
}

// NewViewEventInput is the constructor input. Mirrors the schema 1:1
// so the service can hand the struct to the repository directly.
type NewViewEventInput struct {
	OrganizationID     uuid.UUID
	Persona            Persona
	ViewerUserID       *uuid.UUID
	ViewerIPAnonymized string
	ViewerUAHash       string
	CameFrom           CameFrom
	SearchQuery        *string
	SearchPosition     *int
	ReferrerURL        *string
}

// NewViewEvent validates the input and returns a ViewEvent stamped
// with a fresh UUID + the current wall-clock time. The repository
// MUST NOT regenerate either field — the domain owns identity +
// timestamping.
func NewViewEvent(in NewViewEventInput) (*ViewEvent, error) {
	if in.OrganizationID == uuid.Nil {
		return nil, ErrOrgIDRequired
	}
	if !in.Persona.IsValid() {
		return nil, ErrInvalidPersona
	}
	if !in.CameFrom.IsValid() {
		return nil, ErrInvalidCameFrom
	}
	ip := strings.TrimSpace(in.ViewerIPAnonymized)
	if ip == "" {
		return nil, ErrIPRequired
	}
	if !isParseableIPOrCIDR(ip) {
		return nil, ErrIPInvalid
	}
	if strings.TrimSpace(in.ViewerUAHash) == "" {
		return nil, ErrUAHashRequired
	}
	if in.SearchPosition != nil && *in.SearchPosition < 1 {
		return nil, ErrSearchPosNonNeg
	}

	return &ViewEvent{
		ID:                 uuid.New(),
		OrganizationID:     in.OrganizationID,
		Persona:            in.Persona,
		ViewerUserID:       in.ViewerUserID,
		ViewerIPAnonymized: ip,
		ViewerUAHash:       strings.TrimSpace(in.ViewerUAHash),
		CameFrom:           in.CameFrom,
		SearchQuery:        trimmedPtr(in.SearchQuery),
		SearchPosition:     in.SearchPosition,
		ReferrerURL:        trimmedPtr(in.ReferrerURL),
		CreatedAt:          time.Now().UTC(),
	}, nil
}

// trimmedPtr returns a pointer to the trimmed string, or nil when the
// input is nil OR trims to empty. Used for optional fields that map
// to NULL in the DB.
func trimmedPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// isParseableIPOrCIDR returns true when the input parses as a bare
// IP address OR as a CIDR. Postgres INET accepts both shapes — we
// stay permissive here so the truncation helper can return either.
func isParseableIPOrCIDR(s string) bool {
	if net.ParseIP(s) != nil {
		return true
	}
	if _, _, err := net.ParseCIDR(s); err == nil {
		return true
	}
	return false
}

// AnonymizeIP truncates an IP to a privacy-respecting CIDR network so
// the persisted value is no longer reasonably attributable to an
// identified person (RGPD recital 26).
//
//   - IPv4: keep the /24 network (last octet zeroed) → "203.0.113.0/24"
//   - IPv6: keep the /64 network → "2001:db8::/64"
//
// Returns "" on empty input. Returns the original string when the
// parse fails — the caller's INET cast then rejects the row, surfacing
// the bad input as a 4xx instead of silently logging unmasked data.
func AnonymizeIP(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed := net.ParseIP(trimmed)
	if parsed == nil {
		return trimmed
	}
	if v4 := parsed.To4(); v4 != nil {
		mask := net.CIDRMask(24, 32)
		network := v4.Mask(mask)
		return (&net.IPNet{IP: network, Mask: mask}).String()
	}
	mask := net.CIDRMask(64, 128)
	network := parsed.Mask(mask)
	return (&net.IPNet{IP: network, Mask: mask}).String()
}
