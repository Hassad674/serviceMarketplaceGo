package noop

import (
	"context"

	portservice "marketplace-backend/internal/port/service"
)

// AnalyticsService is the always-safe fallback used when no PostHog
// project key is configured. Every method is a silent no-op so the
// rest of the codebase never branches on "is analytics enabled" —
// callers just dial Capture() and trust the wiring did the right
// thing at boot time.
type AnalyticsService struct{}

// NewAnalyticsService returns a no-op analytics adapter. Wired by
// bootstrap when c.PostHogConfigured() returns false. Logs a single
// WARN at boot so the absence is visible in production telemetry.
func NewAnalyticsService() *AnalyticsService {
	return &AnalyticsService{}
}

// Capture drops the event on the floor. Safe to call from any goroutine.
func (s *AnalyticsService) Capture(_ context.Context, _ portservice.AnalyticsEvent) {
}

// Identify drops the call.
func (s *AnalyticsService) Identify(_ context.Context, _ string, _ map[string]any) {
}

// GroupIdentify drops the call.
func (s *AnalyticsService) GroupIdentify(_ context.Context, _, _ string, _ map[string]any) {
}

// Close returns nil — nothing to flush.
func (s *AnalyticsService) Close() error { return nil }
