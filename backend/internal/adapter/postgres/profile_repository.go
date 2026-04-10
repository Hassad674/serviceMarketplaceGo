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
		INSERT INTO profiles (user_id, title, about, photo_url, presentation_video_url, referrer_about, referrer_video_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query,
		p.UserID, p.Title, p.About, p.PhotoURL,
		p.PresentationVideoURL, p.ReferrerAbout, p.ReferrerVideoURL,
		p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create profile: %w", err)
	}

	return nil
}

func (r *ProfileRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*profile.Profile, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	p, err := r.queryByUserID(ctx, userID)
	if err == nil {
		return p, nil
	}

	if !errors.Is(err, profile.ErrProfileNotFound) {
		return nil, err
	}

	return r.ensureProfile(ctx, userID)
}

func (r *ProfileRepository) Update(ctx context.Context, p *profile.Profile) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		UPDATE profiles
		SET title = $2, about = $3, photo_url = $4, presentation_video_url = $5, referrer_about = $6, referrer_video_url = $7
		WHERE user_id = $1`

	result, err := r.db.ExecContext(ctx, query,
		p.UserID, p.Title, p.About, p.PhotoURL,
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

func (r *ProfileRepository) queryByUserID(ctx context.Context, userID uuid.UUID) (*profile.Profile, error) {
	query := `
		SELECT user_id, title, about, photo_url, presentation_video_url, referrer_about, referrer_video_url, created_at, updated_at
		FROM profiles WHERE user_id = $1`

	p := &profile.Profile{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&p.UserID, &p.Title, &p.About, &p.PhotoURL,
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

func (r *ProfileRepository) SearchPublic(ctx context.Context, roleFilter string, referrerOnly bool, cursorStr string, limit int) ([]*profile.PublicProfile, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		query := `
			SELECT u.id, u.display_name, u.first_name, u.last_name, u.role, u.referrer_enabled,
			       COALESCE(p.title, ''), COALESCE(p.photo_url, ''), u.created_at,
			       COALESCE(r.avg_rating, 0)::float8, COALESCE(r.review_count, 0)::int
			FROM users u
			LEFT JOIN profiles p ON p.user_id = u.id
			LEFT JOIN (
				SELECT reviewed_id,
				       AVG(global_rating)::float8 AS avg_rating,
				       COUNT(*)::int              AS review_count
				FROM reviews
				WHERE moderation_status != 'hidden'
				GROUP BY reviewed_id
			) r ON r.reviewed_id = u.id
			WHERE ($1 = '' OR u.role = $1)
			AND ($2 = false OR (u.role = 'provider' AND u.referrer_enabled = true))
			ORDER BY u.created_at DESC, u.id DESC
			LIMIT $3`
		rows, err = r.db.QueryContext(ctx, query, roleFilter, referrerOnly, limit+1)
	} else {
		c, decErr := cursor.Decode(cursorStr)
		if decErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", decErr)
		}
		query := `
			SELECT u.id, u.display_name, u.first_name, u.last_name, u.role, u.referrer_enabled,
			       COALESCE(p.title, ''), COALESCE(p.photo_url, ''), u.created_at,
			       COALESCE(r.avg_rating, 0)::float8, COALESCE(r.review_count, 0)::int
			FROM users u
			LEFT JOIN profiles p ON p.user_id = u.id
			LEFT JOIN (
				SELECT reviewed_id,
				       AVG(global_rating)::float8 AS avg_rating,
				       COUNT(*)::int              AS review_count
				FROM reviews
				WHERE moderation_status != 'hidden'
				GROUP BY reviewed_id
			) r ON r.reviewed_id = u.id
			WHERE ($1 = '' OR u.role = $1)
			AND ($2 = false OR (u.role = 'provider' AND u.referrer_enabled = true))
			AND (u.created_at, u.id) < ($3, $4)
			ORDER BY u.created_at DESC, u.id DESC
			LIMIT $5`
		rows, err = r.db.QueryContext(ctx, query, roleFilter, referrerOnly, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to search profiles: %w", err)
	}
	defer rows.Close()

	var results []*profile.PublicProfile
	for rows.Next() {
		pp := &profile.PublicProfile{}
		if err := rows.Scan(
			&pp.UserID, &pp.DisplayName, &pp.FirstName, &pp.LastName,
			&pp.Role, &pp.ReferrerEnabled, &pp.Title, &pp.PhotoURL,
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
		nextCursor = cursor.Encode(last.CreatedAt, last.UserID)
		results = results[:limit]
	}

	if results == nil {
		results = []*profile.PublicProfile{}
	}

	return results, nextCursor, nil
}

func (r *ProfileRepository) GetPublicProfilesByUserIDs(ctx context.Context, userIDs []uuid.UUID) ([]*profile.PublicProfile, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if len(userIDs) == 0 {
		return []*profile.PublicProfile{}, nil
	}

	query := `
		SELECT u.id, u.display_name, u.first_name, u.last_name, u.role, u.referrer_enabled,
		       COALESCE(p.title, ''), COALESCE(p.photo_url, ''),
		       COALESCE(r.avg_rating, 0)::float8, COALESCE(r.review_count, 0)::int
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
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
		return nil, fmt.Errorf("get public profiles by user ids: %w", err)
	}
	defer rows.Close()

	var results []*profile.PublicProfile
	for rows.Next() {
		pp := &profile.PublicProfile{}
		if err := rows.Scan(
			&pp.UserID, &pp.DisplayName, &pp.FirstName, &pp.LastName,
			&pp.Role, &pp.ReferrerEnabled, &pp.Title, &pp.PhotoURL,
			&pp.AverageRating, &pp.ReviewCount,
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

func (r *ProfileRepository) ensureProfile(ctx context.Context, userID uuid.UUID) (*profile.Profile, error) {
	newProfile := profile.NewProfile(userID)
	if err := r.Create(ctx, newProfile); err != nil {
		return nil, fmt.Errorf("failed to auto-create profile: %w", err)
	}

	return r.queryByUserID(ctx, userID)
}
