package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	domainskill "marketplace-backend/internal/domain/skill"
)

// ProfileSkillRepository is the PostgreSQL-backed implementation of
// repository.ProfileSkillRepository. It owns exactly one table,
// profile_skills (migration 081), and performs all writes inside a
// single transaction so that readers never observe a partial Replace.
//
// The repository is stateless — the only field is the shared *sql.DB
// handle injected from cmd/api/main.go. It may be constructed once and
// shared across all handlers.
type ProfileSkillRepository struct {
	db *sql.DB
}

// NewProfileSkillRepository returns a repository ready to talk to the
// given *sql.DB. Tuning the handle (SetMaxOpenConns, SetMaxIdleConns,
// …) is the caller's responsibility, as everywhere else in this package.
func NewProfileSkillRepository(db *sql.DB) *ProfileSkillRepository {
	return &ProfileSkillRepository{db: db}
}

// ListByOrgID returns every skill attached to the organization, in
// display order (position ASC). An organization with no skills yields
// an empty (non-nil) slice so the caller can marshal it directly to
// the JSON array `[]`.
//
// The SELECT joins skills_catalog to populate display_text so public
// profile DTOs can render the canonical casing without a follow-up
// catalog lookup. A LEFT JOIN is used defensively so a profile_skills
// row referencing a catalog entry deleted out-of-band still renders
// its raw skill_text rather than disappearing from the list.
func (r *ProfileSkillRepository) ListByOrgID(ctx context.Context, orgID uuid.UUID) ([]*domainskill.ProfileSkill, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx,
		`SELECT ps.organization_id, ps.skill_text, COALESCE(sc.display_text, ps.skill_text), ps.position, ps.created_at
		   FROM profile_skills ps
		   LEFT JOIN skills_catalog sc ON sc.skill_text = ps.skill_text
		  WHERE ps.organization_id = $1
		  ORDER BY ps.position ASC`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("list profile skills by org id: %w", err)
	}
	defer rows.Close()

	skills := make([]*domainskill.ProfileSkill, 0, 16)
	for rows.Next() {
		var skill domainskill.ProfileSkill
		if err := rows.Scan(
			&skill.OrganizationID,
			&skill.SkillText,
			&skill.DisplayText,
			&skill.Position,
			&skill.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan profile skill row: %w", err)
		}
		skills = append(skills, &skill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate profile skill rows: %w", err)
	}
	return skills, nil
}

