package repository

// This file declares the SEGREGATED reader/writer/store interfaces for
// the referral feature, applying the Interface Segregation Principle to
// the wider ReferralRepository contract defined in referral_repository.go.
//
// Rationale: ReferralRepository carries 24 methods because the same
// postgres adapter persists referrals, negotiations, attributions, AND
// commissions. Most consumers only need one of those four sub-domains —
// dashboards read; the create/update flow writes; the commission
// distributor only needs commission methods; the attribution lookup only
// needs attribution methods.
//
// By depending on a smaller interface, an app service:
//   - declares its real surface (easier to grep, easier to mock)
//   - keeps its mock under 5 minutes of work (CLAUDE.md ISP rule)
//   - is testable without 24 panic-stub methods
//
// Implementation note: the postgres adapter struct satisfies ALL of
// these interfaces because Go uses structural typing — the existing
// adapter file `internal/adapter/postgres/referral_repository.go` was
// not touched. Wiring in `cmd/api/main.go` continues to pass the
// concrete struct; consumers tighten their declared dependency type.

import (
	"context"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/referral"
)

// ReferralReader exposes read paths over the referral aggregate and its
// negotiation audit trail. Wallet pages, dashboards, the apporteur
// profile and reputation aggregator all rely on this — none of them
// mutate referrals directly.
type ReferralReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*referral.Referral, error)
	FindActiveByCouple(ctx context.Context, providerID, clientID uuid.UUID) (*referral.Referral, error)
	ListByReferrer(ctx context.Context, referrerID uuid.UUID, filter ReferralListFilter) (rows []*referral.Referral, nextCursor string, err error)
	ListIncomingForProvider(ctx context.Context, providerID uuid.UUID, filter ReferralListFilter) (rows []*referral.Referral, nextCursor string, err error)
	ListIncomingForClient(ctx context.Context, clientID uuid.UUID, filter ReferralListFilter) (rows []*referral.Referral, nextCursor string, err error)
	ListNegotiations(ctx context.Context, referralID uuid.UUID) ([]*referral.Negotiation, error)

	// Cron support — scheduler reads.
	ListExpiringIntros(ctx context.Context, cutoff time.Time, limit int) ([]*referral.Referral, error)
	ListExpiringActives(ctx context.Context, now time.Time, limit int) ([]*referral.Referral, error)

	// Aggregations for the apporteur dashboard.
	CountByReferrer(ctx context.Context, referrerID uuid.UUID) (map[referral.Status]int, error)
	SumCommissionsByReferrer(ctx context.Context, referrerID uuid.UUID) (map[referral.CommissionStatus]int64, error)
}

// ReferralWriter exposes mutation paths over the referral aggregate and
// its negotiation history. Used by the create/update flow and the
// state-machine transitions in app/referral.
type ReferralWriter interface {
	Create(ctx context.Context, r *referral.Referral) error
	Update(ctx context.Context, r *referral.Referral) error
	AppendNegotiation(ctx context.Context, n *referral.Negotiation) error
}

// ReferralAttributionStore covers attribution rows — created when a
// proposal anchors onto a referral, read by the commission distributor
// and the dashboard.
type ReferralAttributionStore interface {
	CreateAttribution(ctx context.Context, a *referral.Attribution) error
	FindAttributionByProposal(ctx context.Context, proposalID uuid.UUID) (*referral.Attribution, error)
	FindAttributionByID(ctx context.Context, id uuid.UUID) (*referral.Attribution, error)
	ListAttributionsByReferral(ctx context.Context, referralID uuid.UUID) ([]*referral.Attribution, error)
	ListAttributionsByReferralIDs(ctx context.Context, referralIDs []uuid.UUID) ([]*referral.Attribution, error)
}

// ReferralCommissionStore covers commission rows — written by the
// distributor on milestone release, read by the wallet, the clawback
// flow on refund, and the kyc-listener that drains the pending-kyc
// bucket once an apporteur finishes onboarding.
type ReferralCommissionStore interface {
	CreateCommission(ctx context.Context, c *referral.Commission) error
	UpdateCommission(ctx context.Context, c *referral.Commission) error
	FindCommissionByMilestone(ctx context.Context, milestoneID uuid.UUID) (*referral.Commission, error)
	ListCommissionsByReferral(ctx context.Context, referralID uuid.UUID) ([]*referral.Commission, error)
	ListPendingKYCByReferrer(ctx context.Context, referrerID uuid.UUID) ([]*referral.Commission, error)
	ListRecentCommissionsByReferrer(ctx context.Context, referrerID uuid.UUID, limit int) ([]*referral.Commission, error)
}

// Compile-time guarantee that the wide ReferralRepository contract is
// always equivalent to the union of its segregated children. If a new
// method ever lands on ReferralRepository without being categorised here,
// the build breaks and the author must place it in the correct child
// interface (or accept they have a 25th method that doesn't fit any of
// the four buckets — a strong signal that the refactor needs revisiting).
var _ ReferralRepository = (interface {
	ReferralReader
	ReferralWriter
	ReferralAttributionStore
	ReferralCommissionStore
})(nil)
