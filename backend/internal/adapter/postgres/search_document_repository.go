package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/search"
)

// SearchDocumentRepository is the PostgreSQL-backed implementation of
// search.SearchDataRepository. It owns the read path that hydrates
// a full SearchDocument worth of signals from the marketplace tables.
//
// Design goals:
//
//   - One query per "shape of data" so the search.Indexer's fan-out
//     hits exactly one round-trip per goroutine — no N+1 hidden
//     inside a scan loop.
//   - Feature-agnostic: the adapter touches every persona's table
//     directly (freelance_profiles, referrer_profiles, profiles) but
//     exposes a single interface (the search.SearchDataRepository)
//     to the rest of the backend. Feature packages never import it.
//   - Timeouts on every query via queryTimeout so a stuck index
//     builder never blocks the outbox worker forever.
//
// The SQL is deliberately straightforward — we rely on Postgres
// indexes (already in place for the profile search path) rather
// than micro-optimising joins. Every query touching > 1000 rows
// is expected to use an existing index. EXPLAIN ANALYZE runs on
// the test fixtures confirm the plans stay within budget.
type SearchDocumentRepository struct {
	db *sql.DB
}

// NewSearchDocumentRepository builds the adapter against a
// connection pool.
func NewSearchDocumentRepository(db *sql.DB) *SearchDocumentRepository {
	return &SearchDocumentRepository{db: db}
}

// (queryTimeout lives in user_repository.go — shared across every
// adapter in this package.)

// LoadActorSignals returns the core profile + shared organization
// data for one actor. Errors:
//   - sql.ErrNoRows is wrapped in a plain "not found" error (caller
//     decides whether it's fatal or expected).
//   - context errors propagate unchanged.
func (r *SearchDocumentRepository) LoadActorSignals(ctx context.Context, orgID uuid.UUID, persona search.Persona) (*search.RawActorSignals, error) {
	switch persona {
	case search.PersonaFreelance:
		return r.loadFreelanceSignals(ctx, orgID)
	case search.PersonaReferrer:
		return r.loadReferrerSignals(ctx, orgID)
	case search.PersonaAgency:
		return r.loadAgencySignals(ctx, orgID)
	default:
		return nil, fmt.Errorf("search repository: unsupported persona %q", persona)
	}
}

// actorSignalsQuery is the CTE-powered SELECT shared by every
// persona variant. The ${profile_table} placeholder is swapped
// at call time to one of freelance_profiles / referrer_profiles,
// which share an identical column set (title, about, video_url,
// availability_status, expertise_domains).
//
// The legacy `profiles` table used by the agency persona has a
// slightly different column list — it lives in loadAgencySignals
// below, which reuses the shared block logic but reads profile
// columns from `profiles` instead.
const actorSignalsQueryTemplate = `
SELECT
    o.id, o.photo_url, o.city, o.country_code, o.latitude, o.longitude,
    o.work_mode, o.languages_professional, o.languages_conversational,
    o.name,
    p.title, p.about, p.video_url, p.availability_status, p.expertise_domains,
    p.created_at, p.updated_at,
    (SELECT COUNT(*) FROM social_links sl WHERE sl.organization_id = o.id AND sl.persona = $2) AS social_count,
    (SELECT MAX(u.last_active_at) FROM users u WHERE u.organization_id = o.id) AS last_active_at
FROM organizations o
JOIN ${profile_table} p ON p.organization_id = o.id
WHERE o.id = $1
`

// loadFreelanceSignals is the freelance persona variant. Reads from
// freelance_profiles + organizations, joined on organization_id.
func (r *SearchDocumentRepository) loadFreelanceSignals(ctx context.Context, orgID uuid.UUID) (*search.RawActorSignals, error) {
	query := strings.ReplaceAll(actorSignalsQueryTemplate, "${profile_table}", "freelance_profiles")
	return r.scanPersonaSignals(ctx, query, orgID, search.PersonaFreelance, "freelance")
}

