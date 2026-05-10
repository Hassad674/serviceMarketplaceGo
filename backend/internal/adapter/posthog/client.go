// Package posthog wires the marketplace's server-side analytics adapter
// against the official posthog-go SDK. The package only depends on the
// AnalyticsService port — it knows nothing about handlers, app
// services, or domain entities. Wiring happens in cmd/api.
//
// Two design choices worth flagging:
//
//  1. Fail-open everywhere. Analytics is observability, not business
//     logic — a failed capture must NEVER bubble up to the caller and
//     break a real request. Every public method swallows errors with
//     a slog.Warn so an outage in PostHog's pipeline degrades to "no
//     dashboard data" rather than "no checkout".
//
//  2. The adapter uses posthog.NewWithConfig + Endpoint=eu.posthog.com
//     so events ship to the EU region. This keeps RGPD scope tight:
//     personal data never leaves the EU even when our backend is
//     deployed elsewhere.
package posthog

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	posthogsdk "github.com/posthog/posthog-go"

	portservice "marketplace-backend/internal/port/service"
)

// AnalyticsService implements port/service.AnalyticsService against
// the PostHog cloud. The struct holds the SDK client + the endpoint
// for diagnostic logging; everything else is delegated to the SDK's
// async dispatcher.
type AnalyticsService struct {
	client   posthogsdk.Client
	endpoint string
}

// Config groups the small surface the adapter actually needs. The
// project key is the public token — same value PostHog ships in the
// browser SDK. Endpoint defaults to the EU host when empty so
// calling code can hand us a zero-value Config in tests.
type Config struct {
	ProjectKey string
	Endpoint   string
	Verbose    bool
}

// NewAnalyticsService constructs the adapter. Returns an error when
// the SDK refuses the config (e.g. blank project key) — the wiring
// layer catches this and falls back to the noop adapter so a
// misconfiguration does not crash the boot.
func NewAnalyticsService(cfg Config) (*AnalyticsService, error) {
	if cfg.ProjectKey == "" {
		return nil, errors.New("posthog: project key is required")
	}
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://eu.posthog.com"
	}
	client, err := posthogsdk.NewWithConfig(cfg.ProjectKey, posthogsdk.Config{
		Endpoint: endpoint,
		Verbose:  cfg.Verbose,
	})
	if err != nil {
		return nil, fmt.Errorf("posthog: init client: %w", err)
	}
	return &AnalyticsService{client: client, endpoint: endpoint}, nil
}

// Capture queues a single event. The SDK batches and ships
// asynchronously — Capture returns immediately. A failed enqueue is
// logged at WARN level and otherwise swallowed so the caller's
// request flow is never disturbed by analytics.
func (s *AnalyticsService) Capture(_ context.Context, evt portservice.AnalyticsEvent) {
	if evt.DistinctID == "" || evt.EventName == "" {
		slog.Warn("posthog: capture rejected — missing distinct_id or event_name",
			"event", evt.EventName, "distinct_id", evt.DistinctID)
		return
	}
	props := posthogsdk.NewProperties()
	for k, v := range evt.Properties {
		props.Set(k, v)
	}
	msg := posthogsdk.Capture{
		Uuid:       evt.MessageID,
		DistinctId: evt.DistinctID,
		Event:      evt.EventName,
		Properties: props,
	}
	if evt.GroupKey != "" {
		msg.Groups = posthogsdk.NewGroups().Set("organization", evt.GroupKey)
	}
	if err := s.client.Enqueue(msg); err != nil {
		slog.Warn("posthog: enqueue capture failed",
			"event", evt.EventName, "error", err)
	}
}

// Identify attaches profile attributes to the distinct id. Idempotent
// — calling it twice with the same payload is a no-op on the PostHog
// side. Use it on login / register / profile edit to keep the user
// dimension fresh in dashboards.
func (s *AnalyticsService) Identify(_ context.Context, distinctID string, properties map[string]any) {
	if distinctID == "" {
		return
	}
	props := posthogsdk.NewProperties()
	for k, v := range properties {
		props.Set(k, v)
	}
	if err := s.client.Enqueue(posthogsdk.Identify{
		DistinctId: distinctID,
		Properties: props,
	}); err != nil {
		slog.Warn("posthog: enqueue identify failed",
			"distinct_id", distinctID, "error", err)
	}
}

// GroupIdentify attaches attributes to a group (e.g. an organization).
// Filters in PostHog dashboards then surface "events from organizations
// on the Premium plan" without us shipping the plan property on every
// individual event.
func (s *AnalyticsService) GroupIdentify(_ context.Context, groupType, groupKey string, properties map[string]any) {
	if groupType == "" || groupKey == "" {
		return
	}
	props := posthogsdk.NewProperties()
	for k, v := range properties {
		props.Set(k, v)
	}
	if err := s.client.Enqueue(posthogsdk.GroupIdentify{
		Type:       groupType,
		Key:        groupKey,
		Properties: props,
	}); err != nil {
		slog.Warn("posthog: enqueue group identify failed",
			"group_type", groupType, "group_key", groupKey, "error", err)
	}
}

// Close flushes the SDK's in-memory batch. Called once during the
// graceful-shutdown window so events captured in the last 5 seconds
// of life still reach PostHog.
func (s *AnalyticsService) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}
