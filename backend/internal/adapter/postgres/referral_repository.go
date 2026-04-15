package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/cursor"
)

// ReferralRepository is the postgres-backed implementation of
// repository.ReferralRepository. It persists the four tables created by
// migrations 105-108: referrals, referral_negotiations, referral_attributions,
// referral_commissions.
type ReferralRepository struct {
	db *sql.DB
}

// Compile-time assertion that ReferralRepository satisfies the port contract.
var _ repository.ReferralRepository = (*ReferralRepository)(nil)

func NewReferralRepository(db *sql.DB) *ReferralRepository {
	return &ReferralRepository{db: db}
}

const dbTimeout = 5 * time.Second

// ─── Referral aggregate ────────────────────────────────────────────────────

func (r *ReferralRepository) Create(ctx context.Context, ref *referral.Referral) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	snapshot, err := referral.MarshalSnapshot(ref.IntroSnapshot)
	if err != nil {
		return fmt.Errorf("marshal intro snapshot: %w", err)
	}

	_, err = r.db.ExecContext(ctx, queryInsertReferral,
		ref.ID, ref.ReferrerID, ref.ProviderID, ref.ClientID,
		ref.RatePct, ref.DurationMonths,
		snapshot, ref.IntroSnapshotVersion,
		ref.IntroMessageProvider, ref.IntroMessageClient,
		string(ref.Status), ref.Version,
		ref.ActivatedAt, ref.ExpiresAt, ref.LastActionAt,
		ref.RejectionReason, ref.RejectedBy,
		ref.CreatedAt, ref.UpdatedAt,
	)
	if err != nil {
		// The partial unique index on (provider_id, client_id) WHERE
		// status non-terminal raises a 23505 unique violation when a
		// concurrent referral already locks the couple.
		if pqErr := (*pq.Error)(nil); errors.As(err, &pqErr) && pqErr.Code == "23505" {
			if strings.Contains(pqErr.Constraint, "active_couple") {
				return referral.ErrCoupleLocked
			}
		}
		return fmt.Errorf("insert referral: %w", err)
	}
	return nil
}

func (r *ReferralRepository) GetByID(ctx context.Context, id uuid.UUID) (*referral.Referral, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	ref, err := scanReferral(r.db.QueryRowContext(ctx, queryGetReferralByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, referral.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get referral by id: %w", err)
	}
	return ref, nil
}

func (r *ReferralRepository) Update(ctx context.Context, ref *referral.Referral) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	res, err := r.db.ExecContext(ctx, queryUpdateReferral,
		ref.ID, ref.RatePct, ref.DurationMonths,
		string(ref.Status), ref.Version,
		ref.ActivatedAt, ref.ExpiresAt, ref.LastActionAt,
		ref.RejectionReason, ref.RejectedBy,
	)
	if err != nil {
		return fmt.Errorf("update referral: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update referral rows affected: %w", err)
	}
	if rows == 0 {
		return referral.ErrNotFound
	}
	return nil
}

func (r *ReferralRepository) FindActiveByCouple(ctx context.Context, providerID, clientID uuid.UUID) (*referral.Referral, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	ref, err := scanReferral(r.db.QueryRowContext(ctx, queryFindActiveReferralByCouple, providerID, clientID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, referral.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find referral by couple: %w", err)
	}
	return ref, nil
}

// ─── Listing ───────────────────────────────────────────────────────────────

func (r *ReferralRepository) ListByReferrer(ctx context.Context, referrerID uuid.UUID, filter repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return r.listByActor(ctx, "referrer_id", referrerID, filter)
}

func (r *ReferralRepository) ListIncomingForProvider(ctx context.Context, providerID uuid.UUID, filter repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return r.listByActor(ctx, "provider_id", providerID, filter)
}

func (r *ReferralRepository) ListIncomingForClient(ctx context.Context, clientID uuid.UUID, filter repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	return r.listByActor(ctx, "client_id", clientID, filter)
}

// listByActor backs the three list endpoints. The actor column is hard-coded
// at the call site (referrer_id / provider_id / client_id) so it is safe to
// interpolate via fmt.Sprintf — there is no user-controlled column name.
func (r *ReferralRepository) listByActor(ctx context.Context, actorCol string, actorID uuid.UUID, filter repository.ReferralListFilter) ([]*referral.Referral, string, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{actorID}
	statusClause := ""
	if len(filter.Statuses) > 0 {
		args = append(args, statusesToStrings(filter.Statuses))
		statusClause = fmt.Sprintf("AND status = ANY($%d)", len(args))
	}

	cursorClause := ""
	if filter.Cursor != "" {
		c, err := cursor.Decode(filter.Cursor)
		if err != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", err)
		}
		args = append(args, c.CreatedAt)
		args = append(args, c.ID)
		cursorClause = fmt.Sprintf("AND (created_at, id) < ($%d, $%d)", len(args)-1, len(args))
	}

	args = append(args, limit+1) // fetch one extra to detect has_more
	query := fmt.Sprintf(queryListReferralsTemplate, actorCol, statusClause, cursorClause, len(args))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("list referrals: %w", err)
	}
	defer rows.Close()

	referrals, err := scanReferralRows(rows)
	if err != nil {
		return nil, "", err
	}

	nextCursor := ""
	if len(referrals) > limit {
		last := referrals[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		referrals = referrals[:limit]
	}
	return referrals, nextCursor, nil
}