// loadReferrerSignals is the referrer persona variant.
func (r *SearchDocumentRepository) loadReferrerSignals(ctx context.Context, orgID uuid.UUID) (*search.RawActorSignals, error) {
	query := strings.ReplaceAll(actorSignalsQueryTemplate, "${profile_table}", "referrer_profiles")
	return r.scanPersonaSignals(ctx, query, orgID, search.PersonaReferrer, "referrer")
}

// loadAgencySignals reads from the legacy `profiles` table (used by
// agencies). The column set differs from freelance/referrer so we
// have a dedicated query rather than shoe-horning it into the
// template.
func (r *SearchDocumentRepository) loadAgencySignals(ctx context.Context, orgID uuid.UUID) (*search.RawActorSignals, error) {
	const query = `
SELECT
    o.id, o.photo_url, o.city, o.country_code, o.latitude, o.longitude,
    o.work_mode, o.languages_professional, o.languages_conversational,
    o.name,
    p.title, p.about, p.presentation_video_url, p.availability_status,
    ARRAY[]::text[] AS expertise_domains,
    p.created_at, p.updated_at,
    (SELECT COUNT(*) FROM social_links sl WHERE sl.organization_id = o.id AND sl.persona = $2) AS social_count,
    (SELECT MAX(u.last_active_at) FROM users u WHERE u.organization_id = o.id) AS last_active_at
FROM organizations o
JOIN profiles p ON p.organization_id = o.id
WHERE o.id = $1
`
	return r.scanPersonaSignals(ctx, query, orgID, search.PersonaAgency, "agency")
}

// scanPersonaSignals runs the query built by the persona-specific
// helpers and scans the row into a RawActorSignals. Shared so the
// three variants produce identical results beyond the table source.
func (r *SearchDocumentRepository) scanPersonaSignals(ctx context.Context, query string, orgID uuid.UUID, persona search.Persona, personaKey string) (*search.RawActorSignals, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var row rawSignalsRow
	err := r.db.QueryRowContext(ctx, query, orgID, personaKey).Scan(
		&row.id, &row.photoURL, &row.city, &row.countryCode, &row.latitude, &row.longitude,
		pq.Array(&row.workMode), pq.Array(&row.languagesPro), pq.Array(&row.languagesConv),
		&row.displayName,
		&row.title, &row.about, &row.videoURL, &row.availabilityStatus, pq.Array(&row.expertiseDomains),
		&row.createdAt, &row.updatedAt,
		&row.socialCount, &row.lastActiveAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("search repository: %s profile not found for org %s", persona, orgID)
	}
	if err != nil {
		return nil, fmt.Errorf("search repository: load %s signals: %w", persona, err)
	}
	return row.toRawActorSignals(persona), nil
}

// rawSignalsRow is the scan target for scanPersonaSignals. Named so
// the Scan argument list stays readable and the conversion to the
// RawActorSignals lives in one place.
type rawSignalsRow struct {
	id                  uuid.UUID
	displayName         string
	photoURL            string
	city                string
	countryCode         string
	latitude            sql.NullFloat64
	longitude           sql.NullFloat64
	workMode            []string
	languagesPro        []string
	languagesConv       []string
	title               string
	about               string
	videoURL            string
	availabilityStatus  string
	expertiseDomains    []string
	createdAt           time.Time
	updatedAt           time.Time
	socialCount         int
	lastActiveAt        sql.NullTime
}

// toRawActorSignals converts the scan row into the search package's
// signals struct. Handles the SQL null types explicitly so the
// downstream ranking code never has to worry about NULLs.
func (r rawSignalsRow) toRawActorSignals(persona search.Persona) *search.RawActorSignals {
	var lat, lng *float64
	if r.latitude.Valid {
		v := r.latitude.Float64
		lat = &v
	}
	if r.longitude.Valid {
		v := r.longitude.Float64
		lng = &v
	}
	lastActive := time.Time{}
	if r.lastActiveAt.Valid {
		lastActive = r.lastActiveAt.Time
	}
	return &search.RawActorSignals{
		OrganizationID:          r.id,
		Persona:                 persona,
		IsPublished:             true, // phase 1: every profile with a persona row is published
		DisplayName:             r.displayName,
		Title:                   r.title,
		About:                   r.about,
		PhotoURL:                r.photoURL,
		VideoURL:                r.videoURL,
		City:                    r.city,
		CountryCode:             r.countryCode,
		Latitude:                lat,
		Longitude:               lng,
		WorkMode:                r.workMode,
		LanguagesProfessional:   r.languagesPro,
		LanguagesConversational: r.languagesConv,
		AvailabilityStatus:      r.availabilityStatus,
		ExpertiseDomains:        r.expertiseDomains,
		SocialLinksCount:        r.socialCount,
		LastActiveAt:            lastActive,
		CreatedAt:               r.createdAt,
		UpdatedAt:               r.updatedAt,
	}
}