// ListByOrgIDs returns every skill attached to any of the supplied
// organization IDs in a single database roundtrip (N+1 prevention).
// The returned map is keyed by organization ID and contains a
// (non-nil, possibly empty) slice for every ID passed in — callers
// can range over the input directly without nil-checks.
//
// Ordering: the SQL ORDER BY position ASC combined with the stable
// iteration over the returned rows guarantees each per-org slice is
// in display order. Orgs with zero skills still appear in the map
// with an empty slice — the method seeds every input key up front.
func (r *ProfileSkillRepository) ListByOrgIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]*domainskill.ProfileSkill, error) {
	out := make(map[uuid.UUID][]*domainskill.ProfileSkill, len(orgIDs))
	for _, id := range orgIDs {
		out[id] = []*domainskill.ProfileSkill{}
	}
	if len(orgIDs) == 0 {
		return out, nil
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// pq.Array lets us pass the UUID slice to a single ANY($1) query;
	// driver converts each uuid.UUID to its canonical TEXT form.
	idStrings := make([]string, len(orgIDs))
	for i, id := range orgIDs {
		idStrings[i] = id.String()
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT ps.organization_id, ps.skill_text, COALESCE(sc.display_text, ps.skill_text), ps.position, ps.created_at
		   FROM profile_skills ps
		   LEFT JOIN skills_catalog sc ON sc.skill_text = ps.skill_text
		  WHERE ps.organization_id = ANY($1)
		  ORDER BY ps.organization_id, ps.position ASC`,
		pq.Array(idStrings),
	)
	if err != nil {
		return nil, fmt.Errorf("list profile skills by org ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var skill domainskill.ProfileSkill
		if err := rows.Scan(
			&skill.OrganizationID,
			&skill.SkillText,
			&skill.DisplayText,
			&skill.Position,
			&skill.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan profile skill row: %w", err)
		}
		out[skill.OrganizationID] = append(out[skill.OrganizationID], &skill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate profile skill rows: %w", err)
	}
	return out, nil
}

// ReplaceForOrg atomically swaps the organization's declared skills
// with the provided slice. Implementation notes:
//
//   - DELETE + INSERT inside a single transaction. A concurrent reader
//     on READ COMMITTED will either see the old list entirely or the
//     new list entirely — never a half-written state. This mirrors the
//     exact pattern used by ExpertiseRepository.Replace (migration 080).
//   - An empty input slice is valid: the DELETE runs, no INSERT is
//     issued, and the transaction commits — effectively clearing the
//     organization's skill list.
//   - The caller is responsible for populating contiguous, 0-indexed
//     Position values on each input ProfileSkill; this method writes
//     the positions verbatim and does not renumber them.
//   - defer tx.Rollback() is safe after a successful Commit: it becomes
//     a no-op, so every error path is covered without explicit rollback
//     calls.
func (r *ProfileSkillRepository) ReplaceForOrg(ctx context.Context, orgID uuid.UUID, skills []*domainskill.ProfileSkill) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin profile skills replace tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM profile_skills WHERE organization_id = $1`,
		orgID,
	); err != nil {
		return fmt.Errorf("delete existing profile skills: %w", err)
	}

	if len(skills) > 0 {
		if err := insertProfileSkillRows(ctx, tx, orgID, skills); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit profile skills replace: %w", err)
	}
	return nil
}

// CountByOrg returns how many skills are currently attached to the
// organization. Used by the service layer to enforce per-org-type
// limits (MaxSkillsForOrgType) without fetching the whole list.
func (r *ProfileSkillRepository) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM profile_skills WHERE organization_id = $1`,
		orgID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count profile skills by org: %w", err)
	}
	return count, nil
}

// DeleteAllByOrg removes every skill attached to the organization.
// Used on an explicit user-initiated "reset my skills" action; the
// cascade-on-org-delete path is handled by the FK in the migration,
// so this method is for application-level wipes only.
func (r *ProfileSkillRepository) DeleteAllByOrg(ctx context.Context, orgID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM profile_skills WHERE organization_id = $1`,
		orgID,
	); err != nil {
		return fmt.Errorf("delete all profile skills by org: %w", err)
	}
	return nil
}

// insertProfileSkillRows issues a single multi-row INSERT for the
// provided skills. Extracted into a helper so ReplaceForOrg stays
// well under the 50-line / 3-nesting cap and reads as a pipeline.
//
// The SQL uses parameterized placeholders exclusively: orgID is $1,
// and each skill contributes two params (skill_text and position).
// No string concatenation touches user input.
func insertProfileSkillRows(ctx context.Context, tx *sql.Tx, orgID uuid.UUID, skills []*domainskill.ProfileSkill) error {
	args := make([]any, 0, 1+len(skills)*2)
	args = append(args, orgID)

	valueClauses := make([]string, 0, len(skills))
	for i, s := range skills {
		// orgID stays at $1; each row consumes two new placeholders.
		textPlaceholder := fmt.Sprintf("$%d", 2+i*2)
		posPlaceholder := fmt.Sprintf("$%d", 3+i*2)
		valueClauses = append(valueClauses, fmt.Sprintf("($1, %s, %s, now())", textPlaceholder, posPlaceholder))
		args = append(args, s.SkillText, s.Position)
	}

	query := "INSERT INTO profile_skills (organization_id, skill_text, position, created_at) VALUES "
	for i, clause := range valueClauses {
		if i > 0 {
			query += ", "
		}
		// gosec G202 suppression rationale: same as
		// expertise_repository.insertExpertiseRows. `clause` only contains
		// numeric placeholders ($1, $2, …) generated from a counter; the
		// skill_text and position user-controlled values flow through
		// `args` and reach the DB via $N binding, never the SQL text.
		// Injection-payload coverage lives in the sql_injection_test.go.
		query += clause // #nosec G202 -- placeholder-only concat, tested
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("insert profile skill rows: %w", err)
	}
	return nil
}
