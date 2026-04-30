// LEGACY AGENCY-ONLY repository.
//
// Since the split-profile refactor (migrations 096-104) this file
// backs ONLY the profiles table rows for agency organizations —
// the provider_personal rows were migrated out to the split
// freelance_profiles / referrer_profiles tables in migration 104.
// Do NOT extend this file for provider_personal use cases; a
// follow-up refactor will split the agency path into its own
// aggregate and retire this file.

package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/cursor"
)

// ProfileRepository is the PostgreSQL-backed implementation of
// repository.ProfileRepository. Migration 083 added the Tier 1
// completion blocks (location, languages, availability) as extra
// columns on the profiles row — the create/update/read queries in
// this file select every one of them so the domain struct is
// always hydrated with the full profile state.
type ProfileRepository struct {
	db *sql.DB
}

func NewProfileRepository(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

// profileSelectColumns enumerates every column the adapter reads
// when hydrating a *profile.Profile. Centralised in a const so the
// GetByOrganizationID, ensureProfile, and future batch reads stay
// in sync — adding a new column means updating this string and the
// paired Scan call, nothing else.
const profileSelectColumns = `
	organization_id, title, about, photo_url, presentation_video_url,
	referrer_about, referrer_video_url, client_description,
	city, country_code, latitude, longitude, work_mode, travel_radius_km,
	languages_professional, languages_conversational,
	availability_status, referrer_availability_status,
	created_at, updated_at`

func (r *ProfileRepository) Create(ctx context.Context, p *profile.Profile) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// The create path only seeds the classic columns (title/about/
	// photo/...). Tier 1 blocks receive their database defaults
	// (empty strings, empty arrays, 'available_now') so a brand-new
	// profile appears consistently across every read.
	query := `
		INSERT INTO profiles (
			organization_id, title, about, photo_url, presentation_video_url,
			referrer_about, referrer_video_url, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (organization_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query,
		p.OrganizationID, p.Title, p.About, p.PhotoURL,
		p.PresentationVideoURL, p.ReferrerAbout, p.ReferrerVideoURL,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	return nil
}

func (r *ProfileRepository) GetByOrganizationID(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	p, err := r.queryByOrgID(ctx, orgID)
	if err == nil {
		return p, nil
	}

	if !errors.Is(err, profile.ErrProfileNotFound) {
		return nil, err
	}

	return r.ensureProfile(ctx, orgID)
}

// Update rewrites the "classic" profile fields (title, about,
// queryUpdateProfile is shared between the pool-bound Update and
// the tx-bound UpdateTx so the column list cannot drift.
const queryUpdateProfile = `
	UPDATE profiles
	SET title = $2, about = $3, photo_url = $4, presentation_video_url = $5, referrer_about = $6, referrer_video_url = $7
	WHERE organization_id = $1`

// photo, videos, referrer about). The Tier 1 blocks have their own
// focused update methods — this function intentionally leaves them
// alone so a caller saving a new title cannot accidentally clobber
// the location / languages / availability state.
func (r *ProfileRepository) Update(ctx context.Context, p *profile.Profile) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, queryUpdateProfile,
		p.OrganizationID, p.Title, p.About, p.PhotoURL,
		p.PresentationVideoURL, p.ReferrerAbout, p.ReferrerVideoURL,
	)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	if rows == 0 {
		return profile.ErrProfileNotFound
	}

	return nil
}

// UpdateTx is the outbox-aware variant of Update.
func (r *ProfileRepository) UpdateTx(ctx context.Context, tx *sql.Tx, p *profile.Profile) error {
	if tx == nil {
		return fmt.Errorf("update profile: tx is required")
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := tx.ExecContext(ctx, queryUpdateProfile,
		p.OrganizationID, p.Title, p.About, p.PhotoURL,
		p.PresentationVideoURL, p.ReferrerAbout, p.ReferrerVideoURL,
	)
	if err != nil {
		return fmt.Errorf("failed to update profile in tx: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected in tx: %w", err)
	}
	if rows == 0 {
		return profile.ErrProfileNotFound
	}
	return nil
}

// queryUpdateProfileLocation is shared between the pool-bound and
// tx-bound location writes.
const queryUpdateProfileLocation = `
	UPDATE profiles
	SET city              = $2,
	    country_code      = $3,
	    latitude          = $4,
	    longitude         = $5,
	    work_mode         = $6,
	    travel_radius_km  = $7
	WHERE organization_id = $1`

// UpdateLocation writes the entire location block (city, country,
// coordinates, work modes, travel radius) in a single SQL UPDATE.
// Every column is always written — a nil pointer clears the column
// to NULL at the database level, preserving the "atomic block"
// semantics the service layer relies on.
func (r *ProfileRepository) UpdateLocation(ctx context.Context, orgID uuid.UUID, input repository.LocationInput) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	args := buildUpdateLocationArgs(orgID, input)
	result, err := r.db.ExecContext(ctx, queryUpdateProfileLocation, args...)
	if err != nil {
		return fmt.Errorf("failed to update profile location: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for location update: %w", err)
	}
	if rows == 0 {
		return profile.ErrProfileNotFound
	}
	return nil
}

// UpdateLocationTx is the outbox-aware variant of UpdateLocation.
func (r *ProfileRepository) UpdateLocationTx(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, input repository.LocationInput) error {
	if tx == nil {
		return fmt.Errorf("update profile location: tx is required")
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	args := buildUpdateLocationArgs(orgID, input)
	result, err := tx.ExecContext(ctx, queryUpdateProfileLocation, args...)
	if err != nil {
		return fmt.Errorf("failed to update profile location in tx: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for location update in tx: %w", err)
	}
	if rows == 0 {
		return profile.ErrProfileNotFound
	}
	return nil
}

// buildUpdateLocationArgs returns the SQL args for the location
// UPDATE. Centralised so the pool and tx code paths cannot drift in
// argument ordering or NULL handling.
func buildUpdateLocationArgs(orgID uuid.UUID, input repository.LocationInput) []any {
	workMode := input.WorkMode
	if workMode == nil {
		workMode = []string{}
	}
	return []any{
		orgID,
		input.City,
		input.CountryCode,
		nullFloat(input.Latitude),
		nullFloat(input.Longitude),
		pq.Array(workMode),
		nullInt(input.TravelRadiusKm),
	}
}

// queryUpdateProfileLanguages is shared between the pool-bound and
// tx-bound language writes.
const queryUpdateProfileLanguages = `
	UPDATE profiles
	SET languages_professional   = $2,
	    languages_conversational = $3
	WHERE organization_id = $1`

// UpdateLanguages replaces the two language arrays atomically. Both
// slices are persisted verbatim — the caller (app/profile service)
// is responsible for normalization and dedup.
func (r *ProfileRepository) UpdateLanguages(ctx context.Context, orgID uuid.UUID, professional, conversational []string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if professional == nil {
		professional = []string{}
	}
	if conversational == nil {
		conversational = []string{}
	}
	result, err := r.db.ExecContext(ctx, queryUpdateProfileLanguages,
		orgID,
		pq.Array(professional),
		pq.Array(conversational),
	)
	if err != nil {
		return fmt.Errorf("failed to update profile languages: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for languages update: %w", err)
	}
	if rows == 0 {
		return profile.ErrProfileNotFound
	}
	return nil
}

// UpdateLanguagesTx is the outbox-aware variant of UpdateLanguages.
func (r *ProfileRepository) UpdateLanguagesTx(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, professional, conversational []string) error {
	if tx == nil {
		return fmt.Errorf("update profile languages: tx is required")
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if professional == nil {
		professional = []string{}
	}
	if conversational == nil {
		conversational = []string{}
	}
	result, err := tx.ExecContext(ctx, queryUpdateProfileLanguages,
		orgID,
		pq.Array(professional),
		pq.Array(conversational),
	)
	if err != nil {
		return fmt.Errorf("failed to update profile languages in tx: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for languages update in tx: %w", err)
	}
	if rows == 0 {
		return profile.ErrProfileNotFound
	}
	return nil
}

// UpdateAvailability patches one or both availability columns. Nil
// pointers mean "leave this column alone" — the UPDATE is built
// dynamically so omitted slots keep their current value. This
// prevents the direct-profile save flow from clobbering the
// referrer column (and vice versa) after they were split across
// two independent pages.
func (r *ProfileRepository) UpdateAvailability(ctx context.Context, orgID uuid.UUID, direct *profile.AvailabilityStatus, referrer *profile.AvailabilityStatus) error {
	if direct == nil && referrer == nil {
		return profile.ErrInvalidAvailabilityStatus
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query, args := buildUpdateAvailabilityQuery(orgID, direct, referrer)
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update profile availability: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for availability update: %w", err)
	}
	if rows == 0 {
		return profile.ErrProfileNotFound
	}
	return nil
}

// UpdateAvailabilityTx is the outbox-aware variant of
// UpdateAvailability.
func (r *ProfileRepository) UpdateAvailabilityTx(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, direct *profile.AvailabilityStatus, referrer *profile.AvailabilityStatus) error {
	if tx == nil {
		return fmt.Errorf("update profile availability: tx is required")
	}
	if direct == nil && referrer == nil {
		return profile.ErrInvalidAvailabilityStatus
	}
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query, args := buildUpdateAvailabilityQuery(orgID, direct, referrer)
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update profile availability in tx: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for availability update in tx: %w", err)
	}
	if rows == 0 {
		return profile.ErrProfileNotFound
	}
	return nil
}

// buildUpdateAvailabilityQuery assembles the dynamic UPDATE for the
// availability columns. The pool and tx code paths share this so the
// SET clause and the placeholder ordering cannot drift.
func buildUpdateAvailabilityQuery(orgID uuid.UUID, direct *profile.AvailabilityStatus, referrer *profile.AvailabilityStatus) (string, []any) {
	sets := make([]string, 0, 2)
	args := make([]any, 0, 3)
	args = append(args, orgID)
	if direct != nil {
		args = append(args, string(*direct))
		sets = append(sets, fmt.Sprintf("availability_status = $%d", len(args)))
	}
	if referrer != nil {
		args = append(args, string(*referrer))
		sets = append(sets, fmt.Sprintf("referrer_availability_status = $%d", len(args)))
	}
	return "UPDATE profiles SET " + strings.Join(sets, ", ") + " WHERE organization_id = $1", args
}

// UpdateClientDescription writes the client_description column in a
// single SQL UPDATE. Scoped to the client profile facet — the other
// columns (title, about, referrer_about, Tier 1 blocks) are untouched
// so the provider-facing state cannot be clobbered by a client-facing
// save.
func (r *ProfileRepository) UpdateClientDescription(ctx context.Context, orgID uuid.UUID, clientDescription string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		UPDATE profiles
		SET client_description = $2
		WHERE organization_id = $1`

	result, err := r.db.ExecContext(ctx, query, orgID, clientDescription)
	if err != nil {
		return fmt.Errorf("failed to update profile client_description: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected for client_description update: %w", err)
	}
	if rows == 0 {
		return profile.ErrProfileNotFound
	}
	return nil
}

