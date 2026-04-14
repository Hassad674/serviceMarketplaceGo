package referrerpricing_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referrerpricing"
)

func intp(v int64) *int64 { return &v }

func TestPricingType_IsValid(t *testing.T) {
	cases := []struct {
		value referrerpricing.PricingType
		want  bool
	}{
		{referrerpricing.TypeCommissionPct, true},
		{referrerpricing.TypeCommissionFlat, true},
		{referrerpricing.PricingType("daily"), false},
		{referrerpricing.PricingType(""), false},
	}
	for _, tc := range cases {
		t.Run(string(tc.value), func(t *testing.T) {
			assert.Equal(t, tc.want, tc.value.IsValid())
		})
	}
}

func TestNewPricing_CommissionPctHappyPath(t *testing.T) {
	profileID := uuid.New()
	p, err := referrerpricing.NewPricing(referrerpricing.NewPricingInput{
		ProfileID: profileID,
		Type:      referrerpricing.TypeCommissionPct,
		MinAmount: 500,  // 5%
		MaxAmount: intp(2000), // 20%
		Currency:  "pct",
	})
	require.NoError(t, err)
	assert.Equal(t, profileID, p.ProfileID)
	assert.Equal(t, referrerpricing.TypeCommissionPct, p.Type)
	assert.Equal(t, int64(500), p.MinAmount)
	require.NotNil(t, p.MaxAmount)
	assert.Equal(t, int64(2000), *p.MaxAmount)
	assert.Equal(t, "pct", p.Currency)
}

func TestNewPricing_CommissionFlatHappyPath(t *testing.T) {
	profileID := uuid.New()
	p, err := referrerpricing.NewPricing(referrerpricing.NewPricingInput{
		ProfileID:  profileID,
		Type:       referrerpricing.TypeCommissionFlat,
		MinAmount:  50000,
		Currency:   "EUR",
		Note:       "per closed deal",
		Negotiable: true,
	})
	require.NoError(t, err)
	assert.Equal(t, referrerpricing.TypeCommissionFlat, p.Type)
	assert.Equal(t, int64(50000), p.MinAmount)
	assert.Nil(t, p.MaxAmount)
	assert.Equal(t, "EUR", p.Currency)
	assert.Equal(t, "per closed deal", p.Note)
	assert.True(t, p.Negotiable)
}

func TestNewPricing_Validation(t *testing.T) {
	profileID := uuid.New()

	tests := []struct {
		name    string
		in      referrerpricing.NewPricingInput
		wantErr error
	}{
		{
			name: "invalid type — daily belongs to freelance",
			in: referrerpricing.NewPricingInput{
				ProfileID: profileID,
				Type:      referrerpricing.PricingType("daily"),
				MinAmount: 100,
				Currency:  "EUR",
			},
			wantErr: referrerpricing.ErrInvalidType,
		},
		{
			name: "empty type",
			in: referrerpricing.NewPricingInput{
				ProfileID: profileID,
				MinAmount: 100,
				Currency:  "EUR",
			},
			wantErr: referrerpricing.ErrInvalidType,
		},
		{
			name: "negative amount",
			in: referrerpricing.NewPricingInput{
				ProfileID: profileID,
				Type:      referrerpricing.TypeCommissionFlat,
				MinAmount: -1,
				Currency:  "EUR",
			},
			wantErr: referrerpricing.ErrNegativeAmount,
		},
		{
			name: "commission_pct without max",
			in: referrerpricing.NewPricingInput{
				ProfileID: profileID,
				Type:      referrerpricing.TypeCommissionPct,
				MinAmount: 500,
				Currency:  "pct",
			},
			wantErr: referrerpricing.ErrRangeRequiredForType,
		},
		{
			name: "commission_pct max less than min",
			in: referrerpricing.NewPricingInput{
				ProfileID: profileID,
				Type:      referrerpricing.TypeCommissionPct,
				MinAmount: 500,
				MaxAmount: intp(300),
				Currency:  "pct",
			},
			wantErr: referrerpricing.ErrMaxLessThanMin,
		},
		{
			name: "commission_flat with max",
			in: referrerpricing.NewPricingInput{
				ProfileID: profileID,
				Type:      referrerpricing.TypeCommissionFlat,
				MinAmount: 100,
				MaxAmount: intp(200),
				Currency:  "EUR",
			},
			wantErr: referrerpricing.ErrRangeNotAllowedForType,
		},
		{
			name: "commission_pct max out of range (>100%)",
			in: referrerpricing.NewPricingInput{
				ProfileID: profileID,
				Type:      referrerpricing.TypeCommissionPct,
				MinAmount: 500,
				MaxAmount: intp(10001),
				Currency:  "pct",
			},
			wantErr: referrerpricing.ErrCommissionPctOutOfRange,
		},
		{
			name: "empty currency",
			in: referrerpricing.NewPricingInput{
				ProfileID: profileID,
				Type:      referrerpricing.TypeCommissionFlat,
				MinAmount: 100,
				Currency:  "",
			},
			wantErr: referrerpricing.ErrInvalidCurrency,
		},
		{
			name: "commission_pct with EUR currency",
			in: referrerpricing.NewPricingInput{
				ProfileID: profileID,
				Type:      referrerpricing.TypeCommissionPct,
				MinAmount: 500,
				MaxAmount: intp(2000),
				Currency:  "EUR",
			},
			wantErr: referrerpricing.ErrInvalidCurrencyForType,
		},
		{
			name: "commission_flat with pct currency",
			in: referrerpricing.NewPricingInput{
				ProfileID: profileID,
				Type:      referrerpricing.TypeCommissionFlat,
				MinAmount: 100,
				Currency:  "pct",
			},
			wantErr: referrerpricing.ErrInvalidCurrencyForType,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := referrerpricing.NewPricing(tc.in)
			assert.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, p)
		})
	}
}