// ─── Negotiations ──────────────────────────────────────────────────────────

func (r *ReferralRepository) AppendNegotiation(ctx context.Context, n *referral.Negotiation) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertNegotiation,
		n.ID, n.ReferralID, n.Version, n.ActorID, string(n.ActorRole),
		string(n.Action), n.RatePct, n.Message, n.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert negotiation: %w", err)
	}
	return nil
}

func (r *ReferralRepository) ListNegotiations(ctx context.Context, referralID uuid.UUID) ([]*referral.Negotiation, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListNegotiations, referralID)
	if err != nil {
		return nil, fmt.Errorf("list negotiations: %w", err)
	}
	defer rows.Close()

	var out []*referral.Negotiation
	for rows.Next() {
		n := &referral.Negotiation{}
		var actorRole, action string
		if err := rows.Scan(&n.ID, &n.ReferralID, &n.Version, &n.ActorID, &actorRole, &action, &n.RatePct, &n.Message, &n.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan negotiation: %w", err)
		}
		n.ActorRole = referral.ActorRole(actorRole)
		n.Action = referral.NegotiationAction(action)
		out = append(out, n)
	}
	return out, rows.Err()
}

// ─── Attributions ──────────────────────────────────────────────────────────

func (r *ReferralRepository) CreateAttribution(ctx context.Context, a *referral.Attribution) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	// ON CONFLICT (proposal_id) DO NOTHING — the attributor port contract
	// says a second call for the same proposal is a no-op, not an error.
	_, err := r.db.ExecContext(ctx, queryInsertAttribution,
		a.ID, a.ReferralID, a.ProposalID, a.ProviderID, a.ClientID, a.RatePctSnapshot, a.AttributedAt,
	)
	if err != nil {
		return fmt.Errorf("insert attribution: %w", err)
	}
	return nil
}

func (r *ReferralRepository) FindAttributionByProposal(ctx context.Context, proposalID uuid.UUID) (*referral.Attribution, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	a := &referral.Attribution{}
	err := r.db.QueryRowContext(ctx, queryFindAttributionByProposal, proposalID).Scan(
		&a.ID, &a.ReferralID, &a.ProposalID, &a.ProviderID, &a.ClientID, &a.RatePctSnapshot, &a.AttributedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, referral.ErrAttributionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find attribution by proposal: %w", err)
	}
	return a, nil
}

func (r *ReferralRepository) FindAttributionByID(ctx context.Context, id uuid.UUID) (*referral.Attribution, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	a := &referral.Attribution{}
	err := r.db.QueryRowContext(ctx, queryFindAttributionByID, id).Scan(
		&a.ID, &a.ReferralID, &a.ProposalID, &a.ProviderID, &a.ClientID, &a.RatePctSnapshot, &a.AttributedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, referral.ErrAttributionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find attribution by id: %w", err)
	}
	return a, nil
}

