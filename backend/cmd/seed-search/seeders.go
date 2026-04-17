package main

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// seeders.go holds the per-persona insertion logic. Every seeder is
// idempotent via ON CONFLICT DO NOTHING + deterministic UUID so a
// re-run overwrites nothing (wipePreviousSeed clears old rows first).

// drawName produces a (firstName, lastName) pair. French and English
// pools are interleaved so the dataset reflects a mixed EU/NA market.
func drawName(r *rand.Rand) (string, string) {
	if r.Intn(2) == 0 {
		return firstNamesFR[r.Intn(len(firstNamesFR))], lastNamesFR[r.Intn(len(lastNamesFR))]
	}
	return firstNamesEN[r.Intn(len(firstNamesEN))], lastNamesEN[r.Intn(len(lastNamesEN))]
}

// composeAbout joins 2-3 snippets from the pool into a bio. The result
// is stable given a seeded rand — running twice yields the same bios.
func composeAbout(r *rand.Rand, pool []string) string {
	idxs := r.Perm(len(pool))
	count := 2 + r.Intn(2)
	parts := make([]string, 0, count)
	for i := 0; i < count && i < len(idxs); i++ {
		parts = append(parts, pool[idxs[i]])
	}
	return strings.Join(parts, " ")
}

// availabilityDBValue maps our bucketed choice ("now", "soon", "not")
// into the DB's constrained enum. Separated from the math/rand caller
// so the constraint stays near the schema it matches.
func availabilityDBValue(bucket string) string {
	switch bucket {
	case "now":
		return "available_now"
	case "soon":
		return "available_soon"
	default:
		return "not_available"
	}
}

func seedFreelance(ctx context.Context, db *sql.DB, index int, r *rand.Rand) error {
	orgID := deterministicUUID(fmt.Sprintf("seedsearch-freelance-org-%d", index))
	userID := deterministicUUID(fmt.Sprintf("seedsearch-freelance-user-%d", index))
	city := citiesPool[r.Intn(len(citiesPool))]
	first, last := drawName(r)
	display := first + " " + last
	email := fmt.Sprintf("freelance-%d-%s@search.seed", index, strings.ToLower(first))

	if err := insertUser(ctx, db, userID, email, first, last, display, "provider", false); err != nil {
		return err
	}
	if err := insertOrganization(ctx, db, orgID, userID, display, "provider_personal", city, r); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE users SET organization_id = $1, last_active_at = $2 WHERE id = $3`,
		orgID, lastActiveAt(r, time.Now()), userID); err != nil {
		return err
	}

	profileID := deterministicUUID(fmt.Sprintf("seedsearch-freelance-profile-%d", index))
	title := titlesFreelance[r.Intn(len(titlesFreelance))]
	about := composeAbout(r, aboutFreelanceSnippets)
	expertise := expertiseForRNG(r)
	availability := availabilityDBValue(availabilityForRNG(r))

	if _, err := db.ExecContext(ctx,
		`INSERT INTO freelance_profiles (id, organization_id, title, about, expertise_domains, availability_status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (organization_id) DO UPDATE
		 SET title = EXCLUDED.title, about = EXCLUDED.about,
		     expertise_domains = EXCLUDED.expertise_domains,
		     availability_status = EXCLUDED.availability_status`,
		profileID, orgID, title, about, pq.Array(expertise), availability); err != nil {
		return fmt.Errorf("insert freelance_profile: %w", err)
	}

	// Pricing — daily rate 400-1200 EUR (40000-120000 cents).
	minPrice := int64(40000 + r.Intn(80001))
	maxPrice := minPrice + int64(10000+r.Intn(30001))
	if _, err := db.ExecContext(ctx,
		`INSERT INTO freelance_pricing (profile_id, pricing_type, min_amount, max_amount, currency, negotiable)
		 VALUES ($1, 'daily', $2, $3, 'EUR', $4)
		 ON CONFLICT (profile_id) DO UPDATE
		 SET min_amount = EXCLUDED.min_amount, max_amount = EXCLUDED.max_amount,
		     negotiable = EXCLUDED.negotiable`,
		profileID, minPrice, maxPrice, r.Intn(100) < 40); err != nil {
		return fmt.Errorf("insert freelance_pricing: %w", err)
	}

	skills := skillsForRNG(r)
	for pos, skill := range skills {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO profile_skills (organization_id, skill_text, position) VALUES ($1, $2, $3)
			 ON CONFLICT DO NOTHING`, orgID, skill, pos); err != nil {
			return fmt.Errorf("insert profile_skill: %w", err)
		}
	}

	return nil
}