// LoadSkills returns the persona's skills as stored in the
// profile_skills table (shared across personas).
func (r *SearchDocumentRepository) LoadSkills(ctx context.Context, orgID uuid.UUID, _ search.Persona) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx,
		`SELECT skill_text FROM profile_skills
		 WHERE organization_id = $1
		 ORDER BY position ASC`, orgID)
	if err != nil {
		return nil, fmt.Errorf("search repository: load skills: %w", err)
	}
	defer rows.Close()

	skills := make([]string, 0, 16)
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("search repository: scan skill: %w", err)
		}
		skills = append(skills, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search repository: iterate skills: %w", err)
	}
	return skills, nil
}

// LoadPricing returns the persona's pricing row or HasPricing=false
// when none has been set. Each persona has its own pricing table;
// the query runs against the right one based on the persona param.
func (r *SearchDocumentRepository) LoadPricing(ctx context.Context, orgID uuid.UUID, persona search.Persona) (*search.RawPricing, error) {
	query, err := pricingQueryFor(persona)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var p pricingRow
	scanErr := r.db.QueryRowContext(ctx, query, orgID).Scan(
		&p.pricingType, &p.minAmount, &p.maxAmount, &p.currency, &p.negotiable,
	)
	if errors.Is(scanErr, sql.ErrNoRows) {
		return &search.RawPricing{HasPricing: false}, nil
	}
	if scanErr != nil {
		return nil, fmt.Errorf("search repository: load pricing: %w", scanErr)
	}
	return p.toRawPricing(), nil
}

// pricingQueryFor returns the right SELECT for the persona's
// dedicated pricing table. Kept as a lookup so adding a new
// persona is a one-line change.
func pricingQueryFor(persona search.Persona) (string, error) {
	switch persona {
	case search.PersonaFreelance:
		return `
SELECT fp.pricing_type, fp.min_amount, fp.max_amount, fp.currency, fp.negotiable
FROM freelance_pricing fp
JOIN freelance_profiles pr ON pr.id = fp.profile_id
WHERE pr.organization_id = $1`, nil
	case search.PersonaReferrer:
		return `
SELECT rp.pricing_type, rp.min_amount, rp.max_amount, rp.currency, rp.negotiable
FROM referrer_pricing rp
JOIN referrer_profiles pr ON pr.id = rp.profile_id
WHERE pr.organization_id = $1`, nil
	case search.PersonaAgency:
		return `
SELECT pp.pricing_type, pp.min_amount, pp.max_amount, pp.currency, pp.negotiable
FROM profile_pricing pp
WHERE pp.organization_id = $1`, nil
	}
	return "", fmt.Errorf("search repository: pricing not implemented for persona %q", persona)
}

// pricingRow is the scan target for LoadPricing, with nullable
// min/max amounts (the schema stores max_amount as NULL when the
// pricing type is "project_from" or "daily").
type pricingRow struct {
	pricingType string
	minAmount   sql.NullInt64
	maxAmount   sql.NullInt64
	currency    string
	negotiable  bool
}

func (p pricingRow) toRawPricing() *search.RawPricing {
	var minPtr, maxPtr *int64
	if p.minAmount.Valid {
		v := p.minAmount.Int64
		minPtr = &v
	}
	if p.maxAmount.Valid {
		v := p.maxAmount.Int64
		maxPtr = &v
	}
	return &search.RawPricing{
		Type:       p.pricingType,
		MinAmount:  minPtr,
		MaxAmount:  maxPtr,
		Currency:   p.currency,
		Negotiable: p.negotiable,
		HasPricing: true,
	}
}

