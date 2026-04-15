// Command reindex is the bulk Typesense indexer.
//
// Usage:
//
//	go run ./cmd/reindex [--persona=freelance|agency|referrer|all] [--batch=100] [--dry-run]
//
// Iterates over every organization of the selected persona(s),
// builds a full SearchDocument using the same indexer + embeddings
// path as the live outbox worker, and upserts the results into
// Typesense in batches of 100 (default).
//
// The command is idempotent: running it twice back to back is a
// no-op because BulkUpsert uses the `action=upsert` flag, and the
// SearchDocument ID is the stable organization ID.
//
// Target: <60s for 50,000 organisations on local hardware.
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"marketplace-backend/internal/adapter/postgres"
	"marketplace-backend/internal/config"
	"marketplace-backend/internal/search"
)

const (
	defaultBatchSize   = 100
	progressEveryN     = 500
	bulkTimeout        = 10 * time.Minute
)

func main() {
	var (
		personaFlag = flag.String("persona", "all", "which persona to reindex: freelance, agency, referrer, all")
		batchSize   = flag.Int("batch", defaultBatchSize, "number of documents per Typesense bulk upsert")
		dryRun      = flag.Bool("dry-run", false, "build documents but do not upsert them (useful for benchmarking the indexer)")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg := config.Load()

	if err := run(cfg, *personaFlag, *batchSize, *dryRun); err != nil {
		slog.Error("reindex failed", "error", err)
		os.Exit(1)
	}
}

func run(cfg *config.Config, personaFlag string, batchSize int, dryRun bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), bulkTimeout)
	defer cancel()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	tsClient, err := search.NewClient(cfg.TypesenseHost, cfg.TypesenseAPIKey)
	if err != nil {
		return fmt.Errorf("typesense client: %w", err)
	}
	if !dryRun {
		if err := search.EnsureSchema(ctx, search.EnsureSchemaDeps{Client: tsClient, Logger: slog.Default()}); err != nil {
			return fmt.Errorf("ensure schema: %w", err)
		}
	}

	embedder := buildEmbedder(cfg)
	searchRepo := postgres.NewSearchDocumentRepository(db)
	indexer, err := search.NewIndexer(searchRepo, embedder)
	if err != nil {
		return fmt.Errorf("build indexer: %w", err)
	}

	personas, err := parsePersonaFlag(personaFlag)
	if err != nil {
		return err
	}

	start := time.Now()
	total := 0
	for _, p := range personas {
		count, err := reindexPersona(ctx, db, tsClient, indexer, p, batchSize, dryRun)
		if err != nil {
			return fmt.Errorf("reindex %s: %w", p, err)
		}
		total += count
	}
	elapsed := time.Since(start)
	slog.Info("reindex complete",
		"total_documents", total,
		"elapsed", elapsed,
		"docs_per_sec", float64(total)/elapsed.Seconds(),
		"dry_run", dryRun)
	return nil
}

// buildEmbedder picks the live OpenAI client when a key is set and
// falls back to the deterministic mock otherwise. The CLI is usable
// either way — the mock will produce a valid document shape but
// semantic queries won't match anything meaningful until phase 3
// re-runs with the live embedder.
func buildEmbedder(cfg *config.Config) search.EmbeddingsClient {
	if cfg.OpenAIAPIKey == "" {
		slog.Warn("reindex: OPENAI_API_KEY not set, using mock embeddings")
		return search.NewMockEmbeddings()
	}
	client, err := search.NewOpenAIEmbeddings(cfg.OpenAIAPIKey, cfg.OpenAIEmbeddingsModel)
	if err != nil {
		slog.Warn("reindex: openai client unavailable, falling back to mock", "error", err)
		return search.NewMockEmbeddings()
	}
	return client
}

// parsePersonaFlag translates the --persona CLI argument into the
// list of personas to reindex. "all" expands to the three known
// personas so a single invocation can rebuild the whole index.
func parsePersonaFlag(raw string) ([]search.Persona, error) {
	switch raw {
	case "all", "":
		return []search.Persona{search.PersonaFreelance, search.PersonaReferrer, search.PersonaAgency}, nil
	case "freelance":
		return []search.Persona{search.PersonaFreelance}, nil
	case "referrer":
		return []search.Persona{search.PersonaReferrer}, nil
	case "agency":
		return []search.Persona{search.PersonaAgency}, nil
	}
	return nil, fmt.Errorf("unknown persona %q", raw)
}

// reindexPersona walks every organization that has a profile row
// for the given persona and pushes a fresh document into Typesense.
// We fetch the IDs first (cheap: one SELECT), then iterate and
// build one document per org — the indexer's internal fan-out
// keeps per-document latency under 200ms so the whole run is
// bounded at a few minutes even for 50k orgs.
func reindexPersona(ctx context.Context, db *sql.DB, client *search.Client, indexer *search.Indexer, persona search.Persona, batchSize int, dryRun bool) (int, error) {
	ids, err := listOrgIDs(ctx, db, persona)
	if err != nil {
		return 0, fmt.Errorf("list org ids: %w", err)
	}
	slog.Info("reindex: starting persona", "persona", persona, "count", len(ids))

	batch := make([]*search.SearchDocument, 0, batchSize)
	built := 0
	for i, id := range ids {
		doc, err := indexer.BuildDocument(ctx, id, persona)
		if err != nil {
			slog.Warn("reindex: skip org on build error",
				"persona", persona, "org_id", id, "error", err)
			continue
		}
		batch = append(batch, doc)
		built++
		if len(batch) >= batchSize {
			if err := flushBatch(ctx, client, batch, dryRun); err != nil {
				return built, err
			}
			batch = batch[:0]
		}
		if (i+1)%progressEveryN == 0 {
			slog.Info("reindex: progress",
				"persona", persona, "processed", i+1, "total", len(ids))
		}
	}
	if len(batch) > 0 {
		if err := flushBatch(ctx, client, batch, dryRun); err != nil {
			return built, err
		}
	}
	slog.Info("reindex: persona complete", "persona", persona, "built", built)
	return built, nil
}

// listOrgIDs returns every organization ID that has a profile row
// for the given persona. The query is persona-dispatched so
// unrelated tables stay untouched — one SELECT, no JOIN storm.
func listOrgIDs(ctx context.Context, db *sql.DB, persona search.Persona) ([]uuid.UUID, error) {
	query, err := listQueryFor(persona)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0, 1024)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// listQueryFor returns the SELECT that enumerates every
// organization with a persona-specific profile row. Kept as a
// lookup so adding a persona is a one-line change.
func listQueryFor(persona search.Persona) (string, error) {
	switch persona {
	case search.PersonaFreelance:
		return `SELECT organization_id FROM freelance_profiles ORDER BY organization_id`, nil
	case search.PersonaReferrer:
		return `SELECT organization_id FROM referrer_profiles ORDER BY organization_id`, nil
	case search.PersonaAgency:
		return `SELECT organization_id FROM profiles ORDER BY organization_id`, nil
	}
	return "", fmt.Errorf("unknown persona %q", persona)
}

// flushBatch pushes the accumulated documents into Typesense
// (or logs the intent when --dry-run is on).
func flushBatch(ctx context.Context, client *search.Client, batch []*search.SearchDocument, dryRun bool) error {
	if dryRun {
		slog.Info("reindex: dry-run batch", "count", len(batch))
		return nil
	}
	if err := client.BulkUpsert(ctx, search.AliasName, batch); err != nil {
		return fmt.Errorf("bulk upsert: %w", err)
	}
	return nil
}
