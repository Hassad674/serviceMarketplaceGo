// Command seed-search populates the database with a large, realistic
// synthetic dataset so the search engine can be exercised end-to-end
// against a representative workload.
//
// Unlike `cmd/seed` (which only creates the admin/roles bootstrap) this
// tool builds profiles that look close to production:
//
//   - ~500 profiles by default, split 300 freelance / 120 agency /
//     80 referrer. The distribution is configurable via --count.
//   - Display names drawn from curated French + English first/last
//     name pools — avoids the "Alice0, Alice1, …" synthetic feel.
//   - Role-appropriate titles from a hand-curated pool of 30+
//     variants (Senior React Developer, Agence webdesign Paris,
//     Apporteur d'affaires SaaS B2B, etc.).
//   - 2-3 sentence bios that reference the role + a real skill.
//   - Skill sets drawn from ~100 tech + non-tech pool (React, Go,
//     Copywriting FR, SEO, Figma, Stripe, Kubernetes, …).
//   - Realistic pricing ranges per persona and a weighted rating
//     distribution (40 % unrated, 35 % high rating, 20 % average,
//     5 % low).
//   - Cities + coordinates from 30 top EU/NA cities so geo filtering
//     returns non-empty results.
//
// The tool is idempotent: re-running wipes the `*@search.seed` rows
// and reinserts deterministically, so tests can assert on fixed
// organization IDs. Determinism comes from seeding math/rand with a
// CLI-provided value (default: 42).
//
// Usage:
//
//	make seed-search                               # default 500 profiles
//	make seed-search ARGS="--count=200 --seed=7"   # 200 profiles, seed 7
//	make seed-search ARGS="--no-reindex"           # skip typesense reindex
//
// After seeding, the tool triggers a full Typesense reindex so the
// engine reflects the new content. Pass --no-reindex when you only
// need the database rows (e.g. for local Postgres integration tests).
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/config"
)

const (
	defaultTotalCount = 500
	defaultSeedValue  = 42
	seedNamespace     = "11111111-2222-3333-4444-555555555555"
	runTimeout        = 10 * time.Minute
)

func main() {
	var (
		total     = flag.Int("count", defaultTotalCount, "total number of profiles to create")
		seedVal   = flag.Int64("seed", defaultSeedValue, "deterministic seed for profile generation")
		noReindex = flag.Bool("no-reindex", false, "skip the Typesense reindex at the end")
		freelance = flag.Int("freelance", 0, "override: number of freelance profiles (default 60% of count)")
		agency    = flag.Int("agency", 0, "override: number of agency profiles (default 24% of count)")
		referrer  = flag.Int("referrer", 0, "override: number of referrer profiles (default 16% of count)")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	counts := personaCounts{
		freelance: *freelance,
		agency:    *agency,
		referrer:  *referrer,
	}
	counts.resolveDefaults(*total)

	if err := run(counts, *seedVal, *noReindex); err != nil {
		slog.Error("seed-search failed", "error", err)
		os.Exit(1)
	}
}

// personaCounts captures the per-persona target count. The resolver
// fills the default distribution (60/24/16) when all overrides are
// zero. Users can pass any non-zero override to customise the split.
type personaCounts struct {
	freelance, agency, referrer int
}

func (c *personaCounts) resolveDefaults(total int) {
	if c.freelance == 0 && c.agency == 0 && c.referrer == 0 {
		c.freelance = total * 60 / 100
		c.agency = total * 24 / 100
		c.referrer = total - c.freelance - c.agency
	}
}

func (c personaCounts) total() int { return c.freelance + c.agency + c.referrer }

func run(counts personaCounts, seedVal int64, noReindex bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	cfg := config.Load()
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping postgres: %w", err)
	}

	r := rand.New(rand.NewSource(seedVal))

	slog.Info("seed-search starting",
		"freelance", counts.freelance, "agency", counts.agency,
		"referrer", counts.referrer, "total", counts.total(), "seed", seedVal)

	if err := wipePreviousSeed(ctx, db); err != nil {
		return fmt.Errorf("wipe previous seed: %w", err)
	}
	if err := seedSkillsCatalog(ctx, db); err != nil {
		return fmt.Errorf("seed skills_catalog: %w", err)
	}
	if err := seedAllPersonas(ctx, db, counts, r); err != nil {
		return fmt.Errorf("seed profiles: %w", err)
	}

	slog.Info("seed-search complete", "total", counts.total())

	if !noReindex {
		if err := runReindex(ctx); err != nil {
			slog.Warn("reindex failed — dataset is in Postgres but not in Typesense", "error", err)
			return err
		}
	}
	return nil
}

