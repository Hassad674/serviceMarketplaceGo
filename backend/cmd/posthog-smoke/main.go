// Command posthog-smoke ships one event to PostHog and exits. Used as
// the integration smoke test for the analytics adapter — when this
// command exits 0 with a "ok" line, we know:
//
//   1. The env vars are loaded correctly.
//   2. The adapter constructs a working SDK client.
//   3. Capture() ships the event AND Close() flushes synchronously
//      before the process exits. Without the Close() the SDK would
//      buffer and the binary would exit before the HTTP request is
//      flushed — a classic disappearing-event bug.
//
// Run it like:
//
//	POSTHOG_PROJECT_KEY=phc_xxx POSTHOG_HOST=https://eu.posthog.com \
//	  go run ./cmd/posthog-smoke
//
// The PostHog UI's activity feed should show a `smoke_test.backend`
// event for distinct id `smoke-backend` within ~2 seconds.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"marketplace-backend/internal/adapter/posthog"
	portservice "marketplace-backend/internal/port/service"
)

func main() {
	key := os.Getenv("POSTHOG_PROJECT_KEY")
	host := os.Getenv("POSTHOG_HOST")
	if host == "" {
		host = "https://eu.posthog.com"
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "POSTHOG_PROJECT_KEY is required")
		os.Exit(1)
	}

	verbose := os.Getenv("POSTHOG_DEBUG") == "true"

	svc, err := posthog.NewAnalyticsService(posthog.Config{
		ProjectKey: key,
		Endpoint:   host,
		Verbose:    verbose,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "init: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	now := time.Now().UTC().Format(time.RFC3339)

	// 1. Identify so the user shows up with attributes in PostHog.
	svc.Identify(ctx, "smoke-backend", map[string]any{
		"role":   "smoke",
		"source": "go-cmd",
	})

	// 2. Capture the actual event — this is what should land in the
	//    PostHog activity feed.
	svc.Capture(ctx, portservice.AnalyticsEvent{
		DistinctID: "smoke-backend",
		EventName:  "smoke_test.backend",
		Properties: map[string]any{
			"ts":      now,
			"source":  "smoke",
			"runtime": "go",
		},
	})

	// 3. Close flushes the SDK's batch synchronously. Without this
	//    the process exits before the HTTP POST lands.
	if err := svc.Close(); err != nil {
		slog.Error("close failed", "error", err)
		os.Exit(1)
	}

	fmt.Println("ok")
}