func seedAgency(ctx context.Context, db *sql.DB, index int, r *rand.Rand) error {
	orgID := deterministicUUID(fmt.Sprintf("seedsearch-agency-org-%d", index))
	userID := deterministicUUID(fmt.Sprintf("seedsearch-agency-user-%d", index))
	city := citiesPool[r.Intn(len(citiesPool))]
	first, last := drawName(r)
	display := titlesAgency[r.Intn(len(titlesAgency))]
	email := fmt.Sprintf("agency-%d-%s@search.seed", index, strings.ToLower(first))

	if err := insertUser(ctx, db, userID, email, first, last, display, "agency", false); err != nil {
		return err
	}
	if err := insertOrganization(ctx, db, orgID, userID, display, "agency", city, r); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE users SET organization_id = $1, last_active_at = $2 WHERE id = $3`,
		orgID, lastActiveAt(r, time.Now()), userID); err != nil {
		return err
	}

	title := display
	about := composeAbout(r, aboutAgencySnippets)
	if _, err := db.ExecContext(ctx,
		`INSERT INTO profiles (organization_id, title, about, city, country_code, latitude, longitude,
		                       work_mode, languages_professional, photo_url)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		 ON CONFLICT (organization_id) DO UPDATE
		 SET title = EXCLUDED.title, about = EXCLUDED.about`,
		orgID, title, about, city.City, city.Country, city.Lat, city.Lng,
		pq.Array(workModesForRNG(r)), pq.Array(languagesForRNG(r)),
		fmt.Sprintf("https://cdn.search.seed/agency-%d.webp", index),
	); err != nil {
		return fmt.Errorf("insert agency profile: %w", err)
	}

	skills := skillsForRNG(r)
	for pos, skill := range skills {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO profile_skills (organization_id, skill_text, position) VALUES ($1, $2, $3)
			 ON CONFLICT DO NOTHING`, orgID, skill, pos); err != nil {
			return fmt.Errorf("insert agency skill: %w", err)
		}
	}

	// V1 pricing: agencies declare `project_from` on the direct
	// kind — the "à partir de" starting budget shape (5 000 € to
	// 50 000 € in cents). The backend write boundary rejects any
	// other type for agencies, so seeding anything else would make
	// a re-index + re-upsert round-trip inconsistent.
	minBudget := int64(500000 + r.Intn(4500001))
	if _, err := db.ExecContext(ctx,
		`INSERT INTO profile_pricing
		   (organization_id, pricing_kind, pricing_type, min_amount, max_amount, currency, negotiable)
		 VALUES ($1, 'direct', 'project_from', $2, NULL, 'EUR', $3)
		 ON CONFLICT (organization_id, pricing_kind) DO UPDATE
		 SET pricing_type = EXCLUDED.pricing_type,
		     min_amount   = EXCLUDED.min_amount,
		     max_amount   = EXCLUDED.max_amount,
		     currency     = EXCLUDED.currency,
		     negotiable   = EXCLUDED.negotiable`,
		orgID, minBudget, r.Intn(100) < 40); err != nil {
		return fmt.Errorf("insert agency profile_pricing: %w", err)
	}

	return nil
}

