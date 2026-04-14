package profilepricing

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- PricingKind / PricingType enum validity ----

func TestPricingKind_IsValid(t *testing.T) {
	tests := []struct {
		name string
		in   PricingKind
		want bool
	}{
		{"direct", KindDirect, true},
		{"referral", KindReferral, true},
		{"empty", "", false},
		{"unknown", "wild", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.in.IsValid())
		})
	}
}

func TestPricingType_IsValid(t *testing.T) {
	valid := []PricingType{
		TypeDaily, TypeHourly, TypeProjectFrom, TypeProjectRange,
		TypeCommissionPct, TypeCommissionFlat,
	}
	for _, tc := range valid {
		t.Run(string(tc), func(t *testing.T) {
			assert.True(t, tc.IsValid())
		})
	}
	assert.False(t, PricingType("").IsValid())
	assert.False(t, PricingType("monthly").IsValid())
}

// ---- AllowedTypesForKind + IsTypeAllowedForKind ----

func TestAllowedTypesForKind(t *testing.T) {
	direct := AllowedTypesForKind(KindDirect)
	assert.ElementsMatch(t,
		[]PricingType{TypeDaily, TypeHourly, TypeProjectFrom, TypeProjectRange},
		direct,
	)

	referral := AllowedTypesForKind(KindReferral)
	assert.ElementsMatch(t,
		[]PricingType{TypeCommissionPct, TypeCommissionFlat},
		referral,
	)

	assert.Nil(t, AllowedTypesForKind(PricingKind("bogus")))
}

func TestIsTypeAllowedForKind(t *testing.T) {
	tests := []struct {
		kind PricingKind
		ptyp PricingType
		want bool
	}{
		{KindDirect, TypeDaily, true},
		{KindDirect, TypeHourly, true},
		{KindDirect, TypeProjectFrom, true},
		{KindDirect, TypeProjectRange, true},
		{KindDirect, TypeCommissionPct, false},
		{KindDirect, TypeCommissionFlat, false},
		{KindReferral, TypeCommissionPct, true},
		{KindReferral, TypeCommissionFlat, true},
		{KindReferral, TypeDaily, false},
		{KindReferral, TypeHourly, false},
		{PricingKind("bogus"), TypeDaily, false},
	}
	for _, tc := range tests {
		t.Run(string(tc.kind)+"/"+string(tc.ptyp), func(t *testing.T) {
			assert.Equal(t, tc.want, IsTypeAllowedForKind(tc.kind, tc.ptyp))
		})
	}
}

// ---- IsKindAllowedForOrg ----

func TestIsKindAllowedForOrg(t *testing.T) {
	tests := []struct {
		name            string
		orgType         OrgType
		referrerEnabled bool
		kind            PricingKind
		want            bool
	}{
		{"agency direct ok", OrgTypeAgency, false, KindDirect, true},
		{"agency referral refused", OrgTypeAgency, false, KindReferral, false},
		{"agency referral refused even with referrer bool true", OrgTypeAgency, true, KindReferral, false},
		{"provider direct ok without referrer", OrgTypeProviderPersonal, false, KindDirect, true},
		{"provider direct ok with referrer", OrgTypeProviderPersonal, true, KindDirect, true},
		{"provider referral refused without referrer", OrgTypeProviderPersonal, false, KindReferral, false},
		{"provider referral ok with referrer", OrgTypeProviderPersonal, true, KindReferral, true},
		{"enterprise direct refused", OrgTypeEnterprise, false, KindDirect, false},
		{"enterprise referral refused", OrgTypeEnterprise, true, KindReferral, false},
		{"unknown org type refused", "galaxy", true, KindDirect, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := IsKindAllowedForOrg(tc.orgType, tc.referrerEnabled, tc.kind)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---- NewPricing happy paths ----

func TestNewPricing_HappyPath_Daily(t *testing.T) {
	orgID := uuid.New()
	p, err := NewPricing(orgID, KindDirect, TypeDaily, 60000, nil, "EUR", "TJM standard")
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, orgID, p.OrganizationID)
	assert.Equal(t, KindDirect, p.Kind)
	assert.Equal(t, TypeDaily, p.Type)
	assert.Equal(t, int64(60000), p.MinAmount)
	assert.Nil(t, p.MaxAmount)
	assert.Equal(t, "EUR", p.Currency)
	assert.Equal(t, "TJM standard", p.Note)
	assert.True(t, p.CreatedAt.IsZero())
}

func TestNewPricing_HappyPath_ProjectRange(t *testing.T) {
	orgID := uuid.New()
	max := int64(500000)
	p, err := NewPricing(orgID, KindDirect, TypeProjectRange, 100000, &max, "EUR", "")
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, int64(100000), p.MinAmount)
	require.NotNil(t, p.MaxAmount)
	assert.Equal(t, int64(500000), *p.MaxAmount)
}

func TestNewPricing_HappyPath_CommissionPct(t *testing.T) {
	orgID := uuid.New()
	max := int64(1500) // 15%
	p, err := NewPricing(orgID, KindReferral, TypeCommissionPct, 500, &max, "pct", "apporteur")
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, "pct", p.Currency)
	assert.Equal(t, int64(500), p.MinAmount)
	require.NotNil(t, p.MaxAmount)
	assert.Equal(t, int64(1500), *p.MaxAmount)
}

