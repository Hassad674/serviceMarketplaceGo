package main

import (
	"testing"

	"marketplace-backend/internal/config"
)

// TestWireHelpers_NilSafe locks in the empty-config behaviour of every
// wire* helper that accepts only a *config.Config.
//
// Every nil-safe path (Stripe absent, FCM absent, Typesense absent,
// LiveKit absent, AWS Rekognition absent, Anthropic absent) must keep
// the application bootable. These helpers are the single seam where
// "feature disabled" decisions are made, so a regression that makes
// them panic on missing env vars would brick the dev backend.
func TestWireHelpers_NilSafe(t *testing.T) {
	cfg := &config.Config{
		Env:                   "development",
		AllowedOrigins:        []string{"http://localhost:3000"},
		FrontendURL:           "http://localhost:3000",
		StorageEndpoint:       "localhost:9000",
		StorageAccessKey:      "minioadmin",
		StorageSecretKey:      "minioadmin",
		StorageBucket:         "test",
		JWTSecret:             "test-secret-32-bytes-of-padding-12",
		OpenAIEmbeddingsModel: "text-embedding-3-small",
	}

	t.Run("wireStripe returns empty struct when not configured", func(t *testing.T) {
		got := wireStripe(cfg)
		if got.Charges != nil || got.Reversals != nil || got.KYCReader != nil {
			t.Errorf("expected all stripe services nil when stripe is not configured, got %+v", got)
		}
	})

	t.Run("buildPushService returns nil when FCM_CREDENTIALS_PATH is empty", func(t *testing.T) {
		if got := buildPushService(cfg); got != nil {
			t.Errorf("expected nil push service when FCM is not configured, got %T", got)
		}
	})

	t.Run("buildContentModeration returns noop when Rekognition is not configured", func(t *testing.T) {
		if got := buildContentModeration(cfg); got == nil {
			t.Error("expected non-nil noop moderation service")
		}
	})

	t.Run("buildTransitStorage returns nil when video moderation is not configured", func(t *testing.T) {
		if got := buildTransitStorage(cfg); got != nil {
			t.Errorf("expected nil transit storage when video moderation is not configured, got %T", got)
		}
	})

	t.Run("buildTextModeration returns noop when no provider is configured", func(t *testing.T) {
		// Default provider is "openai" but with empty key the constructor still works.
		if got := buildTextModeration(cfg); got == nil {
			t.Error("expected non-nil text moderation service (noop fallback)")
		}
	})

	t.Run("wireSearchPublisher returns nil when Typesense is not configured", func(t *testing.T) {
		if got := wireSearchPublisher(cfg, nil); got != nil {
			t.Errorf("expected nil publisher when Typesense is not configured, got %T", got)
		}
	})
}

// TestWsOriginPatterns locks in the expected behaviour of the host
// extraction helper used to build the websocket origin allow-list.
// The router uses this list to reject cross-origin upgrade requests
// from foreign sites.
func TestWsOriginPatterns(t *testing.T) {
	cases := []struct {
		name    string
		origins []string
		want    []string
	}{
		{
			name:    "strips https scheme",
			origins: []string{"https://example.com"},
			want:    []string{"example.com", "localhost:*"},
		},
		{
			name:    "strips http scheme",
			origins: []string{"http://dev.example.com"},
			want:    []string{"dev.example.com", "localhost:*"},
		},
		{
			name:    "drops empty hosts",
			origins: []string{""},
			want:    []string{"localhost:*"},
		},
		{
			name:    "always appends localhost wildcard",
			origins: []string{"https://a.com", "https://b.com"},
			want:    []string{"a.com", "b.com", "localhost:*"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := wsOriginPatterns(tc.origins)
			if len(got) != len(tc.want) {
				t.Fatalf("len(got)=%d, len(want)=%d (got=%v, want=%v)", len(got), len(tc.want), got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("got[%d]=%q, want=%q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestPaymentProcessor_NilWhenStripeDisabled asserts the narrow port
// short-circuit used by the proposal feature: when Stripe is not
// configured, paymentProcessor returns nil so the proposal service
// degrades gracefully (milestone fund flows respond 503 instead of
// panicking against a nil Stripe client).
func TestPaymentProcessor_NilWhenStripeDisabled(t *testing.T) {
	cfg := &config.Config{Env: "development"}
	// Stripe disabled — empty StripeSecretKey makes StripeConfigured() false.
	if got := paymentProcessor(nil, cfg); got != nil {
		t.Errorf("expected nil payment processor when Stripe is not configured, got %T", got)
	}
}
