package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/port/repository"
)

// organization_shared_profile.go adds the shared-profile write path
// (migration 096) to the existing OrganizationRepository. Lives in
// its own file rather than on organization_repository.go so the
// legacy team/Stripe/KYC concerns of that file do not pick up a
// dependency on the shared-profile columns.
//
// The methods defined here together satisfy the
// repository.OrganizationSharedProfileWriter port — the main.go
// wiring can pass a single *OrganizationRepository value under two
// interface handles (OrganizationRepository and
// OrganizationSharedProfileWriter) without a bridging adapter.

// UpdateSharedLocation rewrites the entire location block on the
// organizations row in a single UPDATE. Every column is always
// written — nil pointers clear the column to NULL at the DB level,
// preserving the "atomic block" semantics the legacy profile
// location save uses.
func (r *OrganizationRepository) UpdateSharedLocation(ctx context.Context, orgID uuid.UUID, input repository.SharedProfileLocationInput) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	workMode := input.WorkMode
	if workMode == nil {
		workMode = []string{}
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE organizations
		   SET city             = $2,
		       country_code     = $3,
		       latitude         = $4,
		       longitude        = $5,
		       work_mode        = $6,
		       travel_radius_km = $7,
		       updated_at       = now()
		 WHERE id = $1`,
		orgID,
		input.City,
		input.CountryCode,
		nullFloat(input.Latitude),
		nullFloat(input.Longitude),
		pq.Array(workMode),
		nullInt(input.TravelRadiusKm),
	)
	if err != nil {
		return fmt.Errorf("update organization shared location: %w", err)
	}
	return checkOrgRowsAffected(result)
}

// UpdateSharedLanguages replaces the two language arrays atomically.
func (r *OrganizationRepository) UpdateSharedLanguages(ctx context.Context, orgID uuid.UUID, professional, conversational []string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if professional == nil {
		professional = []string{}
	}
	if conversational == nil {
		conversational = []string{}
	}

	result, err := r.db.ExecContext(ctx, `
		UPDATE organizations
		   SET languages_professional   = $2,
		       languages_conversational = $3,
		       updated_at               = now()
		 WHERE id = $1`,
		orgID,
		pq.Array(professional),
		pq.Array(conversational),
	)
	if err != nil {
		return fmt.Errorf("update organization shared languages: %w", err)
	}
	return checkOrgRowsAffected(result)
}

// UpdateSharedPhotoURL writes a single photo_url value. Empty
// string clears the photo — the caller is responsible for the
// upstream storage delete (if any).
func (r *OrganizationRepository) UpdateSharedPhotoURL(ctx context.Context, orgID uuid.UUID, photoURL string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, `
		UPDATE organizations
		   SET photo_url  = $2,
		       updated_at = now()
		 WHERE id = $1`,
		orgID, photoURL,
	)
	if err != nil {
		return fmt.Errorf("update organization shared photo url: %w", err)
	}
	return checkOrgRowsAffected(result)
}

// GetSharedProfile returns the shared-profile block for an org.
// Used by read paths that need the shared fields on their own
// (without joining a persona).
func (r *OrganizationRepository) GetSharedProfile(ctx context.Context, orgID uuid.UUID) (*repository.OrganizationSharedProfile, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var (
		shared       repository.OrganizationSharedProfile
		lat, lng     sql.NullFloat64
		travelRadius sql.NullInt64
		workMode     []string
		langPro      []string
		langConv     []string
	)
	err := r.db.QueryRowContext(ctx, `
		SELECT photo_url, city, country_code, latitude, longitude,
		       work_mode, travel_radius_km,
		       languages_professional, languages_conversational
		  FROM organizations
		 WHERE id = $1`,
		orgID,
	).Scan(
		&shared.PhotoURL, &shared.City, &shared.CountryCode,
		&lat, &lng,
		pq.Array(&workMode), &travelRadius,
		pq.Array(&langPro), pq.Array(&langConv),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, organization.ErrOrgNotFound
		}
		return nil, fmt.Errorf("get organization shared profile: %w", err)
	}
	hydrateSharedProfile(&shared, lat, lng, travelRadius, workMode, langPro, langConv)
	return &shared, nil
}

// checkOrgRowsAffected turns a sql.Result into either nil or
// organization.ErrOrgNotFound.
func checkOrgRowsAffected(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return organization.ErrOrgNotFound
	}
	return nil
}
