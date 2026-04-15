package referral_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
)

func validAttributionInput() referral.NewAttributionInput {
	return referral.NewAttributionInput{
		ReferralID:      uuid.New(),
		ProposalID:      uuid.New(),
		ProviderID:      uuid.New(),
		ClientID:        uuid.New(),
		RatePctSnapshot: 5,
	}
}

func TestNewAttribution_Valid(t *testing.T) {
	in := validAttributionInput()
	a, err := referral.NewAttribution(in)
	require.NoError(t, err)
	require.NotNil(t, a)
	assert.NotEqual(t, uuid.Nil, a.ID)
	assert.Equal(t, in.ReferralID, a.ReferralID)
	assert.Equal(t, in.ProposalID, a.ProposalID)
	assert.Equal(t, in.RatePctSnapshot, a.RatePctSnapshot)
	assert.False(t, a.AttributedAt.IsZero())
}

func TestNewAttribution_Validation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*referral.NewAttributionInput)
		want   error
	}{
		{"nil referral id", func(in *referral.NewAttributionInput) { in.ReferralID = uuid.Nil }, referral.ErrNotAuthorized},
		{"nil proposal id", func(in *referral.NewAttributionInput) { in.ProposalID = uuid.Nil }, referral.ErrNotAuthorized},
		{"nil provider id", func(in *referral.NewAttributionInput) { in.ProviderID = uuid.Nil }, referral.ErrNotAuthorized},
		{"nil client id", func(in *referral.NewAttributionInput) { in.ClientID = uuid.Nil }, referral.ErrNotAuthorized},
		{
			name: "provider equals client",
			mutate: func(in *referral.NewAttributionInput) {
				in.ClientID = in.ProviderID
			},
			want: referral.ErrSelfReferral,
		},
		{"rate negative", func(in *referral.NewAttributionInput) { in.RatePctSnapshot = -1 }, referral.ErrRateOutOfRange},
		{"rate too high", func(in *referral.NewAttributionInput) { in.RatePctSnapshot = 60 }, referral.ErrRateOutOfRange},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validAttributionInput()
			tt.mutate(&in)
			a, err := referral.NewAttribution(in)
			require.ErrorIs(t, err, tt.want)
			assert.Nil(t, a)
		})
	}
}
