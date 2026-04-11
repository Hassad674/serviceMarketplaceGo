package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/pkg/cursor"
)

type ProfileRepository struct {
	db *sql.DB
}

func NewProfileRepository(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (r *ProfileRepository) Create(ctx context.Context, p *profile.Profile) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		INSERT INTO profiles (organization_id, title, about, photo_url, presentation_video_url, referrer_about, referrer_video_url, created_at, updated_at)
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

func (r *ProfileRepository) Update(ctx context.Context, p *profile.Profile) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		UPDATE profiles
		SET title = $2, about = $3, photo_url = $4, presentation_video_url = $5, referrer_about = $6, referrer_video_url = $7
		WHERE organization_id = $1`

	result, err := r.db.ExecContext(ctx, query,
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

func (r *ProfileRepository) queryByOrgID(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error) {
	query := `
		SELECT organization_id, title, about, photo_url, presentation_video_url, referrer_about, referrer_video_url, created_at, updated_at
		FROM profiles WHERE organization_id = $1`

	p := &profile.Profile{}
	err := r.db.QueryRowContext(ctx, query, orgID).Scan(
		&p.OrganizationID, &p.Title, &p.About, &p.PhotoURL,
		&p.PresentationVideoURL, &p.ReferrerAbout, &p.ReferrerVideoURL,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, profile.ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	return p, nil
}

// SearchPublic returns orgs filtered by type and referrer flag, paginated.
// Review aggregation still happens on the owner user row — in phase R3
// the reviews table gets its own organization_id and this will flip
// to joining on reviews.reviewed_organization_id. Until then the query
// preserves the same aggregate because every agency/enterprise/provider_personal
// has a single owner.
func (r *ProfileRepository) SearchPublic(ctx context.Context, orgTypeFilter string, referrerOnly bool, cursorStr string, limit int) ([]*profile.PublicProfile, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	base := `
		SELECT o.id, o.name, o.type,
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
			FROM reviews
			WHERE moderation_status != 'hidden'
			GROUP BY reviewed_id
		) r ON r.reviewed_id = o.owner_user_id
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
		if err := rows.Scan(
			&pp.OrganizationID, &pp.Name, &pp.OrgType,
			&pp.Title, &pp.PhotoURL, &pp.ReferrerEnabled,
			&pp.CreatedAt, &pp.AverageRating, &pp.ReviewCount,
		); err != nil {
			return nil, "", fmt.Errorf("failed to scan public profile: %w", err)
		}
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
		SELECT o.id, o.name, o.type,
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
			FROM reviews
			WHERE moderation_status != 'hidden'
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
			&pp.OrganizationID, &pp.Name, &pp.OrgType,
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
		       o.id, o.name, o.type,
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
			FROM reviews
			WHERE moderation_status != 'hidden'
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
			&pp.OrganizationID, &pp.Name, &pp.OrgType,
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