// wipePreviousSeed removes every row inserted by a prior run. Rows are
// identified by the `@search.seed` email suffix, which is never used
// outside this seeder. Delete order respects FK dependencies.
func wipePreviousSeed(ctx context.Context, db *sql.DB) error {
	const suffix = `%@search.seed`
	rows, err := db.QueryContext(ctx,
		`SELECT id FROM organizations WHERE owner_user_id IN (
			SELECT id FROM users WHERE email LIKE $1
		)`, suffix)
	if err != nil {
		return err
	}
	defer rows.Close()
	var orgIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		orgIDs = append(orgIDs, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	cleanups := []string{
		`DELETE FROM profile_skills WHERE organization_id = ANY($1)`,
		`DELETE FROM freelance_pricing WHERE profile_id IN (SELECT id FROM freelance_profiles WHERE organization_id = ANY($1))`,
		`DELETE FROM freelance_profiles WHERE organization_id = ANY($1)`,
		`DELETE FROM referrer_profiles WHERE organization_id = ANY($1)`,
		`DELETE FROM profiles WHERE organization_id = ANY($1)`,
	}
	for _, q := range cleanups {
		if _, err := db.ExecContext(ctx, q, pq.Array(orgIDs)); err != nil {
			return fmt.Errorf("cleanup %q: %w", q, err)
		}
	}

	if _, err := db.ExecContext(ctx,
		`UPDATE users SET organization_id = NULL WHERE email LIKE $1`, suffix); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`DELETE FROM organizations WHERE id = ANY($1)`, pq.Array(orgIDs)); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`DELETE FROM users WHERE email LIKE $1`, suffix); err != nil {
		return err
	}
	return nil
}

func seedSkillsCatalog(ctx context.Context, db *sql.DB) error {
	for _, skill := range skillPool {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO skills_catalog (skill_text, display_text) VALUES ($1, $1)
			 ON CONFLICT DO NOTHING`, skill); err != nil {
			return err
		}
	}
	return nil
}

func seedAllPersonas(ctx context.Context, db *sql.DB, counts personaCounts, r *rand.Rand) error {
	for i := 0; i < counts.freelance; i++ {
		if err := seedFreelance(ctx, db, i, r); err != nil {
			return fmt.Errorf("freelance #%d: %w", i, err)
		}
	}
	for i := 0; i < counts.agency; i++ {
		if err := seedAgency(ctx, db, i, r); err != nil {
			return fmt.Errorf("agency #%d: %w", i, err)
		}
	}
	for i := 0; i < counts.referrer; i++ {
		if err := seedReferrer(ctx, db, i, r); err != nil {
			return fmt.Errorf("referrer #%d: %w", i, err)
		}
	}
	return nil
}

func deterministicUUID(label string) uuid.UUID {
	return uuid.NewSHA1(uuid.MustParse(seedNamespace), []byte(label))
}

func runReindex(ctx context.Context) error {
	slog.Info("triggering Typesense reindex")
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/reindex")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	return cmd.Run()
}

// availabilityForRNG biases toward `now` (60 %), `soon` (25 %), `not` (15 %).
func availabilityForRNG(r *rand.Rand) string {
	v := r.Intn(100)
	switch {
	case v < 60:
		return "now"
	case v < 85:
		return "soon"
	default:
		return "not"
	}
}

// ratingBucket returns (avg, count) drawn from the weighted brief
// distribution (40 % unrated / 35 % high / 20 % mid / 5 % low).
func ratingBucket(r *rand.Rand) (float64, int) {
	v := r.Intn(100)
	switch {
	case v < 40:
		return 0, 0
	case v < 75:
		return 4.0 + r.Float64()*1.0, 5 + r.Intn(26)
	case v < 95:
		return 3.0 + r.Float64()*1.0, 2 + r.Intn(9)
	default:
		return 1.0 + r.Float64()*2.0, 1 + r.Intn(5)
	}
}

// workModesForRNG returns 1-3 work modes in a stable order.
func workModesForRNG(r *rand.Rand) []string {
	all := []string{"remote", "on_site", "hybrid"}
	n := 1 + r.Intn(3)
	r.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	return all[:n]
}

// languagesForRNG returns 1-3 language codes. Always includes fr or en.
func languagesForRNG(r *rand.Rand) []string {
	all := []string{"fr", "en", "es", "de", "it", "pt"}
	n := 1 + r.Intn(3)
	r.Shuffle(len(all), func(i, j int) { all[i], all[j] = all[j], all[i] })
	picked := append([]string{}, all[:n]...)
	for _, l := range picked {
		if l == "fr" || l == "en" {
			return picked
		}
	}
	picked[0] = "fr"
	return picked
}

// skillsForRNG returns 3-8 skills drawn without replacement from the pool.
func skillsForRNG(r *rand.Rand) []string {
	n := 3 + r.Intn(6)
	idxs := r.Perm(len(skillPool))[:n]
	out := make([]string, 0, n)
	for _, i := range idxs {
		out = append(out, skillPool[i])
	}
	return out
}

// expertiseForRNG returns 1-3 expertise keys.
func expertiseForRNG(r *rand.Rand) []string {
	n := 1 + r.Intn(3)
	idxs := r.Perm(len(expertisePool))[:n]
	out := make([]string, 0, n)
	for _, i := range idxs {
		out = append(out, expertisePool[i])
	}
	return out
}

// lastActiveAt draws an exponentially-weighted timestamp (median ~4 days ago,
// clamped to 365 days). Ensures the `last_active_at` signal has a realistic
// heavy tail.
func lastActiveAt(r *rand.Rand, now time.Time) time.Time {
	u := r.Float64()
	if u >= 0.999999 {
		u = 0.999999
	}
	daysBack := -math.Log(1-u) / 0.15
	if daysBack > 365 {
		daysBack = 365
	}
	return now.Add(-time.Duration(daysBack*24) * time.Hour)
}
