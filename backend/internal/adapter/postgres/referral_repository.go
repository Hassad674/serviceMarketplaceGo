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

// ListAttributionsByReferralIDs batch-loads attributions for the given
// referral ids. Empty input returns an empty slice without hitting the
// DB.
func (r *ReferralRepository) ListAttributionsByReferralIDs(ctx context.Context, referralIDs []uuid.UUID) ([]*referral.Attribution, error) {
	if len(referralIDs) == 0 {
		return []*referral.Attribution{}, nil
	}
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	ids := make([]string, len(referralIDs))
	for i, id := range referralIDs {
		ids[i] = id.String()
	}

	rows, err := r.db.QueryContext(ctx, queryListAttributionsByReferralIDs, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("list attributions by referral ids: %w", err)
	}
	defer rows.Close()

	out := make([]*referral.Attribution, 0)
	for rows.Next() {
		a := &referral.Attribution{}
		if err := rows.Scan(&a.ID, &a.ReferralID, &a.ProposalID, &a.ProviderID, &a.ClientID, &a.RatePctSnapshot, &a.AttributedAt); err != nil {
			return nil, fmt.Errorf("scan attribution: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
