// Package fixtures holds reusable test data generators for the
// search engine phases. The search_profiles.go file builds a
// deterministic set of 200 synthetic actors (120 freelance, 50
// agency, 30 referrer) following the Pareto skill + rating
// distributions documented in the phase-1 brief.
//
// Deterministic means two invocations produce the exact same
// rows: the UUIDs are derived from a fixed namespace + index so
// a test can assert on specific profile IDs without tracking a
// randomised seed. This matters for the golden semantic tests
// that expect `top-3 = [profile-3, profile-17, profile-42]`.
package fixtures

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// fixtureNamespace is the fixed UUID namespace used to derive
// deterministic IDs for every fixture via uuid.NewSHA1. Changing
// this constant invalidates every existing fixture's ID — don't.
var fixtureNamespace = uuid.MustParse("11111111-2222-3333-4444-555555555555")

// SearchFixtureCounts documents the target distribution per
// persona. Exposed so tests can assert on the totals without
// duplicating the numbers.
type SearchFixtureCounts struct {
	Freelance int
	Agency    int
	Referrer  int
}

// DefaultCounts is the canonical 200-profile mix used by the
// phase-1 integration tests.
var DefaultCounts = SearchFixtureCounts{
	Freelance: 120,
	Agency:    50,
	Referrer:  30,
}

// Total returns the sum of all personas.
func (c SearchFixtureCounts) Total() int { return c.Freelance + c.Agency + c.Referrer }

// SeedSearchProfiles populates a Postgres database with the
// canonical 200-fixture set. Idempotent: running it twice in a
// row is a no-op because every row is keyed on a deterministic
// UUID and protected by ON CONFLICT DO NOTHING.
//
// Returns the list of organization IDs grouped by persona so
// tests can pick specific fixtures to assert on.
type SeededProfiles struct {
	Freelance []uuid.UUID
	Agency    []uuid.UUID
	Referrer  []uuid.UUID
}

