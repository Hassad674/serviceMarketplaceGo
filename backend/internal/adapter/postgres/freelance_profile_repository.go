package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/port/repository"
)

// FreelanceProfileRepository is the PostgreSQL-backed implementation
// of repository.FreelanceProfileRepository. It owns the
// freelance_profiles table (migration 097) and reads the shared
// profile block from organizations via a single JOIN so every
// GetByOrgID returns a fully-hydrated view in one roundtrip.
//
// The repository is stateless — only the shared *sql.DB handle is
// stored. Safe to construct once in cmd/api/main.go and share
// across handlers.
type FreelanceProfileRepository struct {
	db *sql.DB
}

// NewFreelanceProfileRepository returns a repository ready to talk
// to the given *sql.DB.
func NewFreelanceProfileRepository(db *sql.DB) *FreelanceProfileRepository {
	return &FreelanceProfileRepository{db: db}
}

// freelanceProfileSelectColumns enumerates the columns the adapter
// reads when hydrating a freelance profile + joined shared block.
// Centralised in a const so the JOIN column list and the Scan
// target stay in sync — add a column here AND in scanFreelance
// when extending the shape.
const freelanceProfileSelectColumns = `
	fp.id, fp.organization_id, fp.title, fp.about, fp.video_url,
	fp.availability_status, fp.expertise_domains, fp.created_at, fp.updated_at,
	o.photo_url, o.city, o.country_code, o.latitude, o.longitude,
	o.work_mode, o.travel_radius_km,
	o.languages_professional, o.languages_conversational`

// GetByOrgID fetches the freelance profile for the given org JOINed
// with the organization's shared-profile block. One round-trip —
// no N+1 between freelance_profiles and organizations. Strict read:
// surfaces freelanceprofile.ErrProfileNotFound when the row is
// missing. Callers that want lazy creation use GetOrCreateByOrgID.
func (r *FreelanceProfileRepository) GetByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	return r.queryByOrgID(ctx, orgID)
}

// GetOrCreateByOrgID fetches the freelance profile for the given
// org and, if no row exists, inserts a fresh default row before
// returning it. Used by the owner-side GetByOrgID path so a newly
// registered provider_personal user (whose account predates the
// split migration) transparently gets a row on first visit instead
// of hitting a 404. Never returns ErrProfileNotFound.
func (r *FreelanceProfileRepository) GetOrCreateByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	view, err := r.queryByOrgID(ctx, orgID)
	if err == nil {
		return view, nil
	}
	if !errors.Is(err, freelanceprofile.ErrProfileNotFound) {
		return nil, err
	}
	if err := r.insertDefault(ctx, orgID); err != nil {
		return nil, err
	}
	return r.queryByOrgID(ctx, orgID)
}