func seedReferrer(ctx context.Context, db *sql.DB, index int, r *rand.Rand) error {
	orgID := deterministicUUID(fmt.Sprintf("seedsearch-referrer-org-%d", index))
	userID := deterministicUUID(fmt.Sprintf("seedsearch-referrer-user-%d", index))
	city := citiesPool[r.Intn(len(citiesPool))]
	first, last := drawName(r)
	display := first + " " + last
	email := fmt.Sprintf("referrer-%d-%s@search.seed", index, strings.ToLower(first))

	if err := insertUser(ctx, db, userID, email, first, last, display, "provider", true); err != nil {
		return err
	}
	if err := insertOrganization(ctx, db, orgID, userID, display, "provider_personal", city, r); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx,
		`UPDATE users SET organization_id = $1, last_active_at = $2 WHERE id = $3`,
		orgID, lastActiveAt(r, time.Now()), userID); err != nil {
		return err
	}

	profileID := deterministicUUID(fmt.Sprintf("seedsearch-referrer-profile-%d", index))
	title := titlesReferrer[r.Intn(len(titlesReferrer))]
	about := composeAbout(r, aboutReferrerSnippets)
	expertise := expertiseForRNG(r)
	availability := availabilityDBValue(availabilityForRNG(r))

	if _, err := db.ExecContext(ctx,
		`INSERT INTO referrer_profiles (id, organization_id, title, about, expertise_domains, availability_status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (organization_id) DO UPDATE
		 SET title = EXCLUDED.title, about = EXCLUDED.about,
		     expertise_domains = EXCLUDED.expertise_domains,
		     availability_status = EXCLUDED.availability_status`,
		profileID, orgID, title, about, pq.Array(expertise), availability); err != nil {
		return fmt.Errorf("insert referrer_profile: %w", err)
	}

	// V1 pricing: referrers declare `commission_pct` only — one
	// headline rate between 3 % and 20 %, echoed to both min and max
	// so the backend range validator accepts the payload AND the
	// formatter collapses the display to "N % de commission".
	// Amounts are basis points: 3 % -> 300, 20 % -> 2000.
	pct := int64(300 + r.Intn(1701))
	if _, err := db.ExecContext(ctx,
		`INSERT INTO referrer_pricing
		   (profile_id, pricing_type, min_amount, max_amount, currency, negotiable)
		 VALUES ($1, 'commission_pct', $2, $2, 'pct', $3)
		 ON CONFLICT (profile_id) DO UPDATE
		 SET pricing_type = EXCLUDED.pricing_type,
		     min_amount   = EXCLUDED.min_amount,
		     max_amount   = EXCLUDED.max_amount,
		     currency     = EXCLUDED.currency,
		     negotiable   = EXCLUDED.negotiable`,
		profileID, pct, r.Intn(100) < 40); err != nil {
		return fmt.Errorf("insert referrer_pricing: %w", err)
	}

	return nil
}

// insertUser writes the users row. The hashed_password is a placeholder
// since seed profiles are never meant to log in.
func insertUser(ctx context.Context, db *sql.DB, userID uuid.UUID, email, first, last, display, role string, referrerEnabled bool) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO users (id, email, hashed_password, first_name, last_name, display_name,
		                    role, account_type, referrer_enabled, email_verified)
		 VALUES ($1, $2, 'seed-hash-placeholder', $3, $4, $5, $6, 'marketplace_owner', $7, true)
		 ON CONFLICT (id) DO NOTHING`,
		userID, email, first, last, display, role, referrerEnabled)
	return err
}

// insertOrganization writes the organizations row with the city
// coordinates, work_mode slice, and languages slice sampled from the
// RNG. photo_url is synthesised from the index so cards render.
func insertOrganization(ctx context.Context, db *sql.DB, orgID, userID uuid.UUID, name, orgType string, city struct {
	City, Country string
	Lat, Lng      float64
}, r *rand.Rand) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO organizations (id, owner_user_id, type, name, photo_url, city, country_code, latitude, longitude,
		                            work_mode, languages_professional)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 ON CONFLICT (id) DO NOTHING`,
		orgID, userID, orgType, name,
		fmt.Sprintf("https://cdn.search.seed/%s.webp", orgID),
		city.City, city.Country, city.Lat, city.Lng,
		pq.Array(workModesForRNG(r)), pq.Array(languagesForRNG(r)))
	return err
}
