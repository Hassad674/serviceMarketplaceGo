package service

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ReferralWalletReader exposes the apporteur's commission summary so
// the payment feature can surface it inside the /wallet overview
// without ever importing the referral package.
//
// Implementation contract (referral.Service satisfies this):
//   - `GetReferrerSummary` MUST NOT error on an unknown referrerID
//     (return a zero-valued summary instead). The wallet page is a
//     read-only surface and should degrade gracefully.
//   - `RecentCommissions` returns the most recent N commission rows
//     for the referrer, newest first. A hard cap of 100 is applied
//     upstream.
type ReferralWalletReader interface {
	GetReferrerSummary(ctx context.Context, referrerID uuid.UUID) (ReferrerCommissionSummary, error)
	RecentCommissions(ctx context.Context, referrerID uuid.UUID, limit int) ([]ReferralCommissionRecord, error)
}

// ReferrerCommissionSummary aggregates the 4 wallet-relevant statuses
// so the UI can render commission cards with the same grammar as the
// existing "en escrow / disponible / transféré" cards.
//
// All amounts are in minor units of the referrer's currency (cents for
// EUR). Mixed-currency summaries are not supported in V1 — when a
// referrer does business in multiple currencies, the UI will need to
// group per currency (TODO).
type ReferrerCommissionSummary struct {
	PendingCents     int64 // status=pending — queued, transfer not yet attempted
	PendingKYCCents  int64 // status=pending_kyc — waiting on apporteur KYC
	PaidCents        int64 // status=paid — money sent to the apporteur's Stripe
	ClawedBackCents  int64 // status=clawed_back — reversed after a refund
	Currency         string
}

// ReferralCommissionRecord is one history row for the /wallet
// commissions section. Mirrors the shape of the internal Commission
// entity but uses primitive types only (no referral package import).
type ReferralCommissionRecord struct {
	ID               uuid.UUID
	ReferralID       uuid.UUID
	ProposalID       uuid.UUID
	MilestoneID      uuid.UUID
	GrossAmountCents int64
	CommissionCents  int64
	Currency         string
	Status           string
	StripeTransferID string
	StripeReversalID string
	PaidAt           *time.Time
	ClawedBackAt     *time.Time
	CreatedAt        time.Time
}
