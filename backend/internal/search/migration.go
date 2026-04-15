package search

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
)

// migration.go drives the "first boot" schema setup for the
// marketplace Typesense index. Called once from cmd/api/main.go
// before the HTTP server starts listening.
//
// Semantics:
//
//  1. Check whether the `marketplace_actors` alias exists.
//  2. If it does NOT exist:
//     a. Create `marketplace_actors_v1` with the canonical schema.
//     b. Create the alias pointing to `_v1`.
//     That is a brand-new, empty collection — the bulk reindex
//     CLI or the outbox worker will populate it in a second step.
//  3. If it DOES exist: fetch the target collection and compare
//     its field count to the expected one. If it matches, no-op.
//     If it diverges, LOG a warning — we deliberately do NOT
//     auto-migrate, because dropping or renaming fields requires
//     a deliberate alias swap + bulk reindex orchestrated by an
//     operator (see phase 4 for the full `make reindex-swap` flow).
//
// The function is idempotent: calling it twice on a healthy cluster
// is a no-op. This is important because `main.go` can call it on
// every restart.

// EnsureSchemaDeps groups the small dependency set EnsureSchema
// needs. Using a struct keeps the call site readable and lets us
// add future knobs (logger, metrics) without breaking callers.
type EnsureSchemaDeps struct {
	Client *Client
	Logger *slog.Logger
}

// EnsureSchema makes the Typesense cluster ready to accept documents
// for the marketplace index. Returns a structured error on any
// persistent failure so the caller can either fail fast (strict
// mode) or degrade to SQL search (feature-flag mode).
func EnsureSchema(ctx context.Context, deps EnsureSchemaDeps) error {
	if deps.Client == nil {
		return fmt.Errorf("search ensure schema: client is required")
	}
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}

	target, err := deps.Client.GetAlias(ctx, AliasName)
	switch {
	case err == nil:
		// Alias exists — check drift on the current target.
		return inspectExistingAlias(ctx, deps.Client, logger, target)
	case errors.Is(err, ErrNotFound):
		// First boot on this cluster — create v1 from scratch.
		return bootstrapFreshCollection(ctx, deps.Client, logger)
	default:
		return fmt.Errorf("search ensure schema: get alias %q: %w", AliasName, err)
	}
}

// bootstrapFreshCollection is the "first time on this cluster" code
// path: create `_v1` and point the alias at it. Everything else
// (populating documents) happens later via the reindex CLI.
func bootstrapFreshCollection(ctx context.Context, client *Client, logger *slog.Logger) error {
	logger.Info("search: bootstrap fresh collection",
		"collection", CollectionName, "alias", AliasName)

	schema := CollectionSchemaDefinition()
	if err := client.CreateCollection(ctx, schema); err != nil {
		return fmt.Errorf("search bootstrap: create collection %q: %w",
			CollectionName, err)
	}
	if err := client.UpsertAlias(ctx, AliasName, CollectionName); err != nil {
		return fmt.Errorf("search bootstrap: upsert alias %q → %q: %w",
			AliasName, CollectionName, err)
	}

	logger.Info("search: bootstrap complete, collection is empty",
		"next_step", "run `make reindex-bulk` to populate the index")
	return nil
}

// inspectExistingAlias compares the alias's current target schema
// against the expected definition and logs a warning if they
// diverge. Auto-migration is deliberately NOT implemented here —
// a divergence is usually the sign of an in-progress v1 → v2
// migration that an operator is driving manually.
func inspectExistingAlias(ctx context.Context, client *Client, logger *slog.Logger, target string) error {
	logger.Info("search: alias already exists", "alias", AliasName, "target", target)

	live, err := client.GetCollection(ctx, target)
	if err != nil {
		return fmt.Errorf("search inspect: get collection %q: %w", target, err)
	}
	expected := CollectionSchemaDefinition()
	if len(live.Fields) != len(expected.Fields) {
		logger.Warn("search: schema drift detected — manual migration required",
			"target", target,
			"live_field_count", len(live.Fields),
			"expected_field_count", len(expected.Fields),
			"hint", "build a new `_vN` collection, bulk reindex, then swap the alias")
	}
	return nil
}
