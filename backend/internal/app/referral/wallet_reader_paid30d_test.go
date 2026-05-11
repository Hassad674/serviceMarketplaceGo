package referral_test

// Test focused on the WALLET-UX extension to GetReferrerSummary —
// Paid30dCents (rolling 30-day window) and LifetimeCents (paid +
// clawed_back). The fixture seeds commission rows with explicit
// PaidAt timestamps to exercise the cutoff boundary.

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
)

func TestGetReferrerSummary_Paid30dAndLifetime(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	refID, provID, cliID := f.seedActors(t)
	r := f.createIntro(t, refID, provID, cliID, 5)
	bringToActive(t, f.svc, r, provID, cliID)

	// Seed one attribution + 3 commission rows directly on the fake repo
	// so we control PaidAt timestamps exactly:
	//   row A: paid 5 days ago     →  in window (Paid30d + Lifetime)
	//   row B: paid 45 days ago    →  out of window (Lifetime only)
	//   row C: clawed back, no PaidAt → Lifetime only
	proposalID := uuid.New()
	att := &referral.Attribution{
		ID:              uuid.New(),
		ReferralID:      r.ID,
		ProposalID:      proposalID,
		AttributedAt:    time.Now(),
		RatePctSnapshot: 5,
	}
	f.repo.attributionsByID[att.ID] = att
	f.repo.attributions[att.ProposalID] = att

	now := time.Now()
	recent := now.Add(-5 * 24 * time.Hour)
	old := now.Add(-45 * 24 * time.Hour)

	mkPaid := func(amount int64, paidAt time.Time) *referral.Commission {
		return &referral.Commission{
			ID:               uuid.New(),
			AttributionID:    att.ID,
			MilestoneID:      uuid.New(),
			GrossAmountCents: amount * 20,
			CommissionCents:  amount,
			Currency:         "EUR",
			Status:           referral.CommissionPaid,
			StripeTransferID: "tr_x",
			PaidAt:           &paidAt,
			CreatedAt:        paidAt,
		}
	}
	mkClawed := func(amount int64) *referral.Commission {
		return &referral.Commission{
			ID:               uuid.New(),
			AttributionID:    att.ID,
			MilestoneID:      uuid.New(),
			GrossAmountCents: amount * 20,
			CommissionCents:  amount,
			Currency:         "EUR",
			Status:           referral.CommissionClawedBack,
			CreatedAt:        now,
		}
	}

	require.NoError(t, f.repo.CreateCommission(context.Background(), mkPaid(10_000, recent)))
	require.NoError(t, f.repo.CreateCommission(context.Background(), mkPaid(20_000, old)))
	require.NoError(t, f.repo.CreateCommission(context.Background(), mkClawed(3_000)))

	sum, err := f.svc.GetReferrerSummary(context.Background(), refID)
	require.NoError(t, err)
	assert.Equal(t, int64(30_000), sum.PaidCents, "PaidCents = sum of paid rows regardless of age")
	assert.Equal(t, int64(3_000), sum.ClawedBackCents)
	assert.Equal(t, int64(10_000), sum.Paid30dCents, "Paid30d only counts paid_at within 30 days")
	assert.Equal(t, int64(33_000), sum.LifetimeCents, "Lifetime = Paid + ClawedBack")
	assert.Equal(t, "EUR", sum.Currency)
}

func TestGetReferrerSummary_NoPaidRows_Paid30dIsZero(t *testing.T) {
	f := newTestFixture(t, "acct_referrer")
	sum, err := f.svc.GetReferrerSummary(context.Background(), uuid.New())
	require.NoError(t, err)
	assert.Zero(t, sum.Paid30dCents)
	assert.Zero(t, sum.LifetimeCents)
}
