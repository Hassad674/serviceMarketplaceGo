package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/domain/referrerprofile"
	"marketplace-backend/internal/port/repository"
)

// ReferrerProfileRepository is the PostgreSQL-backed implementation
// of repository.ReferrerProfileRepository. Mirrors the shape of the
// freelance repository with one behavioural difference: the read
// path auto-creates the row on miss (lazy creation) because
// referrer profiles are not bulk-seeded during the split migration.
type ReferrerProfileRepository struct {
	db *sql.DB
}

// NewReferrerProfileRepository returns a repository ready to talk
// to the given *sql.DB.
func NewReferrerProfileRepository(db *sql.DB) *ReferrerProfileRepository {
	return &ReferrerProfileRepository{db: db}
}

// referrerProfileSelectColumns enumerates the columns the adapter
// reads when hydrating a referrer profile + joined shared block.
const referrerProfileSelectColumns = `
	rp.id, rp.organization_id, rp.title, rp.about, rp.video_url,
	rp.availability_status, rp.expertise_domains, rp.created_at, rp.updated_at,
	o.photo_url, o.city, o.country_code, o.latitude, o.longitude,
	o.work_mode, o.travel_radius_km,
	o.languages_professional, o.languages_conversational`

// GetOrCreateByOrgID fetches the referrer profile for the given org
// and JOINs the organization's shared-profile block. If no row
// exists the method inserts a fresh default (title/about/video empty,
// availability = available_now, expertise = empty) and re-fetches.
// Never returns referrerprofile.ErrProfileNotFound — the service
// layer can rely on this method being infallible on the "missing
// row" path.
func (r *ReferrerProfileRepository) GetOrCreateByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.ReferrerProfileView, error) {
	view, err := r.queryByOrgID(ctx, orgID)
	if err == nil {
		return view, nil
	}
	if !errors.Is(err, referrerprofile.ErrProfileNotFound) {
		return nil, err
	}
	if err := r.insertDefault(ctx, orgID); err != nil {
		return nil, err
	}
	return r.queryByOrgID(ctx, orgID)
}

// UpdateCore writes the title / about / video_url triplet.
func (r *ReferrerProfileRepository) UpdateCore(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, `
		UPDATE referrer_profiles
		   SET title = $2, about = $3, video_url = $4
		 WHERE organization_id = $1`,
		orgID, title, about, videoURL,
	)
	if err != nil {
		return fmt.Errorf("update referrer profile core: %w", err)
	}
	return checkReferrerRowsAffected(result)
}

// UpdateAvailability writes a single availability_status value.
func (r *ReferrerProfileRepository) UpdateAvailability(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, `
		UPDATE referrer_profiles
		   SET availability_status = $2
		 WHERE organization_id = $1`,
		orgID, string(status),
	)
	if err != nil {
		return fmt.Errorf("update referrer profile availability: %w", err)
	}
	return checkReferrerRowsAffected(result)
}

// UpdateExpertiseDomains rewrites the expertise_domains TEXT[]
// array atomically.
func (r *ReferrerProfileRepository) UpdateExpertiseDomains(ctx context.Context, orgID uuid.UUID, domains []string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if domains == nil {
		domains = []string{}
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE referrer_profiles
		   SET expertise_domains = $2
		 WHERE organization_id = $1`,
		orgID, pq.Array(domains),
	)
	if err != nil {
		return fmt.Errorf("update referrer profile expertise domains: %w", err)
	}
	return checkReferrerRowsAffected(result)
}

// queryByOrgID is the strict read path — returns
// referrerprofile.ErrProfileNotFound when the row is missing.
// Extracted from GetOrCreateByOrgID so the auto-create fallback
// reuses the same query without paying a second SELECT.
func (r *ReferrerProfileRepository) queryByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.ReferrerProfileView, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT ` + referrerProfileSelectColumns + `
		  FROM referrer_profiles rp
		  JOIN organizations     o ON o.id = rp.organization_id
		 WHERE rp.organization_id = $1`

	view, err := scanReferrerProfileRow(r.db.QueryRowContext(ctx, query, orgID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, referrerprofile.ErrProfileNotFound
		}
		return nil, fmt.Errorf("get referrer profile by org id: %w", err)
	}
	return view, nil
}

// insertDefault inserts a fresh referrer profile row with domain
// defaults. Used by the lazy GetOrCreate path.
func (r *ReferrerProfileRepository) insertDefault(ctx context.Context, orgID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	p := referrerprofile.New(orgID)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO referrer_profiles (
			id, organization_id, title, about, video_url,
			availability_status, expertise_domains, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (organization_id) DO NOTHING`,
		p.ID, p.OrganizationID, p.Title, p.About, p.VideoURL,
		string(p.AvailabilityStatus), pq.Array(p.ExpertiseDomains),
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert default referrer profile: %w", err)
	}
	return nil
}

// scanReferrerProfileRow decodes one JOINed SQL row into a
// ReferrerProfileView. Kept private so the Scan order stays in
// sync with referrerProfileSelectColumns.
func scanReferrerProfileRow(row *sql.Row) (*repository.ReferrerProfileView, error) {
	var (
		p            referrerprofile.Profile
		availability string
		domains      []string
		shared       repository.OrganizationSharedProfile
		lat, lng     sql.NullFloat64
		travelRadius sql.NullInt64
		workMode     []string
		langPro      []string
		langConv     []string
	)
	err := row.Scan(
		&p.ID, &p.OrganizationID, &p.Title, &p.About, &p.VideoURL,
		&availability, pq.Array(&domains),
		&p.CreatedAt, &p.UpdatedAt,
		&shared.PhotoURL, &shared.City, &shared.CountryCode,
		&lat, &lng, pq.Array(&workMode), &travelRadius,
		pq.Array(&langPro), pq.Array(&langConv),
	)
	if err != nil {
		return nil, err
	}
	p.AvailabilityStatus = profile.AvailabilityStatus(availability)
	p.ExpertiseDomains = nilToEmpty(domains)
	hydrateSharedProfile(&shared, lat, lng, travelRadius, workMode, langPro, langConv)
	return &repository.ReferrerProfileView{
		Profile: &p,
		Shared:  shared,
	}, nil
}

// checkReferrerRowsAffected turns a sql.Result into either nil
// or referrerprofile.ErrProfileNotFound.
func checkReferrerRowsAffected(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return referrerprofile.ErrProfileNotFound
	}
	return nil
}
