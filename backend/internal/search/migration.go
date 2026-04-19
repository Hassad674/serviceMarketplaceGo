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
		if inspectErr := inspectExistingAlias(ctx, deps.Client, logger, target); inspectErr != nil {
			return inspectErr
		}
	case errors.Is(err, ErrNotFound):
		// First boot on this cluster — create v1 from scratch.
		if bootErr := bootstrapFreshCollection(ctx, deps.Client, logger); bootErr != nil {
			return bootErr
		}
	default:
		return fmt.Errorf("search ensure schema: get alias %q: %w", AliasName, err)
	}

	// Synonyms are upserted on every boot. The seed list is small
	// (~30 entries) and the operation is idempotent on the
	// Typesense side, so the cost is negligible. We log a warning
	// instead of failing hard so a degraded synonyms endpoint
	// cannot block the API from starting.
	if synErr := SeedSynonyms(ctx, deps.Client, logger); synErr != nil {
		logger.Warn("search: synonyms seed failed, continuing without synonyms",
			"error", synErr)
	}
	return nil
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
// against the expected definition. When the drift is purely
// additive (every live field exists in the expected schema with
// the same type, and the expected schema has N more fields) we
// auto-apply the delta via PATCH /collections/:name. Non-additive
// drift (removals, renames, type changes) still requires the
// manual `_vN` alias-swap flow because PATCH cannot express those
// changes.
//
// Rationale: every ranking phase ships a handful of new numeric
// signals. Without auto-patching, each phase would require a
// manual Typesense migration before deploy. Additive changes are
// safe (legacy docs return default zero values for missing fields
// on query) so automating them removes an operator chokepoint.
func inspectExistingAlias(ctx context.Context, client *Client, logger *slog.Logger, target string) error {
	logger.Info("search: alias already exists", "alias", AliasName, "target", target)

	live, err := client.GetCollection(ctx, target)
	if err != nil {
		return fmt.Errorf("search inspect: get collection %q: %w", target, err)
	}
	expected := CollectionSchemaDefinition()
	if len(live.Fields) == len(expected.Fields) {
		return nil
	}

	missing, mismatch := diffSchemaFields(live.Fields, expected.Fields)
	if len(mismatch) > 0 {
		logger.Warn("search: non-additive schema drift detected — manual migration required",
			"target", target,
			"live_field_count", len(live.Fields),
			"expected_field_count", len(expected.Fields),
			"mismatch_fields", mismatchNames(mismatch),
			"hint", "build a new `_vN` collection, bulk reindex, then swap the alias")
		return nil
	}
	if len(missing) == 0 {
		// Live has extra fields we didn't declare — operator has
		// ongoing work we should leave alone.
		logger.Warn("search: live collection has unexpected extra fields — leaving as-is",
			"target", target,
			"live_field_count", len(live.Fields),
			"expected_field_count", len(expected.Fields))
		return nil
	}
	logger.Info("search: applying additive schema drift",
		"target", target,
		"missing_fields", missingNames(missing))
	if err := client.AddFields(ctx, target, missing); err != nil {
		logger.Warn("search: additive schema patch failed — falling back to warning",
			"target", target, "error", err,
			"hint", "run the manual `_vN` alias-swap flow before deploying code that depends on the new fields")
	}
	return nil
}

// diffSchemaFields compares two SchemaField slices by name. Returns:
//   - missing: fields present in `expected` but absent from `live`
//   - mismatch: fields present in both but with a differing type
//
// Fields present in `live` but missing from `expected` are ignored —
// operator drift (extra debugging fields) should not trip automation.
func diffSchemaFields(live, expected []SchemaField) (missing, mismatch []SchemaField) {
	liveByName := make(map[string]SchemaField, len(live))
	for _, f := range live {
		liveByName[f.Name] = f
	}
	for _, exp := range expected {
		got, ok := liveByName[exp.Name]
		if !ok {
			missing = append(missing, exp)
			continue
		}
		if got.Type != exp.Type {
			mismatch = append(mismatch, exp)
		}
	}
	return missing, mismatch
}

// missingNames projects a slice of SchemaField to its names — used
// only for log attribute formatting.
func missingNames(fields []SchemaField) []string {
	out := make([]string, len(fields))
	for i, f := range fields {
		out[i] = f.Name
	}
	return out
}

// mismatchNames is the twin of missingNames for non-additive drift.
func mismatchNames(fields []SchemaField) []string {
	return missingNames(fields)
}