func TestNewPricing_HappyPath_CommissionFlat(t *testing.T) {
	orgID := uuid.New()
	p, err := NewPricing(orgID, KindReferral, TypeCommissionFlat, 25000, nil, "EUR", "")
	require.NoError(t, err)
	require.NotNil(t, p)
	assert.Equal(t, TypeCommissionFlat, p.Type)
}

// ---- NewPricing failure branches ----

func TestNewPricing_Errors(t *testing.T) {
	orgID := uuid.New()
	maxBelowMin := int64(100)
	maxAboveMin := int64(10000)

	tests := []struct {
		name      string
		kind      PricingKind
		ptype     PricingType
		min       int64
		max       *int64
		currency  string
		wantErrIs error
	}{
		{
			name: "invalid kind", kind: "", ptype: TypeDaily, min: 1, currency: "EUR",
			wantErrIs: ErrInvalidKind,
		},
		{
			name: "invalid type", kind: KindDirect, ptype: "monthly", min: 1, currency: "EUR",
			wantErrIs: ErrInvalidType,
		},
		{
			name: "type not allowed for direct kind", kind: KindDirect, ptype: TypeCommissionPct, min: 100, max: &maxAboveMin, currency: "pct",
			wantErrIs: ErrTypeNotAllowedForKind,
		},
		{
			name: "type not allowed for referral kind", kind: KindReferral, ptype: TypeDaily, min: 100, currency: "EUR",
			wantErrIs: ErrTypeNotAllowedForKind,
		},
		{
			name: "negative min amount", kind: KindDirect, ptype: TypeDaily, min: -1, currency: "EUR",
			wantErrIs: ErrNegativeAmount,
		},
		{
			name: "max below min on range", kind: KindDirect, ptype: TypeProjectRange, min: 200, max: &maxBelowMin, currency: "EUR",
			wantErrIs: ErrMaxLessThanMin,
		},
		{
			name: "range not allowed for daily", kind: KindDirect, ptype: TypeDaily, min: 100, max: &maxAboveMin, currency: "EUR",
			wantErrIs: ErrRangeNotAllowedForType,
		},
		{
			name: "range not allowed for hourly", kind: KindDirect, ptype: TypeHourly, min: 50, max: &maxAboveMin, currency: "EUR",
			wantErrIs: ErrRangeNotAllowedForType,
		},
		{
			name: "range not allowed for project_from", kind: KindDirect, ptype: TypeProjectFrom, min: 100, max: &maxAboveMin, currency: "EUR",
			wantErrIs: ErrRangeNotAllowedForType,
		},
		{
			name: "range not allowed for commission_flat", kind: KindReferral, ptype: TypeCommissionFlat, min: 100, max: &maxAboveMin, currency: "EUR",
			wantErrIs: ErrRangeNotAllowedForType,
		},
		{
			name: "range required for project_range", kind: KindDirect, ptype: TypeProjectRange, min: 100, max: nil, currency: "EUR",
			wantErrIs: ErrRangeRequiredForType,
		},
		{
			name: "range required for commission_pct", kind: KindReferral, ptype: TypeCommissionPct, min: 100, max: nil, currency: "pct",
			wantErrIs: ErrRangeRequiredForType,
		},
		{
			name: "empty currency rejected", kind: KindDirect, ptype: TypeDaily, min: 1, currency: "",
			wantErrIs: ErrInvalidCurrency,
		},
		{
			name: "pct currency rejected for non-commission_pct", kind: KindReferral, ptype: TypeCommissionFlat, min: 1, currency: "pct",
			wantErrIs: ErrInvalidCurrencyForType,
		},
		{
			name: "non-pct currency rejected for commission_pct", kind: KindReferral, ptype: TypeCommissionPct, min: 100, max: &maxAboveMin, currency: "EUR",
			wantErrIs: ErrInvalidCurrencyForType,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPricing(orgID, tc.kind, tc.ptype, tc.min, tc.max, tc.currency, "")
			assert.ErrorIs(t, err, tc.wantErrIs)
		})
	}
}

// Max == min is accepted — clients often declare a "10% only" row as
// a degenerate range. The domain allows it rather than forcing the
// user to switch pricing types.
func TestNewPricing_MaxEqualsMinIsAccepted(t *testing.T) {
	orgID := uuid.New()
	m := int64(100)
	p, err := NewPricing(orgID, KindDirect, TypeProjectRange, 100, &m, "EUR", "")
	require.NoError(t, err)
	require.NotNil(t, p.MaxAmount)
	assert.Equal(t, int64(100), *p.MaxAmount)
}

// Zero min amount is accepted for every type — interpreted as "price
// on request" or "starting from 0".
func TestNewPricing_ZeroMinIsAccepted(t *testing.T) {
	orgID := uuid.New()
	p, err := NewPricing(orgID, KindDirect, TypeDaily, 0, nil, "EUR", "")
	require.NoError(t, err)
	assert.Equal(t, int64(0), p.MinAmount)
}