// LoadRatingAggregate computes avg + count of reviews where the
// actor was the reviewed party. Only published, client→provider
// reviews are counted — internal / unpublished ones stay out of
// the ranking.
func (r *SearchDocumentRepository) LoadRatingAggregate(ctx context.Context, orgID uuid.UUID) (*search.RawRatingAggregate, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var avg sql.NullFloat64
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(AVG(global_rating), 0), COUNT(*)
		 FROM reviews
		 WHERE reviewed_organization_id = $1
		   AND side = 'client_to_provider'
		   AND published_at IS NOT NULL`, orgID).Scan(&avg, &count)
	if err != nil {
		return nil, fmt.Errorf("search repository: load rating aggregate: %w", err)
	}
	return &search.RawRatingAggregate{Average: avg.Float64, Count: count}, nil
}

// LoadEarningsAggregate sums released milestone amounts + completed
// project count for proposals where the actor was the provider.
func (r *SearchDocumentRepository) LoadEarningsAggregate(ctx context.Context, orgID uuid.UUID) (*search.RawEarningsAggregate, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// The provider_id column on proposals holds the USER id of the
	// provider. Join through users.organization_id to map to the
	// actor organisation.
	var total sql.NullInt64
	var completed int
	err := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(pm.amount), 0),
		        COUNT(DISTINCT pm.proposal_id) FILTER (WHERE pm.status = 'released')
		 FROM proposal_milestones pm
		 JOIN proposals pr ON pr.id = pm.proposal_id
		 JOIN users u ON u.id = pr.provider_id
		 WHERE u.organization_id = $1
		   AND pm.status = 'released'`, orgID).Scan(&total, &completed)
	if err != nil {
		return nil, fmt.Errorf("search repository: load earnings aggregate: %w", err)
	}
	return &search.RawEarningsAggregate{
		TotalAmount:       total.Int64,
		CompletedProjects: completed,
	}, nil
}

// LoadVerificationStatus reports whether the organisation has a
// Stripe Connect account in good standing. Without a dedicated
// kyc_verifications table we use the existing signal: a non-null
// stripe_account_id means the user has completed the Stripe
// embedded KYC flow.
//
// Phase 4 may introduce a finer-grained `kyc_verifications` table
// once we surface per-document verification in the admin; for
// phase 1 this is the pragmatic truth source.
func (r *SearchDocumentRepository) LoadVerificationStatus(ctx context.Context, orgID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var hasAccount bool
	err := r.db.QueryRowContext(ctx,
		`SELECT stripe_account_id IS NOT NULL
		 FROM organizations
		 WHERE id = $1`, orgID).Scan(&hasAccount)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("search repository: load verification: %w", err)
	}
	return hasAccount, nil
}

// LoadMessagingSignals computes a simple response-rate proxy: the
// ratio of proposals the actor replied to (accepted OR declined)
// over the total number of proposals they received. Phase 1 keeps
// it simple — finer signals (reply latency, conversation opens)
// land in phase 3 once the messaging analytics table is in place.
func (r *SearchDocumentRepository) LoadMessagingSignals(ctx context.Context, orgID uuid.UUID) (*search.RawMessagingSignals, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var responseRate sql.NullFloat64
	err := r.db.QueryRowContext(ctx,
		`SELECT CASE
		           WHEN COUNT(*) = 0 THEN NULL
		           ELSE (COUNT(*) FILTER (WHERE accepted_at IS NOT NULL OR declined_at IS NOT NULL))::float / COUNT(*)::float
		        END
		 FROM proposals
		 WHERE organization_id = $1
		    OR recipient_id IN (SELECT id FROM users WHERE organization_id = $1)`, orgID).Scan(&responseRate)
	if err != nil {
		return nil, fmt.Errorf("search repository: load messaging signals: %w", err)
	}
	return &search.RawMessagingSignals{ResponseRate: responseRate.Float64}, nil
}

// Compile-time check: the adapter must satisfy the port. If the
// interface ever grows a method we get a clear build error here
// instead of an opaque nil-interface panic at wiring time.
var _ search.SearchDataRepository = (*SearchDocumentRepository)(nil)
