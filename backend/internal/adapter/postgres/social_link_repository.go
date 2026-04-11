package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

// SocialLinkRepository implements repository.SocialLinkRepository with PostgreSQL.
type SocialLinkRepository struct {
	db *sql.DB
}

// NewSocialLinkRepository creates a new SocialLinkRepository backed by the given DB.
func NewSocialLinkRepository(db *sql.DB) *SocialLinkRepository {
	return &SocialLinkRepository{db: db}
}

// ListByOrganization returns all social links for a given organization.
func (r *SocialLinkRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*profile.SocialLink, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, organization_id, platform, url, created_at, updated_at
		FROM social_links
		WHERE organization_id = $1
		ORDER BY platform ASC`

	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("list social links: %w", err)
	}
	defer rows.Close()

	var links []*profile.SocialLink
	for rows.Next() {
		link := &profile.SocialLink{}
		if err := rows.Scan(
			&link.ID, &link.OrganizationID, &link.Platform,
			&link.URL, &link.CreatedAt, &link.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan social link: %w", err)
		}
		links = append(links, link)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("social links rows: %w", err)
	}

	if links == nil {
		links = []*profile.SocialLink{}
	}
	return links, nil
}

// Upsert inserts a new social link or updates the URL if one already exists
// for the same (organization_id, platform) pair.
func (r *SocialLinkRepository) Upsert(ctx context.Context, link *profile.SocialLink) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		INSERT INTO social_links (organization_id, platform, url)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id, platform)
		DO UPDATE SET url = $3, updated_at = now()`

	_, err := r.db.ExecContext(ctx, query, link.OrganizationID, link.Platform, link.URL)
	if err != nil {
		return fmt.Errorf("upsert social link: %w", err)
	}
	return nil
}

// Delete removes a social link for a given org and platform.
func (r *SocialLinkRepository) Delete(ctx context.Context, orgID uuid.UUID, platform string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `DELETE FROM social_links WHERE organization_id = $1 AND platform = $2`
	_, err := r.db.ExecContext(ctx, query, orgID, platform)
	if err != nil {
		return fmt.Errorf("delete social link: %w", err)
	}
	return nil
}
