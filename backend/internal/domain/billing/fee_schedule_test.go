package billing_test

import (
	"testing"

	"marketplace-backend/internal/domain/billing"

	"github.com/stretchr/testify/assert"
)

func TestCalculate_Freelance(t *testing.T) {
	tests := []struct {
		name          string
		amountCents   int64
		wantFeeCents  int64
		wantNetCents  int64
		wantTierIndex int
	}{
		{"zero amount", 0, 0, 0, -1},
		{"1 cent", 1, 900, -899, 0},
		{"below tier 1 upper bound", 19999, 900, 19099, 0},
		{"at tier 1 upper bound promotes to tier 2", 20000, 1500, 18500, 1},
		{"middle of tier 2", 50000, 1500, 48500, 1},
		{"below tier 2 upper bound", 99999, 1500, 98499, 1},
		{"at tier 2 upper bound promotes to tier 3", 100000, 2500, 97500, 2},
		{"well above tier 2 upper bound", 500000, 2500, 497500, 2},
		{"very large amount", 10_000_000, 2500, 9_997_500, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := billing.Calculate(billing.RoleFreelance, tc.amountCents)
			assert.Equal(t, tc.amountCents, got.AmountCents)
			assert.Equal(t, tc.wantFeeCents, got.FeeCents, "fee cents")
			assert.Equal(t, tc.wantNetCents, got.NetCents, "net cents")
			assert.Equal(t, tc.wantTierIndex, got.ActiveTierIndex, "active tier index")
			assert.Equal(t, billing.RoleFreelance, got.Role)
			assert.Len(t, got.Tiers, 3)
		})
	}
}

func TestCalculate_Agency(t *testing.T) {
	tests := []struct {
		name          string
		amountCents   int64
		wantFeeCents  int64
		wantNetCents  int64
		wantTierIndex int
	}{
		{"zero amount", 0, 0, 0, -1},
		{"1 cent", 1, 1900, -1899, 0},
		{"below tier 1 upper bound", 49999, 1900, 48099, 0},
		{"at tier 1 upper bound promotes to tier 2", 50000, 3900, 46100, 1},
		{"middle of tier 2", 150000, 3900, 146100, 1},
		{"below tier 2 upper bound", 249999, 3900, 246099, 1},
		{"at tier 2 upper bound promotes to tier 3", 250000, 6900, 243100, 2},
		{"well above tier 2 upper bound", 1_000_000, 6900, 993100, 2},
		{"very large amount", 50_000_000, 6900, 49_993_100, 2},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := billing.Calculate(billing.RoleAgency, tc.amountCents)
			assert.Equal(t, tc.amountCents, got.AmountCents)
			assert.Equal(t, tc.wantFeeCents, got.FeeCents, "fee cents")
			assert.Equal(t, tc.wantNetCents, got.NetCents, "net cents")
			assert.Equal(t, tc.wantTierIndex, got.ActiveTierIndex, "active tier index")
			assert.Equal(t, billing.RoleAgency, got.Role)
			assert.Len(t, got.Tiers, 3)
		})
	}
}

func TestCalculate_NegativeAmount_ReturnsZeroFee(t *testing.T) {
	for _, role := range []billing.Role{billing.RoleFreelance, billing.RoleAgency} {
		t.Run(string(role), func(t *testing.T) {
			got := billing.Calculate(role, -1)
			assert.Equal(t, int64(0), got.FeeCents)
			assert.Equal(t, -1, got.ActiveTierIndex)
		})
	}
}

func TestCalculate_UnknownRole_FallsBackToFreelance(t *testing.T) {
	// Defensive: an unexpected role value should not panic or return an empty
	// grid. We fall back to the freelance schedule which is the cheaper side,
	// so any bug skews toward under-charging rather than over-charging.
	got := billing.Calculate(billing.Role("unknown"), 50000)
	assert.Equal(t, int64(1500), got.FeeCents)
	assert.Len(t, got.Tiers, 3)
}

func TestTiersFor_ReturnsCopy(t *testing.T) {
	tiers := billing.TiersFor(billing.RoleFreelance)
	tiers[0].FeeCents = 99999
	fresh := billing.TiersFor(billing.RoleFreelance)
	assert.Equal(t, int64(900), fresh[0].FeeCents, "caller mutation must not leak into package-level schedule")
}

func TestRole_IsValid(t *testing.T) {
	assert.True(t, billing.RoleFreelance.IsValid())
	assert.True(t, billing.RoleAgency.IsValid())
	assert.False(t, billing.Role("enterprise").IsValid())
	assert.False(t, billing.Role("").IsValid())
}

func TestRoleFromUser(t *testing.T) {
	tests := []struct {
		userRole string
		want     billing.Role
	}{
		{"agency", billing.RoleAgency},
		{"provider", billing.RoleFreelance},
		{"enterprise", billing.RoleFreelance},
		{"admin", billing.RoleFreelance},
		{"", billing.RoleFreelance},
		{"unknown", billing.RoleFreelance},
	}
	for _, tc := range tests {
		t.Run(tc.userRole, func(t *testing.T) {
			assert.Equal(t, tc.want, billing.RoleFromUser(tc.userRole))
		})
	}
}

func TestSchedule_BoundaryValues(t *testing.T) {
	// Explicit assertion that the tiered grid matches the product spec.
	// If anyone changes the fee schedule, this test must fail loudly to
	// force a review of all downstream consumers (fee preview UI, invoice
	// labels, marketing pages).
	freelance := billing.TiersFor(billing.RoleFreelance)
	assert.Equal(t, int64(900), freelance[0].FeeCents)
	assert.Equal(t, int64(20000), *freelance[0].MaxCents)
	assert.Equal(t, int64(1500), freelance[1].FeeCents)
	assert.Equal(t, int64(100000), *freelance[1].MaxCents)
	assert.Equal(t, int64(2500), freelance[2].FeeCents)
	assert.Nil(t, freelance[2].MaxCents)

	agency := billing.TiersFor(billing.RoleAgency)
	assert.Equal(t, int64(1900), agency[0].FeeCents)
	assert.Equal(t, int64(50000), *agency[0].MaxCents)
	assert.Equal(t, int64(3900), agency[1].FeeCents)
	assert.Equal(t, int64(250000), *agency[1].MaxCents)
	assert.Equal(t, int64(6900), agency[2].FeeCents)
	assert.Nil(t, agency[2].MaxCents)
}
