package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/profilepricing"
)

// ProfilePricingRepository is the PostgreSQL-backed implementation
// of repository.ProfilePricingRepository. It owns exactly one table
// (profile_pricing, migration 083) with a composite primary key
// capping cardinality at 2 rows per organization.
//
// The repository is stateless — the only field is the shared
// *sql.DB handle injected from cmd/api/main.go. It may be
// constructed once and shared across all handlers.
type ProfilePricingRepository struct {
	db *sql.DB
}

// NewProfilePricingRepository returns a repository ready to talk to
// the given *sql.DB. Tuning the handle (SetMaxOpenConns, ...) is
// the caller's responsibility, as everywhere else in this package.
func NewProfilePricingRepository(db *sql.DB) *ProfilePricingRepository {
	return &ProfilePricingRepository{db: db}
}

// pricingSelectColumns enumerates every column the adapter reads
// when hydrating a *profilepricing.Pricing. Centralised so the
// single-org and batch read paths stay in sync — adding a new
// column means updating this string and the paired Scan call.
const pricingSelectColumns = `
	organization_id, pricing_kind, pricing_type,
	min_amount, max_amount, currency, pricing_note,
	created_at, updated_at`

// Upsert writes or updates the pricing row identified by
// (OrganizationID, Kind). Uses ON CONFLICT on the primary key so a
// second save of the same kind is an idempotent UPDATE, not an
// insertion error.
//
// created_at is NOT overwritten on conflict — the trigger
// profile_pricing_updated_at bumps updated_at automatically. The
// caller passes `DEFAULT` via omitting the column from the INSERT.
func (r *ProfilePricingRepository) Upsert(ctx context.Context, p *profilepricing.Pricing) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	query := `
		INSERT INTO profile_pricing (
			organization_id, pricing_kind, pricing_type,
			min_amount, max_amount, currency, pricing_note
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (organization_id, pricing_kind) DO UPDATE
		SET pricing_type = EXCLUDED.pricing_type,
		    min_amount   = EXCLUDED.min_amount,
		    max_amount   = EXCLUDED.max_amount,
		    currency     = EXCLUDED.currency,
		    pricing_note = EXCLUDED.pricing_note`

	var maxAmount sql.NullInt64
	if p.MaxAmount != nil {
		maxAmount = sql.NullInt64{Int64: *p.MaxAmount, Valid: true}
	}

	if _, err := r.db.ExecContext(ctx, query,
		p.OrganizationID, string(p.Kind), string(p.Type),
		p.MinAmount, maxAmount, p.Currency, p.Note,
	); err != nil {
		return fmt.Errorf("upsert profile pricing: %w", err)
	}
	return nil
}

// FindByOrgID returns every pricing row for the org, ordered by
// pricing_kind so callers receive direct-then-referral consistently.
// An empty (non-nil) slice is returned when the org has no pricing.
func (r *ProfilePricingRepository) FindByOrgID(ctx context.Context, orgID uuid.UUID) ([]*profilepricing.Pricing, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx,
		`SELECT `+pricingSelectColumns+`
		   FROM profile_pricing
		  WHERE organization_id = $1
		  ORDER BY pricing_kind`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("find profile pricing by org id: %w", err)
	}
	defer rows.Close()

	out := make([]*profilepricing.Pricing, 0, 2)
	for rows.Next() {
		p, err := scanPricingRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate profile pricing rows: %w", err)
	}
	return out, nil
}

// ListByOrgIDs is the batch variant used by listing endpoints.
// Seeds the return map with an empty slice for every input org so
// callers never need nil-checks, then fills in the rows from a
// single `WHERE organization_id = ANY($1)` query — N+1 impossible.
func (r *ProfilePricingRepository) ListByOrgIDs(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]*profilepricing.Pricing, error) {
	out := make(map[uuid.UUID][]*profilepricing.Pricing, len(orgIDs))
	for _, id := range orgIDs {
		out[id] = []*profilepricing.Pricing{}
	}
	if len(orgIDs) == 0 {
		return out, nil
	}

	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	idStrings := make([]string, len(orgIDs))
	for i, id := range orgIDs {
		idStrings[i] = id.String()
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT `+pricingSelectColumns+`
		   FROM profile_pricing
		  WHERE organization_id = ANY($1)
		  ORDER BY organization_id, pricing_kind`,
		pq.Array(idStrings),
	)
	if err != nil {
		return nil, fmt.Errorf("list profile pricing by org ids: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		p, err := scanPricingRow(rows)
		if err != nil {
			return nil, err
		}
		out[p.OrganizationID] = append(out[p.OrganizationID], p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate profile pricing rows: %w", err)
	}
	return out, nil
}

// DeleteByKind removes the (org_id, kind) row. No error when the
// row does not exist — deletion is idempotent so the UI can
// surface a delete button without racing.
func (r *ProfilePricingRepository) DeleteByKind(ctx context.Context, orgID uuid.UUID, kind profilepricing.PricingKind) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if _, err := r.db.ExecContext(ctx,
		`DELETE FROM profile_pricing WHERE organization_id = $1 AND pricing_kind = $2`,
		orgID, string(kind),
	); err != nil {
		return fmt.Errorf("delete profile pricing: %w", err)
	}
	return nil
}

// scanPricingRow decodes one SQL row into a *profilepricing.Pricing.
// Keeps the kind/type string-to-enum conversion in one place so
// both read paths stay consistent — the values have already been
// validated by the CHECK constraints in migration 083 and by
// NewPricing on write, so we cast without re-validating.
func scanPricingRow(rows *sql.Rows) (*profilepricing.Pricing, error) {
	var (
		p        profilepricing.Pricing
		kind     string
		ptype    string
		maxAmt   sql.NullInt64
		currency string
		note     string
	)
	if err := rows.Scan(
		&p.OrganizationID,
		&kind,
		&ptype,
		&p.MinAmount,
		&maxAmt,
		&currency,
		&note,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan profile pricing row: %w", err)
	}
	p.Kind = profilepricing.PricingKind(kind)
	p.Type = profilepricing.PricingType(ptype)
	p.Currency = currency
	p.Note = note
	if maxAmt.Valid {
		v := maxAmt.Int64
		p.MaxAmount = &v
	}
	return &p, nil
}