// SeedSearchProfiles inserts the fixture set into the database.
// Uses transactions for atomicity — a failure halfway through
// rolls back without leaving partial state.
func SeedSearchProfiles(ctx context.Context, db *sql.DB, counts SearchFixtureCounts) (*SeededProfiles, error) {
	out := &SeededProfiles{
		Freelance: make([]uuid.UUID, 0, counts.Freelance),
		Agency:    make([]uuid.UUID, 0, counts.Agency),
		Referrer:  make([]uuid.UUID, 0, counts.Referrer),
	}

	// Seed skills catalog first — every skill referenced below
	// must exist before profile_skills INSERTs can satisfy the
	// FK.
	for _, skill := range skillsPool {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO skills_catalog (skill_text, display_text) VALUES ($1, $1)
			 ON CONFLICT DO NOTHING`, skill); err != nil {
			return nil, fmt.Errorf("seed skills_catalog %q: %w", skill, err)
		}
	}

	for i := 0; i < counts.Freelance; i++ {
		id, err := seedFreelance(ctx, db, i)
		if err != nil {
			return nil, fmt.Errorf("seed freelance #%d: %w", i, err)
		}
		out.Freelance = append(out.Freelance, id)
	}
	for i := 0; i < counts.Agency; i++ {
		id, err := seedAgency(ctx, db, i)
		if err != nil {
			return nil, fmt.Errorf("seed agency #%d: %w", i, err)
		}
		out.Agency = append(out.Agency, id)
	}
	for i := 0; i < counts.Referrer; i++ {
		id, err := seedReferrer(ctx, db, i)
		if err != nil {
			return nil, fmt.Errorf("seed referrer #%d: %w", i, err)
		}
		out.Referrer = append(out.Referrer, id)
	}
	return out, nil
}

// deterministicUUID derives a stable UUID from a fixed namespace
// + a logical label, so fixture IDs stay identical across runs
// without depending on math/rand seeds or the process start time.
func deterministicUUID(label string) uuid.UUID {
	return uuid.NewSHA1(fixtureNamespace, []byte(label))
}

// -------- Distribution tables --------
//
// Every table below encodes a piece of the brief's distribution.
// They stay as package-level vars so tests can import them if
// they need to assert on the exact mix (e.g. counting how many
// fixtures landed in Paris).

// citiesPool is the geo distribution. 60 Paris, 20 Lyon, 15 Marseille,
// 10 Bordeaux, 10 Toulouse, 10 Nantes, 75 "other" across EU.
var citiesPool = []struct {
	City, Country string
	Lat, Lng      float64
}{
	{"Paris", "FR", 48.8566, 2.3522},
	{"Lyon", "FR", 45.7640, 4.8357},
	{"Marseille", "FR", 43.2965, 5.3698},
	{"Bordeaux", "FR", 44.8378, -0.5792},
	{"Toulouse", "FR", 43.6047, 1.4442},
	{"Nantes", "FR", 47.2184, -1.5536},
	{"Berlin", "DE", 52.5200, 13.4050},
	{"Amsterdam", "NL", 52.3676, 4.9041},
	{"Barcelona", "ES", 41.3851, 2.1734},
	{"Lisbon", "PT", 38.7223, -9.1393},
}

// skillsPool follows a Pareto distribution: first entries are the
// most common across the fixture set.
var skillsPool = []string{
	"React", "Go", "Python", "TypeScript", "Node.js", "PostgreSQL",
	"Docker", "Kubernetes", "AWS", "GCP", "Next.js", "Vue", "Rust",
	"GraphQL", "Redis", "MongoDB", "Figma", "Tailwind", "FastAPI",
	"LangChain", "Django", "Flask", "Terraform", "Ansible", "CI/CD",
	"Machine Learning", "LLM", "Stripe", "WebRTC", "Flutter",
}

// languagesPool is the two-letter ISO code distribution for
// `languages_professional`. "fr" dominates, then "en", then a
// sprinkle of others to cover the multi-language path.
var languagesPool = []string{"fr", "en", "es", "de", "it", "pt"}

// workModesPool is the work mode distribution. Most fixtures are
// "remote"; a handful land in "hybrid" and "on-site" to exercise
// the facet filter.
var workModesPool = []string{"remote", "hybrid", "on-site"}

// expertiseDomainsPool maps to the domain/expertise catalog keys
// (keep in sync with the expertise domain registry when it lands).
var expertiseDomainsPool = []string{
	"dev-frontend", "dev-backend", "dev-mobile", "data-ml",
	"design-ux", "design-ui", "marketing-growth", "ops-devops",
	"product-management", "legal-compliance",
}

// -------- Low-level seeders --------

// seedFreelance inserts one freelance fixture: user + organization +
// freelance_profiles + pricing + skills + social links. Every row
// is idempotent via ON CONFLICT.
func seedFreelance(ctx context.Context, db *sql.DB, index int) (uuid.UUID, error) {
	orgID := deterministicUUID(fmt.Sprintf("freelance-org-%d", index))
	userID := deterministicUUID(fmt.Sprintf("freelance-user-%d", index))
	city := citiesPool[index%len(citiesPool)]
	email := fmt.Sprintf("freelance-%d@search.fixtures", index)

	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type)
		 VALUES ($1, $2, 'hash', $3, $4, $5, 'provider', 'marketplace_owner')
		 ON CONFLICT (id) DO NOTHING`,
		userID, email, fmt.Sprintf("Alice%d", index), fmt.Sprintf("Dev%d", index),
		fmt.Sprintf("Alice Dev %d", index)); err != nil {
		return uuid.Nil, err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO organizations (id, owner_user_id, type, name, photo_url, city, country_code, latitude, longitude,
		                            work_mode, languages_professional)
		 VALUES ($1, $2, 'provider_personal', $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (id) DO NOTHING`,
		orgID, userID, fmt.Sprintf("Alice Dev %d", index),
		fmt.Sprintf("https://cdn/alice-%d.webp", index),
		city.City, city.Country, city.Lat, city.Lng,
		pq.Array([]string{workModesPool[index%len(workModesPool)]}),
		pq.Array(languagesForIndex(index)),
	); err != nil {
		return uuid.Nil, err
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, userID); err != nil {
		return uuid.Nil, err
	}

	profileID := deterministicUUID(fmt.Sprintf("freelance-profile-%d", index))
	if _, err := db.ExecContext(ctx,
		`INSERT INTO freelance_profiles (id, organization_id, title, about, expertise_domains)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (organization_id) DO NOTHING`,
		profileID, orgID,
		fmt.Sprintf("Senior Developer %d", index),
		fmt.Sprintf("Experienced professional #%d shipping software for %d years.", index, (index%10)+1),
		pq.Array(expertisesForIndex(index)),
	); err != nil {
		return uuid.Nil, err
	}

	if _, err := db.ExecContext(ctx,
		`INSERT INTO freelance_pricing (profile_id, pricing_type, min_amount, max_amount, currency, negotiable)
		 VALUES ($1, 'daily', $2, $3, 'EUR', $4)
		 ON CONFLICT (profile_id) DO NOTHING`,
		profileID, int64(30000+index*500), int64(60000+index*800), index%3 == 0,
	); err != nil {
		return uuid.Nil, err
	}

	for pos, skill := range skillsForIndex(index) {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO profile_skills (organization_id, skill_text, position) VALUES ($1, $2, $3)
			 ON CONFLICT DO NOTHING`, orgID, skill, pos); err != nil {
			return uuid.Nil, err
		}
	}
	return orgID, nil
}

