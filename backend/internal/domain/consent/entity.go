// Package consent records every consent decision (accept_all,
// refuse_all, custom) made through the cookie banner. The entry is
// the server-side proof of consent required by RGPD art. 7-1.
package consent

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Action identifies what choice the visitor made on the cookie banner.
type Action string

const (
	ActionAcceptAll Action = "accept_all"
	ActionRefuseAll Action = "refuse_all"
	ActionCustom    Action = "custom"
)

// IsValid returns true when the action matches one of the three
// allowed enum values. The DB CHECK constraint mirrors this list.
func (a Action) IsValid() bool {
	switch a {
	case ActionAcceptAll, ActionRefuseAll, ActionCustom:
		return true
	default:
		return false
	}
}

// Allowed category identifiers. Kept open-ended via the underlying
// TEXT[] column so adding a new vendor (e.g. session_replay) does not
// require a schema migration.
const (
	CategoryAnalytics  = "analytics"
	CategoryMarketing  = "marketing"
	CategoryFunctional = "functional"
)

// Entry is one consent_log row. UserID is nil for anonymous visitors;
// SessionID falls back to "" in that case.
//
// IPAnonymized is the truncated IP (IPv4 /16, IPv6 /32) — the raw IP
// MUST NEVER reach this struct. UserAgentHash is a hex-encoded SHA-256
// of the User-Agent header.
type Entry struct {
	ID            uuid.UUID
	UserID        *uuid.UUID
	SessionID     string
	Categories    []string
	Action        Action
	IPAnonymized  string
	UserAgentHash string
	CreatedAt     time.Time
}

// Validation errors. Sentinels — never wrapped inside the domain layer
// per backend/CLAUDE.md error-handling rules.
var (
	ErrInvalidAction       = errors.New("consent: invalid action")
	ErrCategoriesRequired  = errors.New("consent: categories must be non-empty")
	ErrIPAnonymizedRequired = errors.New("consent: ip_anonymized must be set")
	ErrUserAgentHashRequired = errors.New("consent: user_agent_hash must be set")
)

// NewInput is the constructor input. Mirrors the schema 1:1 so the
// service layer can hand the struct directly to the repository.
type NewInput struct {
	UserID        *uuid.UUID
	SessionID     string
	Categories    []string
	Action        Action
	IPAnonymized  string
	UserAgentHash string
}

// New validates the input and returns an Entry stamped with a fresh
// UUID + the current wall-clock time. The repository must NOT
// regenerate either field — the domain owns identity + timestamping.
func New(in NewInput) (*Entry, error) {
	if !in.Action.IsValid() {
		return nil, ErrInvalidAction
	}
	categories := normalizeCategories(in.Categories)
	if len(categories) == 0 {
		return nil, ErrCategoriesRequired
	}
	if strings.TrimSpace(in.IPAnonymized) == "" {
		return nil, ErrIPAnonymizedRequired
	}
	if strings.TrimSpace(in.UserAgentHash) == "" {
		return nil, ErrUserAgentHashRequired
	}
	return &Entry{
		ID:            uuid.New(),
		UserID:        in.UserID,
		SessionID:     strings.TrimSpace(in.SessionID),
		Categories:    categories,
		Action:        in.Action,
		IPAnonymized:  strings.TrimSpace(in.IPAnonymized),
		UserAgentHash: strings.TrimSpace(in.UserAgentHash),
		CreatedAt:     time.Now().UTC(),
	}, nil
}

// normalizeCategories trims, deduplicates (preserving first-seen
// order), and drops empty strings. Returns the cleaned slice.
func normalizeCategories(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, value := range raw {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
