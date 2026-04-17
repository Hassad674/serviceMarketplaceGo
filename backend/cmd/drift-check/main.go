// Command drift-check compares the per-persona document counts in
// Postgres against Typesense and logs a WARN when drift exceeds
// 0.5%.
//
// Exit codes:
//
//	0  no drift above threshold
//	2  drift above threshold (useful for CI / cron monitoring)
//	1  operational error (DB unreachable, Typesense unreachable, …)
//
// Usage:
//
//	go run ./cmd/drift-check [--threshold=0.005]
//
// NOT wired into Go code as a cron job. Schedule externally via
// systemd timer (`OnCalendar=hourly`) or Kubernetes CronJob. Example
// systemd unit:
//
//	[Unit]
//	Description=Hourly Typesense <-> Postgres drift check
//	After=network.target
//
//	[Service]
//	ExecStart=/opt/marketplace/bin/drift-check
//	EnvironmentFile=/etc/marketplace/backend.env
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	_ "github.com/lib/pq"

	"marketplace-backend/internal/config"
	"marketplace-backend/internal/search"
)

const driftTimeout = 2 * time.Minute

func main() {
	threshold := flag.Float64("threshold", search.DriftThreshold, "fraction of permissible drift (e.g. 0.005 = 0.5%)")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)
	cfg := config.Load()

	code, err := run(cfg, *threshold)
	if err != nil {
		slog.Error("drift check failed", "error", err)
		os.Exit(1)
	}
	os.Exit(code)
}

func run(cfg *config.Config, threshold float64) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), driftTimeout)
	defer cancel()

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return 1, fmt.Errorf("open postgres: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return 1, fmt.Errorf("ping postgres: %w", err)
	}

	postgresCounts, err := countPostgresPerPersona(ctx, db)
	if err != nil {
		return 1, fmt.Errorf("count postgres: %w", err)
	}
	slog.Info("postgres counts", "counts", postgresCounts)

	tsClient, err := search.NewClient(cfg.TypesenseHost, cfg.TypesenseAPIKey)
	if err != nil {
		return 1, fmt.Errorf("build typesense client: %w", err)
	}
	typesenseCounts, err := tsClient.CountDocumentsByPersona(ctx, search.AliasName)
	if err != nil {
		return 1, fmt.Errorf("count typesense: %w", err)
	}
	slog.Info("typesense counts", "counts", typesenseCounts)

	report := search.DetectDrift(postgresCounts, typesenseCounts, search.DetectDriftOpts{Threshold: threshold})
	if report.IsCritical {
		slog.Warn("drift detected above threshold",
			"threshold", threshold,
			"max_ratio", report.MaxRatio,
			"ratios", report.Ratios,
			"postgres", report.Postgres,
			"typesense", report.Typesense)
		return 2, nil
	}
	slog.Info("drift check passed",
		"threshold", threshold,
		"max_ratio", report.MaxRatio,
		"ratios", report.Ratios)
	return 0, nil
}

// countPostgresPerPersona counts rows per persona table. Every row
// in a persona table is treated as `is_published: true` by the
// indexer (phase 1 rationale: personas are opt-in so no hidden
// state). Freelance + agency + referrer each live in their own
// table — we aggregate with a single UNION ALL to keep the DB
// round-trip count at 1.
func countPostgresPerPersona(ctx context.Context, db *sql.DB) ([]search.PersonaCount, error) {
	const query = `
SELECT persona, count(*)::bigint FROM (
    SELECT 'freelance'::text AS persona
      FROM freelance_profiles
    UNION ALL
    SELECT 'agency'::text AS persona
      FROM profiles
    UNION ALL
    SELECT 'referrer'::text AS persona
      FROM referrer_profiles
) t GROUP BY persona`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[search.Persona]int64{
		search.PersonaFreelance: 0,
		search.PersonaAgency:    0,
		search.PersonaReferrer:  0,
	}
	for rows.Next() {
		var p string
		var count int64
		if err := rows.Scan(&p, &count); err != nil {
			return nil, err
		}
		out[search.Persona(p)] = count
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	counts := make([]search.PersonaCount, 0, 3)
	for _, p := range []search.Persona{search.PersonaFreelance, search.PersonaAgency, search.PersonaReferrer} {
		counts = append(counts, search.PersonaCount{Persona: p, Count: out[p]})
	}
	return counts, nil
}