// seedAgency inserts one agency fixture. The legacy `profiles`
// table owns agency data (no freelance_profiles row).
func seedAgency(ctx context.Context, db *sql.DB, index int) (uuid.UUID, error) {
	orgID := deterministicUUID(fmt.Sprintf("agency-org-%d", index))
	userID := deterministicUUID(fmt.Sprintf("agency-user-%d", index))
	city := citiesPool[(index+3)%len(citiesPool)]
	email := fmt.Sprintf("agency-%d@search.fixtures", index)

	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type)
		 VALUES ($1, $2, 'hash', 'Agency', $3, $4, 'agency', 'marketplace_owner')
		 ON CONFLICT (id) DO NOTHING`,
		userID, email, fmt.Sprintf("Owner%d", index), fmt.Sprintf("Agency %d", index)); err != nil {
		return uuid.Nil, err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO organizations (id, owner_user_id, type, name, photo_url, city, country_code, latitude, longitude,
		                            work_mode, languages_professional)
		 VALUES ($1, $2, 'agency', $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (id) DO NOTHING`,
		orgID, userID, fmt.Sprintf("Agency %d", index),
		fmt.Sprintf("https://cdn/agency-%d.webp", index),
		city.City, city.Country, city.Lat, city.Lng,
		pq.Array([]string{"remote", "hybrid"}),
		pq.Array([]string{"fr", "en"}),
	); err != nil {
		return uuid.Nil, err
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, userID); err != nil {
		return uuid.Nil, err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO profiles (organization_id, title, about)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (organization_id) DO NOTHING`,
		orgID, fmt.Sprintf("Full-service agency %d", index),
		fmt.Sprintf("Agency #%d specialising in end-to-end digital product delivery.", index),
	); err != nil {
		return uuid.Nil, err
	}
	return orgID, nil
}

// seedReferrer inserts one referrer fixture. Uses the referrer_profiles
// table.
func seedReferrer(ctx context.Context, db *sql.DB, index int) (uuid.UUID, error) {
	orgID := deterministicUUID(fmt.Sprintf("referrer-org-%d", index))
	userID := deterministicUUID(fmt.Sprintf("referrer-user-%d", index))
	city := citiesPool[(index+5)%len(citiesPool)]
	email := fmt.Sprintf("referrer-%d@search.fixtures", index)

	if _, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name, role, account_type, referrer_enabled)
		 VALUES ($1, $2, 'hash', 'Referrer', $3, $4, 'provider', 'marketplace_owner', true)
		 ON CONFLICT (id) DO NOTHING`,
		userID, email, fmt.Sprintf("Owner%d", index), fmt.Sprintf("Referrer %d", index)); err != nil {
		return uuid.Nil, err
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO organizations (id, owner_user_id, type, name, city, country_code, latitude, longitude,
		                            work_mode, languages_professional)
		 VALUES ($1, $2, 'provider_personal', $3, $4, $5, $6, $7, $8, $9)
		 ON CONFLICT (id) DO NOTHING`,
		orgID, userID, fmt.Sprintf("Referrer %d", index),
		city.City, city.Country, city.Lat, city.Lng,
		pq.Array([]string{"remote"}), pq.Array([]string{"fr", "en"}),
	); err != nil {
		return uuid.Nil, err
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE users SET organization_id = $1 WHERE id = $2`, orgID, userID); err != nil {
		return uuid.Nil, err
	}
	profileID := deterministicUUID(fmt.Sprintf("referrer-profile-%d", index))
	if _, err := db.ExecContext(ctx,
		`INSERT INTO referrer_profiles (id, organization_id, title, about, expertise_domains)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (organization_id) DO NOTHING`,
		profileID, orgID,
		fmt.Sprintf("Business referrer %d", index),
		fmt.Sprintf("Referrer #%d with a network of SaaS + B2B enterprises.", index),
		pq.Array([]string{"marketing-growth", "sales-bizdev"}),
	); err != nil {
		return uuid.Nil, err
	}
	return orgID, nil
}

