package service

import "context"

// AnalyticsEvent is the structured payload an app service hands to the
// analytics adapter. Distinct from a generic map so the field names
// stay grep-able and the adapter can refuse a malformed payload at
// the boundary instead of silently dropping it.
//
// Idempotency: when MessageID is set the adapter passes it through to
// the SDK so duplicate captures (e.g. a Stripe webhook retried by
// Stripe's reliability layer) collapse into a single PostHog event.
// Use the upstream provider's event ID (Stripe event ID, FCM message
// ID) so the dedupe key is stable across our own retry windows.
type AnalyticsEvent struct {
	// DistinctID is the user UUID this event belongs to. For pre-auth
	// events (registration completion before the JWT is issued) pass
	// the soon-to-be-created user_id once it is known.
	DistinctID string
	// EventName follows the dotted convention `domain.action_done`.
	// Examples: `auth.user_registered`, `proposal.payment_succeeded`.
	EventName string
	// Properties are flat key/value attributes captured alongside the
	// event. Strongly prefer scalar values (int, string, bool) — the
	// PostHog UI does not flatten nested objects in dashboards.
	Properties map[string]any
	// MessageID is an optional idempotency token. When set the SDK
	// uses it as the event UUID so PostHog dedupes server-side.
	MessageID string
	// GroupKey is the organization UUID the event should be attached
	// to (PostHog "group analytics"). Empty = ungrouped.
	GroupKey string
}

// AnalyticsService is the port the app layer calls when it wants to
// emit an event. Always fail-open: the implementation MUST NOT return
// errors that block the main request, only log them (analytics is
// observability, never a hard dependency).
type AnalyticsService interface {
	// Capture sends a single event. Safe to call from any goroutine.
	// The adapter buffers + flushes async — callers do not wait.
	Capture(ctx context.Context, evt AnalyticsEvent)
	// Identify attaches profile attributes to the distinct id. Idempotent.
	Identify(ctx context.Context, distinctID string, properties map[string]any)
	// GroupIdentify attaches attributes to a group (e.g. organization).
	GroupIdentify(ctx context.Context, groupType, groupKey string, properties map[string]any)
	// Close flushes any buffered events. Called once at shutdown.
	Close() error
}
