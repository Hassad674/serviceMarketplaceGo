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
//
// gosec G202 suppression rationale: `sets` contains only literal
// column-name fragments — `availability_status = $N` and
// `referrer_availability_status = $N` — built from constants in this
// function. The two AvailabilityStatus values reach the query via $N
// positional args, never via the SQL text. The sql_injection_test.go
// suite proves attempted injection payloads (`'; DROP TABLE…`,
// `' OR 1=1`, `\x00…`) are bound as opaque text and never executed as
// SQL.
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
	return "UPDATE profiles SET " + strings.Join(sets, ", ") + " WHERE organization_id = $1", args //nolint:gosec // G202: column allowlist + parameterised values, tested
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