func (r *ReferralRepository) ListAttributionsByReferral(ctx context.Context, referralID uuid.UUID) ([]*referral.Attribution, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListAttributionsByReferral, referralID)
	if err != nil {
		return nil, fmt.Errorf("list attributions: %w", err)
	}
	defer rows.Close()

	var out []*referral.Attribution
	for rows.Next() {
		a := &referral.Attribution{}
		if err := rows.Scan(&a.ID, &a.ReferralID, &a.ProposalID, &a.ProviderID, &a.ClientID, &a.RatePctSnapshot, &a.AttributedAt); err != nil {
			return nil, fmt.Errorf("scan attribution: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ─── Commissions ───────────────────────────────────────────────────────────

func (r *ReferralRepository) CreateCommission(ctx context.Context, c *referral.Commission) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertCommission,
		c.ID, c.AttributionID, c.MilestoneID,
		c.GrossAmountCents, c.CommissionCents, c.Currency,
		string(c.Status), c.StripeTransferID, c.StripeReversalID, c.FailureReason,
		c.PaidAt, c.ClawedBackAt, c.CreatedAt, c.UpdatedAt,
	)
	if err != nil {
		// UNIQUE(attribution_id, milestone_id) — the distributor uses this
		// to detect that another retry already created the row and skip.
		if pqErr := (*pq.Error)(nil); errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return referral.ErrCommissionAlreadyExists
		}
		return fmt.Errorf("insert commission: %w", err)
	}
	return nil
}

func (r *ReferralRepository) UpdateCommission(ctx context.Context, c *referral.Commission) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	res, err := r.db.ExecContext(ctx, queryUpdateCommission,
		c.ID, string(c.Status), c.StripeTransferID, c.StripeReversalID, c.FailureReason, c.PaidAt, c.ClawedBackAt,
	)
	if err != nil {
		return fmt.Errorf("update commission: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update commission rows affected: %w", err)
	}
	if rows == 0 {
		return referral.ErrCommissionNotFound
	}
	return nil
}

func (r *ReferralRepository) FindCommissionByMilestone(ctx context.Context, milestoneID uuid.UUID) (*referral.Commission, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	c, err := scanCommission(r.db.QueryRowContext(ctx, queryFindCommissionByMilestone, milestoneID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, referral.ErrCommissionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find commission by milestone: %w", err)
	}
	return c, nil
}

func (r *ReferralRepository) ListCommissionsByReferral(ctx context.Context, referralID uuid.UUID) ([]*referral.Commission, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListCommissionsByReferral, referralID)
	if err != nil {
		return nil, fmt.Errorf("list commissions by referral: %w", err)
	}
	defer rows.Close()
	return scanCommissionRows(rows)
}

func (r *ReferralRepository) ListPendingKYCByReferrer(ctx context.Context, referrerID uuid.UUID) ([]*referral.Commission, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListPendingKYCByReferrer, referrerID)
	if err != nil {
		return nil, fmt.Errorf("list pending_kyc commissions: %w", err)
	}
	defer rows.Close()
	return scanCommissionRows(rows)
}

// ─── Cron support ──────────────────────────────────────────────────────────

func (r *ReferralRepository) ListExpiringIntros(ctx context.Context, cutoff time.Time, limit int) ([]*referral.Referral, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, queryListExpiringIntros, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("list expiring intros: %w", err)
	}
	defer rows.Close()
	return scanReferralRows(rows)
}

func (r *ReferralRepository) ListExpiringActives(ctx context.Context, now time.Time, limit int) ([]*referral.Referral, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, queryListExpiringActives, now, limit)
	if err != nil {
		return nil, fmt.Errorf("list expiring actives: %w", err)
	}
	defer rows.Close()
	return scanReferralRows(rows)
}

// ─── Aggregations ──────────────────────────────────────────────────────────

func (r *ReferralRepository) CountByReferrer(ctx context.Context, referrerID uuid.UUID) (map[referral.Status]int, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryCountByReferrer, referrerID)
	if err != nil {
		return nil, fmt.Errorf("count referrals by status: %w", err)
	}
	defer rows.Close()

	out := make(map[referral.Status]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("scan count row: %w", err)
		}
		out[referral.Status(status)] = count
	}
	return out, rows.Err()
}

