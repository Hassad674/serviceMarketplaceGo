package repository

import (
	"context"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
)

// ReferralListFilter is the filter bag used by all the listing methods.
// Empty fields mean "no filter on this dimension".
type ReferralListFilter struct {
	Statuses []referral.Status
	Cursor   string
	Limit    int
}

// ReferralRepository persists Referral aggregates and their child rows.
//
// Modularity contract: this interface is the ONLY public surface the app/
// layer uses to touch the referral tables. The postgres adapter implements
// it and is wired in cmd/api/main.go. Other features never import this
// package — they receive the service ports defined in port/service/referral_*.go
// instead.
type ReferralRepository interface {
	// ─── Referral aggregate ──────────────────────────────────────────────

	// Create inserts a new referral. Returns ErrCoupleLocked if the partial
	// unique index on (provider_id, client_id) WHERE status non-terminal
	// is violated.
	Create(ctx context.Context, r *referral.Referral) error

	// GetByID loads a referral by its primary key. Returns ErrNotFound when
	// no row matches.
	GetByID(ctx context.Context, id uuid.UUID) (*referral.Referral, error)

	// Update persists state-machine mutations. The implementation MUST also
	// rewrite last_action_at and updated_at (already set by the entity).
	Update(ctx context.Context, r *referral.Referral) error

	// FindActiveByCouple returns the single non-terminal referral matching
	// the given (provider, client) pair, if any. Used by the create flow to
	// pre-empt the unique-violation race on a clean error path, and by the
	// attributor to look up which referral should attribute a new proposal.
	// Returns ErrNotFound if no active referral exists for the couple.
	FindActiveByCouple(ctx context.Context, providerID, clientID uuid.UUID) (*referral.Referral, error)

	// ListByReferrer returns all referrals where the requesting user is the
	// apporteur. Cursor-based pagination via meta.next_cursor.
	ListByReferrer(ctx context.Context, referrerID uuid.UUID, filter ReferralListFilter) (rows []*referral.Referral, nextCursor string, err error)

	// ListIncomingForProvider returns referrals where the requesting user is
	// the provider party (i.e., they need to act on or have acted on it).
	ListIncomingForProvider(ctx context.Context, providerID uuid.UUID, filter ReferralListFilter) (rows []*referral.Referral, nextCursor string, err error)

	// ListIncomingForClient returns referrals where the requesting user is
	// the client party.
	ListIncomingForClient(ctx context.Context, clientID uuid.UUID, filter ReferralListFilter) (rows []*referral.Referral, nextCursor string, err error)

	// ─── Negotiation audit trail ─────────────────────────────────────────

	// AppendNegotiation inserts a new negotiation row. The caller is
	// responsible for incrementing the parent referral's version when the
	// action represents a new rate proposal (countered/proposed).
	AppendNegotiation(ctx context.Context, n *referral.Negotiation) error

	// ListNegotiations returns the audit trail for a referral, oldest first
	// (so the UI can render a chronological timeline).
	ListNegotiations(ctx context.Context, referralID uuid.UUID) ([]*referral.Negotiation, error)

	// ─── Attribution ─────────────────────────────────────────────────────

	// CreateAttribution inserts a new attribution. Returns nil on duplicate
	// proposal_id (the UNIQUE(proposal_id) index acts as a no-op idempotency
	// guard for the attributor port — calling it twice for the same proposal
	// must NOT raise an error).
	CreateAttribution(ctx context.Context, a *referral.Attribution) error

	// FindAttributionByProposal returns the attribution row for a proposal,
	// or ErrAttributionNotFound when none exists. Used by the commission
	// distributor to know whether a milestone payout should split.
	FindAttributionByProposal(ctx context.Context, proposalID uuid.UUID) (*referral.Attribution, error)

	// ListAttributionsByReferral lists every proposal attributed to a
	// referral, for the dashboard timeline.
	ListAttributionsByReferral(ctx context.Context, referralID uuid.UUID) ([]*referral.Attribution, error)

	// ─── Commission ──────────────────────────────────────────────────────

	// CreateCommission inserts a commission row. Returns ErrCommissionAlreadyExists
	// on UNIQUE(attribution_id, milestone_id) violation — the distributor
	// uses this to short-circuit on retry without calling Stripe twice.
	CreateCommission(ctx context.Context, c *referral.Commission) error

	// UpdateCommission persists status transitions on a commission row.
	UpdateCommission(ctx context.Context, c *referral.Commission) error

	// FindCommissionByMilestone returns the commission for a milestone, or
	// ErrCommissionNotFound. Used by the clawback flow on refund.
	FindCommissionByMilestone(ctx context.Context, milestoneID uuid.UUID) (*referral.Commission, error)

	// ListCommissionsByReferral lists commissions across all attributions of
	// a referral, for the apporteur dashboard.
	ListCommissionsByReferral(ctx context.Context, referralID uuid.UUID) ([]*referral.Commission, error)

	// ListPendingKYCByReferrer returns commissions parked because the
	// referrer had no Stripe Connect account at payout time. Drained by
	// referral.kyc_listener.OnStripeAccountReady.
	ListPendingKYCByReferrer(ctx context.Context, referrerID uuid.UUID) ([]*referral.Commission, error)

	// ─── Cron support ─────────────────────────────────────────────────────

	// ListExpiringIntros returns referrals in a pending state with
	// last_action_at < cutoff. The expirer cron calls this with
	// (now - 14 days) to find intros that have been silent too long.
	ListExpiringIntros(ctx context.Context, cutoff time.Time, limit int) ([]*referral.Referral, error)

	// ListExpiringActives returns referrals with status=active AND
	// expires_at < now. The expirer cron transitions these to expired.
	ListExpiringActives(ctx context.Context, now time.Time, limit int) ([]*referral.Referral, error)

	// ─── Aggregations for dashboard ──────────────────────────────────────

	// CountByReferrer returns the count of referrals grouped by status for
	// the dashboard stat cards.
	CountByReferrer(ctx context.Context, referrerID uuid.UUID) (map[referral.Status]int, error)

	// SumCommissionsByReferrer returns total commission cents grouped by
	// status (paid, pending, pending_kyc, etc.) for the dashboard.
	SumCommissionsByReferrer(ctx context.Context, referrerID uuid.UUID) (map[referral.CommissionStatus]int64, error)
}