// queryByOrgID fetches the full profile row (including every Tier 1
// column). Extracted from GetByOrganizationID so the auto-create
// fallback can reuse it without paying a second SELECT.
func (r *ProfileRepository) queryByOrgID(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	query := `SELECT ` + profileSelectColumns + ` FROM profiles WHERE organization_id = $1`

	p := &profile.Profile{}
	var (
		lat, lng          sql.NullFloat64
		travelRadius      sql.NullInt64
		availabilityStr   string
		referrerAvailStr  sql.NullString
		workMode          []string
		languagesPro      []string
		languagesConvList []string
	)
	err := r.db.QueryRowContext(ctx, query, orgID).Scan(
		&p.OrganizationID, &p.Title, &p.About, &p.PhotoURL,
		&p.PresentationVideoURL, &p.ReferrerAbout, &p.ReferrerVideoURL,
		&p.ClientDescription,
		&p.City, &p.CountryCode, &lat, &lng,
		pq.Array(&workMode), &travelRadius,
		pq.Array(&languagesPro), pq.Array(&languagesConvList),
		&availabilityStr, &referrerAvailStr,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, profile.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	hydrateProfileTier1(p, profileTier1Row{
		Latitude:          lat,
		Longitude:         lng,
		WorkMode:          workMode,
		TravelRadiusKm:    travelRadius,
		LanguagesPro:      languagesPro,
		LanguagesConv:     languagesConvList,
		Availability:      availabilityStr,
		ReferrerAvailable: referrerAvailStr,
	})
	return p, nil
}

// profileTier1Row is the raw-sql shape of the Tier 1 columns. Kept
// local to this file so hydrateProfileTier1 stays a thin type-swap
// and the Scan signature never exceeds the parameter budget.
type profileTier1Row struct {
	Latitude          sql.NullFloat64
	Longitude         sql.NullFloat64
	WorkMode          []string
	TravelRadiusKm    sql.NullInt64
	LanguagesPro      []string
	LanguagesConv     []string
	Availability      string
	ReferrerAvailable sql.NullString
}

// hydrateProfileTier1 copies the raw SQL Tier 1 values onto the
// domain struct, translating SQL nullables to their *T / typed-enum
// equivalents. Separated from the main Scan so the 50-line / nested
// caps stay respected in queryByOrgID.
func hydrateProfileTier1(p *profile.Profile, row profileTier1Row) {
	if row.Latitude.Valid {
		v := row.Latitude.Float64
		p.Latitude = &v
	}
	if row.Longitude.Valid {
		v := row.Longitude.Float64
		p.Longitude = &v
	}
	p.WorkMode = nilToEmpty(row.WorkMode)
	if row.TravelRadiusKm.Valid {
		v := int(row.TravelRadiusKm.Int64)
		p.TravelRadiusKm = &v
	}
	p.LanguagesProfessional = nilToEmpty(row.LanguagesPro)
	p.LanguagesConversational = nilToEmpty(row.LanguagesConv)
	p.AvailabilityStatus = profile.AvailabilityStatus(row.Availability)
	if row.ReferrerAvailable.Valid {
		a := profile.AvailabilityStatus(row.ReferrerAvailable.String)
		p.ReferrerAvailabilityStatus = &a
	}
}

// SearchPublic returns orgs filtered by type and referrer flag, paginated.
//
// Review aggregation still happens on the owner user row — in phase R3
// the reviews table gets its own organization_id and this will flip
// to joining on reviews.reviewed_organization_id. Until then the query
// preserves the same aggregate because every agency/enterprise/provider_personal
// has a single owner.
//
// Since the split-profile refactor the query sources persona-specific
// fields from three distinct tables depending on org type:
//
//   - agency / enterprise → legacy `profiles` row (joined as p).
//   - provider_personal (freelance directory) → `freelance_profiles`
//     (joined as fp). Title and availability come from fp; shared
//     fields (photo, city, country, languages) come from organizations.
//   - provider_personal (referrer directory) → `referrer_profiles`
//     (joined as rp). Same shared-fields rule; title and availability
//     come from rp.
//
// The `referrerOnly` flag toggles the persona: when true the SELECT
// surfaces fields from `referrer_profiles`; otherwise it prefers the
// freelance columns for provider_personal and falls back to the
// legacy profiles row for agency / enterprise.
//
// Two aggregate fields (total_earned, completed_projects) are batched
// into the same query via a LEFT JOIN subquery that walks
// `proposal_milestones` → `proposals` keyed on the org owner. No N+1:
// the subquery groups once per owner and the search page picks the
// pre-aggregated row via a cheap equality join. The whole query stays
// sub-50ms on the target dataset size per EXPLAIN ANALYZE.
func (r *ProfileRepository) SearchPublic(ctx context.Context, orgTypeFilter string, referrerOnly bool, cursorStr string, limit int) ([]*profile.PublicProfile, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	// persona_title / persona_availability pick the right persona's
	// values via COALESCE: referrer directory → rp first, freelance
	// directory → fp first, agency / enterprise → legacy p. The order
	// in COALESCE matches the precedence the handler expects per
	// persona, so a single SELECT serves every directory.
	personaTitleExpr := "COALESCE(fp.title, p.title, '')"
	personaAvailExpr := "COALESCE(fp.availability_status, p.availability_status, '')"
	if referrerOnly {
		personaTitleExpr = "COALESCE(rp.title, '')"
		personaAvailExpr = "COALESCE(rp.availability_status, '')"
	}

	base := `
		SELECT o.id, o.owner_user_id, o.name, o.type,
		       ` + personaTitleExpr + `,
		       COALESCE(o.photo_url, COALESCE(p.photo_url, '')),
		       COALESCE(u.referrer_enabled, false),
		       o.created_at,
		       COALESCE(rv.avg_rating, 0)::float8, COALESCE(rv.review_count, 0)::int,
		       COALESCE(o.city, COALESCE(p.city, '')),
		       COALESCE(o.country_code, COALESCE(p.country_code, '')),
		       COALESCE(o.languages_professional, COALESCE(p.languages_professional, '{}'))::text[],
		       ` + personaAvailExpr + `,
		       COALESCE(pm.total_earned, 0)::bigint,
		       COALESCE(pm.completed_projects, 0)::int
		FROM organizations o
		LEFT JOIN profiles p            ON p.organization_id = o.id
		LEFT JOIN freelance_profiles fp ON fp.organization_id = o.id
		LEFT JOIN referrer_profiles rp  ON rp.organization_id = o.id
		LEFT JOIN users u               ON u.id = o.owner_user_id
		LEFT JOIN (
			SELECT reviewed_id,
			       AVG(global_rating)::float8 AS avg_rating,
			       COUNT(*)::int              AS review_count
			FROM reviews rv
			WHERE NOT EXISTS (
			    SELECT 1 FROM moderation_results mr
			     WHERE mr.content_type = 'review'
			       AND mr.content_id = rv.id
			       AND mr.status IN ('hidden', 'deleted')
			       AND mr.reviewed_at IS NULL
			)
			GROUP BY reviewed_id
		) rv ON rv.reviewed_id = o.owner_user_id
		LEFT JOIN (
			SELECT pr.provider_id,
			       SUM(m.amount)::bigint               AS total_earned,
			       COUNT(DISTINCT m.proposal_id)::int  AS completed_projects
			FROM proposal_milestones m
			JOIN proposals pr ON pr.id = m.proposal_id
			WHERE m.status = 'released'
			GROUP BY pr.provider_id
		) pm ON pm.provider_id = o.owner_user_id
		WHERE ($1 = '' OR o.type = $1)
		  AND ($2 = false OR (o.type = 'provider_personal' AND COALESCE(u.referrer_enabled, false) = true))`

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		query := base + `
			ORDER BY o.created_at DESC, o.id DESC
			LIMIT $3`
		rows, err = r.db.QueryContext(ctx, query, orgTypeFilter, referrerOnly, limit+1)
	} else {
		c, decErr := cursor.Decode(cursorStr)
		if decErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", decErr)
		}
		query := base + `
			AND (o.created_at, o.id) < ($3, $4)
			ORDER BY o.created_at DESC, o.id DESC
			LIMIT $5`
		rows, err = r.db.QueryContext(ctx, query, orgTypeFilter, referrerOnly, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to search profiles: %w", err)
	}
	defer rows.Close()

	var results []*profile.PublicProfile
	for rows.Next() {
		pp := &profile.PublicProfile{}
		var languages pq.StringArray
		if err := rows.Scan(
			&pp.OrganizationID, &pp.OwnerUserID, &pp.Name, &pp.OrgType,
			&pp.Title, &pp.PhotoURL, &pp.ReferrerEnabled,
			&pp.CreatedAt, &pp.AverageRating, &pp.ReviewCount,
			&pp.City, &pp.CountryCode, &languages, &pp.AvailabilityStatus,
			&pp.TotalEarned, &pp.CompletedProjects,
		); err != nil {
			return nil, "", fmt.Errorf("failed to scan public profile: %w", err)
		}
		pp.LanguagesProfessional = []string(languages)
		results = append(results, pp)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration error: %w", err)
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.OrganizationID)
		results = results[:limit]
	}

	if results == nil {
		results = []*profile.PublicProfile{}
	}

	return results, nextCursor, nil
}

