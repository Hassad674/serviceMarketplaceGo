package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
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
		INSERT INTO profiles (user_id, title, photo_url, presentation_video_url, referrer_video_url, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO NOTHING`

	_, err := r.db.ExecContext(ctx, query,
		p.UserID, p.Title, p.PhotoURL,
		p.PresentationVideoURL, p.ReferrerVideoURL,
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
		SET title = $2, photo_url = $3, presentation_video_url = $4, referrer_video_url = $5
		WHERE user_id = $1`

	result, err := r.db.ExecContext(ctx, query,
		p.UserID, p.Title, p.PhotoURL,
		p.PresentationVideoURL, p.ReferrerVideoURL,
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
		SELECT user_id, title, photo_url, presentation_video_url, referrer_video_url, created_at, updated_at
		FROM profiles WHERE user_id = $1`

	p := &profile.Profile{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&p.UserID, &p.Title, &p.PhotoURL,
		&p.PresentationVideoURL, &p.ReferrerVideoURL,
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

func (r *ProfileRepository) ensureProfile(ctx context.Context, userID uuid.UUID) (*profile.Profile, error) {
	newProfile := profile.NewProfile(userID)
	if err := r.Create(ctx, newProfile); err != nil {
		return nil, fmt.Errorf("failed to auto-create profile: %w", err)
	}

	return r.queryByUserID(ctx, userID)
}