// queryByOrgID is the strict read path shared between GetByOrgID
// and the lazy GetOrCreateByOrgID fallback. Extracted so both
// call sites scan the same JOINed row shape without drifting.
func (r *FreelanceProfileRepository) queryByOrgID(ctx context.Context, orgID uuid.UUID) (*repository.FreelanceProfileView, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT ` + freelanceProfileSelectColumns + `
		  FROM freelance_profiles fp
		  JOIN organizations      o ON o.id = fp.organization_id
		 WHERE fp.organization_id = $1`

	view, err := scanFreelanceProfileRow(r.db.QueryRowContext(ctx, query, orgID))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, freelanceprofile.ErrProfileNotFound
		}
		return nil, fmt.Errorf("get freelance profile by org id: %w", err)
	}
	return view, nil
}

// insertDefault inserts a fresh freelance profile row with domain
// defaults. Used by the lazy GetOrCreate path. The ON CONFLICT
// clause makes the insert idempotent so concurrent first-access
// requests don't fight.
func (r *FreelanceProfileRepository) insertDefault(ctx context.Context, orgID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	p := freelanceprofile.New(orgID)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO freelance_profiles (
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
		return fmt.Errorf("insert default freelance profile: %w", err)
	}
	return nil
}

// UpdateCore writes the title / about / video_url triplet.
func (r *FreelanceProfileRepository) UpdateCore(ctx context.Context, orgID uuid.UUID, title, about, videoURL string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, `
		UPDATE freelance_profiles
		   SET title = $2, about = $3, video_url = $4
		 WHERE organization_id = $1`,
		orgID, title, about, videoURL,
	)
	if err != nil {
		return fmt.Errorf("update freelance profile core: %w", err)
	}
	return checkFreelanceRowsAffected(result)
}

// UpdateAvailability writes a single availability_status value.
func (r *FreelanceProfileRepository) UpdateAvailability(ctx context.Context, orgID uuid.UUID, status profile.AvailabilityStatus) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, `
		UPDATE freelance_profiles
		   SET availability_status = $2
		 WHERE organization_id = $1`,
		orgID, string(status),
	)
	if err != nil {
		return fmt.Errorf("update freelance profile availability: %w", err)
	}
	return checkFreelanceRowsAffected(result)
}

// UpdateVideo writes the video_url slot in isolation so the upload
// handler never races with an in-flight core text edit. Empty string
// clears the column (DELETE path).
func (r *FreelanceProfileRepository) UpdateVideo(ctx context.Context, orgID uuid.UUID, videoURL string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, `
		UPDATE freelance_profiles
		   SET video_url = $2
		 WHERE organization_id = $1`,
		orgID, videoURL,
	)
	if err != nil {
		return fmt.Errorf("update freelance profile video: %w", err)
	}
	return checkFreelanceRowsAffected(result)
}

// GetVideoURL returns the currently stored video_url for the org so
// the upload handler can delete the prior MinIO object before
// overwriting. Returns ErrProfileNotFound when no row exists.
func (r *FreelanceProfileRepository) GetVideoURL(ctx context.Context, orgID uuid.UUID) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var videoURL string
	err := r.db.QueryRowContext(ctx, `
		SELECT video_url
		  FROM freelance_profiles
		 WHERE organization_id = $1`, orgID).Scan(&videoURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", freelanceprofile.ErrProfileNotFound
		}
		return "", fmt.Errorf("get freelance profile video url: %w", err)
	}
	return videoURL, nil
}

// UpdateExpertiseDomains rewrites the expertise_domains TEXT[] array
// atomically. A nil slice is coerced to an empty array so the NOT
// NULL constraint is honored.
func (r *FreelanceProfileRepository) UpdateExpertiseDomains(ctx context.Context, orgID uuid.UUID, domains []string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if domains == nil {
		domains = []string{}
	}
	result, err := r.db.ExecContext(ctx, `
		UPDATE freelance_profiles
		   SET expertise_domains = $2
		 WHERE organization_id = $1`,
		orgID, pq.Array(domains),
	)
	if err != nil {
		return fmt.Errorf("update freelance profile expertise domains: %w", err)
	}
	return checkFreelanceRowsAffected(result)
}

// scanFreelanceProfileRow decodes one JOINed SQL row into a
// FreelanceProfileView. Kept private to this file so the Scan
// order stays exactly in sync with freelanceProfileSelectColumns.
func scanFreelanceProfileRow(row *sql.Row) (*repository.FreelanceProfileView, error) {
	var (
		p            freelanceprofile.Profile
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
	return &repository.FreelanceProfileView{
		Profile: &p,
		Shared:  shared,
	}, nil
}

// checkFreelanceRowsAffected turns a sql.Result into either nil
// (one row affected — success) or freelanceprofile.ErrProfileNotFound
// (zero rows — the org does not have a freelance row).
func checkFreelanceRowsAffected(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return freelanceprofile.ErrProfileNotFound
	}
	return nil
}

// hydrateSharedProfile copies the raw SQL nullables + TEXT[] arrays
// onto the shared struct, translating sql.NullXXX to *T and ensuring
// slices are non-nil. Extracted so the per-persona scan helpers
// (freelance + referrer) stay small and focused.
func hydrateSharedProfile(
	shared *repository.OrganizationSharedProfile,
	lat, lng sql.NullFloat64,
	travelRadius sql.NullInt64,
	workMode, langPro, langConv []string,
) {
	if lat.Valid {
		v := lat.Float64
		shared.Latitude = &v
	}
	if lng.Valid {
		v := lng.Float64
		shared.Longitude = &v
	}
	if travelRadius.Valid {
		v := int(travelRadius.Int64)
		shared.TravelRadiusKm = &v
	}
	shared.WorkMode = nilToEmpty(workMode)
	shared.LanguagesProfessional = nilToEmpty(langPro)
	shared.LanguagesConversational = nilToEmpty(langConv)
}
