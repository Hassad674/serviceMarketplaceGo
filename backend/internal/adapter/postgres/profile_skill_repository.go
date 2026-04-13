package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	domainskill "marketplace-backend/internal/domain/skill"
)

// ProfileSkillRepository is the PostgreSQL-backed implementation of
// repository.ProfileSkillRepository. It owns the profile_skills table
// (migration 081) and performs the full-list atomic ReplaceForOrg
// swap inside a single transaction so concurrent readers never see a
// partially-written state.
type ProfileSkillRepository struct {
	db *sql.DB
}

// NewProfileSkillRepository returns a profile skills repository bound
// to the given *sql.DB.
func NewProfileSkillRepository(db *sql.DB) *ProfileSkillRepository {
	return &ProfileSkillRepository{db: db}
}

// ListByOrgID returns all skills attached to the organization, sorted
// by position ASC. Always returns a non-nil slice.
func (r *ProfileSkillRepository) ListByOrgID(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*domainskill.ProfileSkill, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx,
		`SELECT organization_id, skill_text, position, created_at
		   FROM profile_skills
		  WHERE organization_id = $1
		  ORDER BY position ASC`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list profile skills: %w", err)
	}
	defer rows.Close()

	out := make([]*domainskill.ProfileSkill, 0, 8)
	for rows.Next() {
		var ps domainskill.ProfileSkill
		if err := rows.Scan(&ps.OrganizationID, &ps.SkillText, &ps.Position, &ps.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan profile skill row: %w", err)
		}
		out = append(out, &ps)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate profile skill rows: %w", err)
	}
	return out, nil
}

// ReplaceForOrg atomically swaps the organization's profile skills.
// DELETE + INSERT inside a single transaction. Also maintains the
// usage_count denormalization on skills_catalog by decrementing every
// removed skill and incrementing every added skill, in the same
// transaction so the two values never drift.
func (r *ProfileSkillRepository) ReplaceForOrg(
	ctx context.Context,
	orgID uuid.UUID,
	skills []*domainskill.ProfileSkill,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin profile skills replace tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Capture the current set so we can compute the symmetric diff
	// against the incoming list — only truly added / removed rows
	// bump usage_count, skills that stayed put don't.
	existing, err := listSkillTextsForOrg(ctx, tx, orgID)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM profile_skills WHERE organization_id = $1`,
		orgID,
	); err != nil {
		return fmt.Errorf("delete existing profile skills: %w", err)
	}

	if len(skills) > 0 {
		if err := insertProfileSkillRows(ctx, tx, skills); err != nil {
			return err
		}
	}

	if err := applyUsageCountDiff(ctx, tx, existing, skills); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit profile skills replace: %w", err)
	}
	return nil
}

// CountByOrg returns the number of skills currently attached to the
// given organization.
func (r *ProfileSkillRepository) CountByOrg(
	ctx context.Context,
	orgID uuid.UUID,
) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var n int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM profile_skills WHERE organization_id = $1`,
		orgID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count profile skills: %w", err)
	}
	return n, nil
}

// DeleteAllByOrg removes every skill attached to the organization and
// decrements the catalog usage counters accordingly.
func (r *ProfileSkillRepository) DeleteAllByOrg(
	ctx context.Context,
	orgID uuid.UUID,
) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin profile skills delete tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	existing, err := listSkillTextsForOrg(ctx, tx, orgID)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM profile_skills WHERE organization_id = $1`,
		orgID,
	); err != nil {
		return fmt.Errorf("delete profile skills: %w", err)
	}

	for _, text := range existing {
		if _, err := tx.ExecContext(ctx,
			`UPDATE skills_catalog
			    SET usage_count = GREATEST(usage_count - 1, 0)
			  WHERE skill_text = $1`,
			text,
		); err != nil {
			return fmt.Errorf("decrement skill usage: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit profile skills delete: %w", err)
	}
	return nil
}

// listSkillTextsForOrg returns the current set of skill_text values
// attached to the organization. Used inside ReplaceForOrg and
// DeleteAllByOrg to feed the usage_count diff.
func listSkillTextsForOrg(
	ctx context.Context,
	tx *sql.Tx,
	orgID uuid.UUID,
) ([]string, error) {
	rows, err := tx.QueryContext(ctx,
		`SELECT skill_text FROM profile_skills WHERE organization_id = $1`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list existing profile skills: %w", err)
	}
	defer rows.Close()

	out := make([]string, 0, 8)
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return nil, fmt.Errorf("scan profile skill text: %w", err)
		}
		out = append(out, text)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate profile skill texts: %w", err)
	}
	return out, nil
}

// insertProfileSkillRows inserts the ordered slice of skills via
// per-row ExecContext calls. A single multi-row INSERT would be
// marginally faster, but the typical batch size here is < 40 and the
// per-row form keeps the code legible.
func insertProfileSkillRows(
	ctx context.Context,
	tx *sql.Tx,
	skills []*domainskill.ProfileSkill,
) error {
	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO profile_skills (organization_id, skill_text, position)
		 VALUES ($1, $2, $3)`,
	)
	if err != nil {
		return fmt.Errorf("prepare profile skill insert: %w", err)
	}
	defer stmt.Close()

	for _, ps := range skills {
		if _, err := stmt.ExecContext(ctx, ps.OrganizationID, ps.SkillText, ps.Position); err != nil {
			return fmt.Errorf("insert profile skill: %w", err)
		}
	}
	return nil
}

// applyUsageCountDiff bumps usage_count for skills that appear in the
// new list but not the old, and decrements it for skills that appear
// in the old list but not the new. Skills present in both are left
// untouched.
func applyUsageCountDiff(
	ctx context.Context,
	tx *sql.Tx,
	oldTexts []string,
	newSkills []*domainskill.ProfileSkill,
) error {
	oldSet := make(map[string]struct{}, len(oldTexts))
	for _, t := range oldTexts {
		oldSet[t] = struct{}{}
	}
	newSet := make(map[string]struct{}, len(newSkills))
	for _, ps := range newSkills {
		newSet[ps.SkillText] = struct{}{}
	}

	// Decrement removed (in old, not in new).
	for text := range oldSet {
		if _, stillThere := newSet[text]; stillThere {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE skills_catalog
			    SET usage_count = GREATEST(usage_count - 1, 0)
			  WHERE skill_text = $1`,
			text,
		); err != nil {
			return fmt.Errorf("decrement skill usage: %w", err)
		}
	}
	// Increment added (in new, not in old).
	for text := range newSet {
		if _, wasThere := oldSet[text]; wasThere {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`UPDATE skills_catalog
			    SET usage_count = usage_count + 1
			  WHERE skill_text = $1`,
			text,
		); err != nil {
			return fmt.Errorf("increment skill usage: %w", err)
		}
	}
	return nil
}
