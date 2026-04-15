package referral_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
)

func TestIntroSnapshot_Validate_Empty(t *testing.T) {
	// An empty snapshot should be valid (the apporteur may decline to reveal
	// any auto-prefilled fields).
	s := referral.IntroSnapshot{}
	require.NoError(t, s.Validate())
}

func TestIntroSnapshot_Validate_HappyPath(t *testing.T) {
	years := 8
	rating := 4.7
	count := 34
	min := int64(60_000)
	max := int64(85_000)

	s := referral.IntroSnapshot{
		Provider: referral.ProviderSnapshot{
			ExpertiseDomains:  []string{"branding", "ui"},
			YearsExperience:   &years,
			AverageRating:     &rating,
			ReviewCount:       &count,
			PricingMinCents:   &min,
			PricingMaxCents:   &max,
			PricingCurrency:   "EUR",
			PricingType:       "daily",
			Region:            "Île-de-France",
			Languages:         []string{"fr", "en"},
			AvailabilityState: "available",
		},
		Client: referral.ClientSnapshot{
			Industry:    "Mode",
			SizeBucket:  "pme",
			Region:      "IDF",
			NeedSummary: "Refonte branding",
		},
	}
	require.NoError(t, s.Validate())
}

func TestProviderSnapshot_Validation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*referral.ProviderSnapshot)
		wantErr bool
	}{
		{
			name: "negative years",
			mutate: func(p *referral.ProviderSnapshot) {
				bad := -1
				p.YearsExperience = &bad
			},
			wantErr: true,
		},
		{
			name: "years too high",
			mutate: func(p *referral.ProviderSnapshot) {
				bad := 101
				p.YearsExperience = &bad
			},
			wantErr: true,
		},
		{
			name: "rating below zero",
			mutate: func(p *referral.ProviderSnapshot) {
				bad := -0.1
				p.AverageRating = &bad
			},
			wantErr: true,
		},
		{
			name: "rating above five",
			mutate: func(p *referral.ProviderSnapshot) {
				bad := 5.5
				p.AverageRating = &bad
			},
			wantErr: true,
		},
		{
			name: "negative review count",
			mutate: func(p *referral.ProviderSnapshot) {
				bad := -3
				p.ReviewCount = &bad
			},
			wantErr: true,
		},
		{
			name: "max < min pricing",
			mutate: func(p *referral.ProviderSnapshot) {
				min := int64(10_000)
				max := int64(5_000)
				p.PricingMinCents = &min
				p.PricingMaxCents = &max
			},
			wantErr: true,
		},
		{
			name: "negative pricing",
			mutate: func(p *referral.ProviderSnapshot) {
				bad := int64(-1)
				p.PricingMinCents = &bad
			},
			wantErr: true,
		},
		{
			name: "region too long",
			mutate: func(p *referral.ProviderSnapshot) {
				p.Region = strings.Repeat("a", 300)
			},
			wantErr: true,
		},
		{
			name: "too many expertise domains",
			mutate: func(p *referral.ProviderSnapshot) {
				p.ExpertiseDomains = make([]string, 30)
				for i := range p.ExpertiseDomains {
					p.ExpertiseDomains[i] = "x"
				}
			},
			wantErr: true,
		},
		{
			name: "expertise item too long",
			mutate: func(p *referral.ProviderSnapshot) {
				p.ExpertiseDomains = []string{strings.Repeat("a", 100)}
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := referral.IntroSnapshot{}
			tt.mutate(&s.Provider)
			err := s.Validate()
			if tt.wantErr {
				require.ErrorIs(t, err, referral.ErrSnapshotInvalid)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestClientSnapshot_Validation(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*referral.ClientSnapshot)
		wantErr bool
	}{
		{
			name:    "industry too long",
			mutate:  func(c *referral.ClientSnapshot) { c.Industry = strings.Repeat("z", 300) },
			wantErr: true,
		},
		{
			name:    "need summary too long",
			mutate:  func(c *referral.ClientSnapshot) { c.NeedSummary = strings.Repeat("y", 1000) },
			wantErr: true,
		},
		{
			name: "max < min budget",
			mutate: func(c *referral.ClientSnapshot) {
				min := int64(20_000)
				max := int64(10_000)
				c.BudgetEstimateMin = &min
				c.BudgetEstimateMax = &max
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := referral.IntroSnapshot{}
			tt.mutate(&s.Client)
			err := s.Validate()
			if tt.wantErr {
				require.ErrorIs(t, err, referral.ErrSnapshotInvalid)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSnapshot_RoundTrip(t *testing.T) {
	years := 5
	original := referral.IntroSnapshot{
		Provider: referral.ProviderSnapshot{
			ExpertiseDomains: []string{"web", "mobile"},
			YearsExperience:  &years,
			Region:           "IDF",
		},
		Client: referral.ClientSnapshot{
			Industry:   "Tech",
			SizeBucket: "tpe",
		},
	}
	raw, err := referral.MarshalSnapshot(original)
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	decoded, err := referral.UnmarshalSnapshot(raw)
	require.NoError(t, err)
	assert.Equal(t, original, decoded)
}

func TestUnmarshalSnapshot_EmptyBytes(t *testing.T) {
	s, err := referral.UnmarshalSnapshot(nil)
	require.NoError(t, err)
	assert.Equal(t, referral.IntroSnapshot{}, s)
}