func (r *ProfileRepository) GetPublicProfilesByOrgIDs(ctx context.Context, orgIDs []uuid.UUID) ([]*profile.PublicProfile, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if len(orgIDs) == 0 {
		return []*profile.PublicProfile{}, nil
	}

	query := `
		SELECT o.id, o.owner_user_id, o.name, o.type,
		       COALESCE(p.title, ''), COALESCE(p.photo_url, ''),
		       COALESCE(u.referrer_enabled, false),
		       o.created_at,
		       COALESCE(r.avg_rating, 0)::float8, COALESCE(r.review_count, 0)::int
		FROM organizations o
		LEFT JOIN profiles p ON p.organization_id = o.id
		LEFT JOIN users u    ON u.id = o.owner_user_id
		LEFT JOIN (
			SELECT reviewed_id,
			       AVG(global_rating)::float8 AS avg_rating,
			       COUNT(*)::int              AS review_count
			FROM reviews rv
			WHERE NOT EXISTS (
			    SELECT 1 FROM moderation_results mr
			     WHERE mr.content_type = 'review'
			       AND mr.content_id = rv.id
			       AND mr.status IN ('hidden', 'deleted')
			       AND mr.reviewed_at IS NULL
			)
			GROUP BY reviewed_id
		) r ON r.reviewed_id = o.owner_user_id
		WHERE o.id = ANY($1)`

	ids := make([]string, len(orgIDs))
	for i, id := range orgIDs {
		ids[i] = id.String()
	}

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("get public profiles by org ids: %w", err)
	}
	defer rows.Close()

	var results []*profile.PublicProfile
	for rows.Next() {
		pp := &profile.PublicProfile{}
		if err := rows.Scan(
			&pp.OrganizationID, &pp.OwnerUserID, &pp.Name, &pp.OrgType,
			&pp.Title, &pp.PhotoURL, &pp.ReferrerEnabled,
			&pp.CreatedAt, &pp.AverageRating, &pp.ReviewCount,
		); err != nil {
			return nil, fmt.Errorf("scan public profile: %w", err)
		}
		results = append(results, pp)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if results == nil {
		results = []*profile.PublicProfile{}
	}
	return results, nil
}