func (r *ReferralRepository) SumCommissionsByReferrer(ctx context.Context, referrerID uuid.UUID) (map[referral.CommissionStatus]int64, error) {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, querySumCommissionsByReferrer, referrerID)
	if err != nil {
		return nil, fmt.Errorf("sum commissions by status: %w", err)
	}
	defer rows.Close()

	out := make(map[referral.CommissionStatus]int64)
	for rows.Next() {
		var status string
		var sum int64
		if err := rows.Scan(&status, &sum); err != nil {
			return nil, fmt.Errorf("scan sum row: %w", err)
		}
		out[referral.CommissionStatus(status)] = sum
	}
	return out, rows.Err()
}

// ─── Scan helpers ──────────────────────────────────────────────────────────

func scanReferral(row *sql.Row) (*referral.Referral, error) {
	ref := &referral.Referral{}
	var snapshotRaw []byte
	var status string
	if err := row.Scan(
		&ref.ID, &ref.ReferrerID, &ref.ProviderID, &ref.ClientID,
		&ref.RatePct, &ref.DurationMonths,
		&snapshotRaw, &ref.IntroSnapshotVersion,
		&ref.IntroMessageProvider, &ref.IntroMessageClient,
		&status, &ref.Version,
		&ref.ActivatedAt, &ref.ExpiresAt, &ref.LastActionAt,
		&ref.RejectionReason, &ref.RejectedBy,
		&ref.CreatedAt, &ref.UpdatedAt,
	); err != nil {
		return nil, err
	}
	ref.Status = referral.Status(status)
	snapshot, err := referral.UnmarshalSnapshot(snapshotRaw)
	if err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	ref.IntroSnapshot = snapshot
	return ref, nil
}

func scanReferralRows(rows *sql.Rows) ([]*referral.Referral, error) {
	var out []*referral.Referral
	for rows.Next() {
		ref := &referral.Referral{}
		var snapshotRaw []byte
		var status string
		if err := rows.Scan(
			&ref.ID, &ref.ReferrerID, &ref.ProviderID, &ref.ClientID,
			&ref.RatePct, &ref.DurationMonths,
			&snapshotRaw, &ref.IntroSnapshotVersion,
			&ref.IntroMessageProvider, &ref.IntroMessageClient,
			&status, &ref.Version,
			&ref.ActivatedAt, &ref.ExpiresAt, &ref.LastActionAt,
			&ref.RejectionReason, &ref.RejectedBy,
			&ref.CreatedAt, &ref.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan referral row: %w", err)
		}
		ref.Status = referral.Status(status)
		snapshot, err := referral.UnmarshalSnapshot(snapshotRaw)
		if err != nil {
			return nil, fmt.Errorf("unmarshal snapshot in row: %w", err)
		}
		ref.IntroSnapshot = snapshot
		out = append(out, ref)
	}
	return out, rows.Err()
}

func scanCommission(row *sql.Row) (*referral.Commission, error) {
	c := &referral.Commission{}
	var status string
	if err := row.Scan(
		&c.ID, &c.AttributionID, &c.MilestoneID,
		&c.GrossAmountCents, &c.CommissionCents, &c.Currency,
		&status, &c.StripeTransferID, &c.StripeReversalID, &c.FailureReason,
		&c.PaidAt, &c.ClawedBackAt, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return nil, err
	}
	c.Status = referral.CommissionStatus(status)
	return c, nil
}

func scanCommissionRows(rows *sql.Rows) ([]*referral.Commission, error) {
	var out []*referral.Commission
	for rows.Next() {
		c := &referral.Commission{}
		var status string
		if err := rows.Scan(
			&c.ID, &c.AttributionID, &c.MilestoneID,
			&c.GrossAmountCents, &c.CommissionCents, &c.Currency,
			&status, &c.StripeTransferID, &c.StripeReversalID, &c.FailureReason,
			&c.PaidAt, &c.ClawedBackAt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan commission row: %w", err)
		}
		c.Status = referral.CommissionStatus(status)
		out = append(out, c)
	}
	return out, rows.Err()
}

func statusesToStrings(in []referral.Status) pq.StringArray {
	out := make(pq.StringArray, 0, len(in))
	for _, s := range in {
		out = append(out, string(s))
	}
	return out
}
