package security

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/audit"
	"marketplace-backend/internal/port/repository"
)

// authActions is the closed set of audit actions surfaced on the
// /me/security/activity endpoint. The list is intentionally narrow:
// the page exists to show a user "where my account was authenticated
// from" — feature-level audit (receipts, referrals, ...) belongs to
// admin tooling, not to the per-user activity tab.
var authActions = map[audit.Action]struct{}{
	audit.ActionLoginSuccess:          {},
	audit.ActionLogout:                {},
	audit.ActionTokenRefresh:          {},
	audit.ActionPasswordResetRequest:  {},
	audit.ActionPasswordResetComplete: {},
}

// IsAuthAction returns true when action is one of the auth-related
// audit actions surfaced by the security activity endpoint. Exposed
// so the handler-level filter test can pin the canonical list.
func IsAuthAction(action audit.Action) bool {
	_, ok := authActions[action]
	return ok
}

// Service is the use-case orchestrator for the security activity tab.
// It owns nothing but a read port over the audit_logs table; deleting
// the package leaves the repository untouched.
type Service struct {
	audits repository.AuditRepository
}

// NewService wires the service. A nil audits repository is rejected
// at construction time so callers do not have to guard against it on
// every call site.
func NewService(audits repository.AuditRepository) *Service {
	if audits == nil {
		return nil
	}
	return &Service{audits: audits}
}

// Event is the per-row projection returned to the handler. The
// handler maps Event into the JSON DTO; keeping the shape here lets
// the use-case live without a dependency on the http package.
type Event struct {
	ID               uuid.UUID
	Action           audit.Action
	IPAddress        string
	UserAgentRaw     string
	UserAgentSummary UserAgentSummary
	CountryHint      string
	CreatedAt        time.Time
}

// ListPage is the cursor-paginated result. Events is non-nil even
// when empty so the JSON encoding renders `[]` (not `null`).
type ListPage struct {
	Events     []Event
	NextCursor string
}

// ErrInvalidUser is returned when a zero-valued user id reaches the
// service. Handlers should guard against this at the boundary, but
// the service rejects it as a defense-in-depth check.
var ErrInvalidUser = errors.New("security activity: user id required")

// ListActivity returns the most recent authentication-related audit
// events attributable to the given user, paginated newest-first.
//
// The repository fetches by user_id; we then drop rows whose action
// is outside the auth allow-list. We over-fetch by a small factor
// (≤ limit*overFetchFactor, capped at 100) so a page of feature
// audit rows interleaved with auth rows still yields `limit` auth
// rows on the first call. The cursor returned is the cursor from
// the underlying repository — pagination stays stable even when the
// first page filtered out unrelated rows.
//
// The function is intentionally pure transformation over the
// repository; concurrency, RLS, and DB timeouts are the
// repository's responsibility.
func (s *Service) ListActivity(ctx context.Context, userID uuid.UUID, cursor string, limit int) (*ListPage, error) {
	if s == nil || s.audits == nil {
		return nil, errors.New("security service not configured")
	}
	if userID == uuid.Nil {
		return nil, ErrInvalidUser
	}
	limit = clampLimit(limit)

	// Over-fetch up to 3× to absorb non-auth rows the filter drops.
	// Capped at 100 so the SQL stays cheap; if 100 rows are not
	// enough to fill a single page of auth events the caller can
	// just request the next cursor.
	fetchSize := limit * 3
	if fetchSize > 100 {
		fetchSize = 100
	}

	entries, nextCursor, err := s.audits.ListByUser(ctx, userID, cursor, fetchSize)
	if err != nil {
		return nil, err
	}

	out := make([]Event, 0, len(entries))
	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if _, ok := authActions[entry.Action]; !ok {
			continue
		}
		out = append(out, toEvent(entry))
		if len(out) >= limit {
			break
		}
	}
	return &ListPage{Events: out, NextCursor: nextCursor}, nil
}

func clampLimit(limit int) int {
	switch {
	case limit <= 0:
		return 20
	case limit > 50:
		return 50
	default:
		return limit
	}
}

// toEvent projects a domain audit entry into the public Event shape,
// extracting the IP and user-agent from either the dedicated audit
// fields (IPAddress) or the metadata bag (user_agent / country) the
// auth service may populate. The metadata fall-back keeps the
// endpoint useful even on legacy rows.
func toEvent(entry *audit.Entry) Event {
	ev := Event{
		ID:        entry.ID,
		Action:    entry.Action,
		CreatedAt: entry.CreatedAt,
	}
	if entry.IPAddress != nil {
		ev.IPAddress = entry.IPAddress.String()
	}
	if entry.Metadata != nil {
		if v, ok := entry.Metadata["user_agent"].(string); ok {
			ev.UserAgentRaw = strings.TrimSpace(v)
		}
		if v, ok := entry.Metadata["country"].(string); ok {
			ev.CountryHint = strings.TrimSpace(v)
		}
		if ev.IPAddress == "" {
			if v, ok := entry.Metadata["ip"].(string); ok {
				ev.IPAddress = strings.TrimSpace(v)
			}
		}
	}
	ev.UserAgentSummary = ParseUserAgent(ev.UserAgentRaw)
	return ev
}
