package freelancepricing_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/freelancepricing"
)

func intp(v int64) *int64 { return &v }

func TestPricingType_IsValid(t *testing.T) {
	cases := []struct {
		value freelancepricing.PricingType
		want  bool
	}{
		{freelancepricing.TypeDaily, true},
		{freelancepricing.TypeHourly, true},
		{freelancepricing.TypeProjectFrom, true},
		{freelancepricing.TypeProjectRange, true},
		{freelancepricing.PricingType("commission_pct"), false},
		{freelancepricing.PricingType(""), false},
		{freelancepricing.PricingType("unknown"), false},
	}
	for _, tc := range cases {
		t.Run(string(tc.value), func(t *testing.T) {
			assert.Equal(t, tc.want, tc.value.IsValid())
		})
	}
}

func TestNewPricing_HappyPaths(t *testing.T) {
	profileID := uuid.New()

	tests := []struct {
		name string
		in   freelancepricing.NewPricingInput
	}{
		{
			name: "daily scalar",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.TypeDaily,
				MinAmount: 60000,
				Currency:  "EUR",
			},
		},
		{
			name: "hourly scalar",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.TypeHourly,
				MinAmount: 8000,
				Currency:  "USD",
			},
		},
		{
			name: "project_from scalar",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.TypeProjectFrom,
				MinAmount: 500000,
				Currency:  "EUR",
			},
		},
		{
			name: "project_range with max",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.TypeProjectRange,
				MinAmount: 500000,
				MaxAmount: intp(1500000),
				Currency:  "EUR",
				Note:      "minimum 3 month engagement",
			},
		},
		{
			name: "negotiable flag is stored",
			in: freelancepricing.NewPricingInput{
				ProfileID:  profileID,
				Type:       freelancepricing.TypeDaily,
				MinAmount:  60000,
				Currency:   "EUR",
				Negotiable: true,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := freelancepricing.NewPricing(tc.in)
			require.NoError(t, err)
			require.NotNil(t, p)
			assert.Equal(t, tc.in.ProfileID, p.ProfileID)
			assert.Equal(t, tc.in.Type, p.Type)
			assert.Equal(t, tc.in.MinAmount, p.MinAmount)
			assert.Equal(t, tc.in.Currency, p.Currency)
			assert.Equal(t, tc.in.Note, p.Note)
			assert.Equal(t, tc.in.Negotiable, p.Negotiable)
			if tc.in.MaxAmount != nil {
				require.NotNil(t, p.MaxAmount)
				assert.Equal(t, *tc.in.MaxAmount, *p.MaxAmount)
			}
		})
	}
}

func TestNewPricing_Validation(t *testing.T) {
	profileID := uuid.New()

	tests := []struct {
		name    string
		in      freelancepricing.NewPricingInput
		wantErr error
	}{
		{
			name: "invalid type — commission_pct belongs to referrer",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.PricingType("commission_pct"),
				MinAmount: 100,
				Currency:  "EUR",
			},
			wantErr: freelancepricing.ErrInvalidType,
		},
		{
			name: "empty type",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				MinAmount: 100,
				Currency:  "EUR",
			},
			wantErr: freelancepricing.ErrInvalidType,
		},
		{
			name: "negative amount",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.TypeDaily,
				MinAmount: -1,
				Currency:  "EUR",
			},
			wantErr: freelancepricing.ErrNegativeAmount,
		},
		{
			name: "range with max < min",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.TypeProjectRange,
				MinAmount: 500,
				MaxAmount: intp(100),
				Currency:  "EUR",
			},
			wantErr: freelancepricing.ErrMaxLessThanMin,
		},
		{
			name: "range on a scalar type",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.TypeDaily,
				MinAmount: 100,
				MaxAmount: intp(200),
				Currency:  "EUR",
			},
			wantErr: freelancepricing.ErrRangeNotAllowedForType,
		},
		{
			name: "range type without max",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.TypeProjectRange,
				MinAmount: 100,
				Currency:  "EUR",
			},
			wantErr: freelancepricing.ErrRangeRequiredForType,
		},
		{
			name: "empty currency",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.TypeDaily,
				MinAmount: 100,
				Currency:  "",
			},
			wantErr: freelancepricing.ErrInvalidCurrency,
		},
		{
			name: "pct currency rejected on freelance side",
			in: freelancepricing.NewPricingInput{
				ProfileID: profileID,
				Type:      freelancepricing.TypeDaily,
				MinAmount: 100,
				Currency:  "pct",
			},
			wantErr: freelancepricing.ErrInvalidCurrencyForType,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p, err := freelancepricing.NewPricing(tc.in)
			assert.ErrorIs(t, err, tc.wantErr)
			assert.Nil(t, p)
		})
	}
}
