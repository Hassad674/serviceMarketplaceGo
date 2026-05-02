package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/profile"
)

// SocialLinkRepository implements repository.SocialLinkRepository with PostgreSQL.
// Every query is scoped by (organization_id, persona) so that the
// freelance, referrer and agency sets never bleed into each other.
type SocialLinkRepository struct {
	db *sql.DB
}

// NewSocialLinkRepository creates a new SocialLinkRepository backed by the given DB.
func NewSocialLinkRepository(db *sql.DB) *SocialLinkRepository {
	return &SocialLinkRepository{db: db}
}

// ListByOrganizationPersona returns all social links for a given
// organization under the specified persona.
func (r *SocialLinkRepository) ListByOrganizationPersona(
	ctx context.Context,
	orgID uuid.UUID,
	persona profile.SocialLinkPersona,
) ([]*profile.SocialLink, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		SELECT id, organization_id, persona, platform, url, created_at, updated_at
		FROM social_links
		WHERE organization_id = $1 AND persona = $2
		ORDER BY platform ASC`

	rows, err := Query(ctx, r.db, query, orgID, string(persona))
	if err != nil {
		return nil, fmt.Errorf("list social links: %w", err)
	}
	defer rows.Close()

	var links []*profile.SocialLink
	for rows.Next() {
		link := &profile.SocialLink{}
		var personaStr string
		if err := rows.Scan(
			&link.ID, &link.OrganizationID, &personaStr, &link.Platform,
			&link.URL, &link.CreatedAt, &link.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan social link: %w", err)
		}
		link.Persona = profile.SocialLinkPersona(personaStr)
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

// Upsert inserts a new social link or updates the URL if one already
// exists for the same (organization_id, persona, platform) triple.
func (r *SocialLinkRepository) Upsert(ctx context.Context, link *profile.SocialLink) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		INSERT INTO social_links (organization_id, persona, platform, url)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (organization_id, persona, platform)
		DO UPDATE SET url = $4, updated_at = now()`

	_, err := Exec(
		ctx, r.db, query,
		link.OrganizationID, string(link.Persona), link.Platform, link.URL,
	)
	if err != nil {
		return fmt.Errorf("upsert social link: %w", err)
	}
	return nil
}

// Delete removes a social link for the given (org, persona, platform) triple.
func (r *SocialLinkRepository) Delete(
	ctx context.Context,
	orgID uuid.UUID,
	persona profile.SocialLinkPersona,
	platform string,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `DELETE FROM social_links WHERE organization_id = $1 AND persona = $2 AND platform = $3`
	_, err := Exec(ctx, r.db, query, orgID, string(persona), platform)
	if err != nil {
		return fmt.Errorf("delete social link: %w", err)
	}
	return nil
}