// OrgProfilesByUserIDs joins users→organizations→profiles so callers
// that hold a user_id (job application applicant, review reviewer)
// can surface the matching org's public profile. Returns a map keyed
// by user_id for easy lookup — users without an org or without a
// profile row simply don't appear in the map.
func (r *ProfileRepository) OrgProfilesByUserIDs(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*profile.PublicProfile, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	out := make(map[uuid.UUID]*profile.PublicProfile, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}

	query := `
		SELECT u.id AS user_id,
		       o.id, o.owner_user_id, o.name, o.type,
		       COALESCE(p.title, ''), COALESCE(p.photo_url, ''),
		       COALESCE(u.referrer_enabled, false),
		       o.created_at,
		       COALESCE(r.avg_rating, 0)::float8, COALESCE(r.review_count, 0)::int
		FROM users u
		JOIN organizations o ON o.id = u.organization_id
		LEFT JOIN profiles p ON p.organization_id = o.id
		LEFT JOIN (
			SELECT reviewed_id,
			       AVG(global_rating)::float8 AS avg_rating,
			       COUNT(*)::int              AS review_count
			FROM reviews rv
			WHERE NOT EXISTS (
			    SELECT 1 FROM moderation_results mr
			     WHERE mr.content_type = 'review'
			       AND mr.content_id = rv.id
			       AND mr.status IN ('hidden', 'deleted')
			       AND mr.reviewed_at IS NULL
			)
			GROUP BY reviewed_id
		) r ON r.reviewed_id = u.id
		WHERE u.id = ANY($1)`

	ids := make([]string, len(userIDs))
	for i, id := range userIDs {
		ids[i] = id.String()
	}

	rows, err := r.db.QueryContext(ctx, query, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("org profiles by user ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var userID uuid.UUID
		pp := &profile.PublicProfile{}
		if err := rows.Scan(
			&userID,
			&pp.OrganizationID, &pp.OwnerUserID, &pp.Name, &pp.OrgType,
			&pp.Title, &pp.PhotoURL, &pp.ReferrerEnabled,
			&pp.CreatedAt, &pp.AverageRating, &pp.ReviewCount,
		); err != nil {
			return nil, fmt.Errorf("scan org profile: %w", err)
		}
		out[userID] = pp
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	return out, nil
}

func (r *ProfileRepository) ensureProfile(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	newProfile := profile.NewProfile(orgID)
	if err := r.Create(ctx, newProfile); err != nil {
		return nil, fmt.Errorf("failed to auto-create profile: %w", err)
	}

	return r.queryByOrgID(ctx, orgID)
}

// ---- small helpers for sql.Null → *T conversions ----

// nullFloat converts a nullable *float64 to a sql.NullFloat64 for
// write paths. nil → Invalid (writes NULL), non-nil → Valid (writes
// the value). Symmetric helper for nullInt / nullString.
func nullFloat(f *float64) sql.NullFloat64 {
	if f == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *f, Valid: true}
}

// nullInt converts a nullable *int to a sql.NullInt64.
func nullInt(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}

// nilToEmpty returns a guaranteed non-nil slice. Postgres TEXT[]
// reads may return nil when the row value is '{}' depending on
// driver version — the domain expects empty (non-nil) slices so
// downstream DTO construction can marshal them to `[]` without a
// nil check.
func nilToEmpty(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}