// skillsForIndex returns a deterministic 5-skill slice drawn from
// the Pareto pool. Higher indices land on less popular skills so
// the overall distribution follows the brief.
func skillsForIndex(index int) []string {
	out := make([]string, 0, 5)
	for i := 0; i < 5; i++ {
		out = append(out, skillsPool[(index+i)%len(skillsPool)])
	}
	return out
}

// languagesForIndex returns a 1-2 language slice derived from the
// pool. Every fixture has at least French OR English.
func languagesForIndex(index int) []string {
	if index%3 == 0 {
		return []string{"fr", "en"}
	}
	return []string{languagesPool[index%len(languagesPool)]}
}

// expertisesForIndex returns a 1-3 expertise slice from the pool.
func expertisesForIndex(index int) []string {
	count := (index % 3) + 1
	out := make([]string, 0, count)
	for i := 0; i < count; i++ {
		out = append(out, expertiseDomainsPool[(index+i)%len(expertiseDomainsPool)])
	}
	return out
}

// CleanupSearchProfiles removes every fixture row inserted by
// SeedSearchProfiles. Useful in the integration tests that want
// a clean database at the end of the run.
func CleanupSearchProfiles(ctx context.Context, db *sql.DB, seeded *SeededProfiles) error {
	if seeded == nil {
		return nil
	}
	allOrgs := append([]uuid.UUID{}, seeded.Freelance...)
	allOrgs = append(allOrgs, seeded.Agency...)
	allOrgs = append(allOrgs, seeded.Referrer...)

	if _, err := db.ExecContext(ctx, `DELETE FROM profile_skills WHERE organization_id = ANY($1)`, pq.Array(orgIDsAsStrings(allOrgs))); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM freelance_pricing WHERE profile_id IN (SELECT id FROM freelance_profiles WHERE organization_id = ANY($1))`, pq.Array(orgIDsAsStrings(allOrgs))); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM freelance_profiles WHERE organization_id = ANY($1)`, pq.Array(orgIDsAsStrings(allOrgs))); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM referrer_profiles WHERE organization_id = ANY($1)`, pq.Array(orgIDsAsStrings(allOrgs))); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM profiles WHERE organization_id = ANY($1)`, pq.Array(orgIDsAsStrings(allOrgs))); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM users WHERE organization_id = ANY($1)`, pq.Array(orgIDsAsStrings(allOrgs))); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM organizations WHERE id = ANY($1)`, pq.Array(orgIDsAsStrings(allOrgs))); err != nil {
		return err
	}
	return nil
}

// orgIDsAsStrings converts a []uuid.UUID into the []string shape
// pq.Array expects for UUID ANY comparisons.
func orgIDsAsStrings(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}
