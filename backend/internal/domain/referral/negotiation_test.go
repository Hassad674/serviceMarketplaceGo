package referral_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/referral"
)

func validNegotiationInput() referral.NewNegotiationInput {
	return referral.NewNegotiationInput{
		ReferralID: uuid.New(),
		Version:    1,
		ActorID:    uuid.New(),
		ActorRole:  referral.ActorReferrer,
		Action:     referral.NegoActionProposed,
		RatePct:    5,
		Message:    "ma proposition initiale",
	}
}

func TestNewNegotiation_Valid(t *testing.T) {
	in := validNegotiationInput()
	n, err := referral.NewNegotiation(in)
	require.NoError(t, err)
	require.NotNil(t, n)
	assert.NotEqual(t, uuid.Nil, n.ID)
	assert.Equal(t, in.ReferralID, n.ReferralID)
	assert.Equal(t, in.RatePct, n.RatePct)
	assert.False(t, n.CreatedAt.IsZero())
}

func TestNewNegotiation_Validation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*referral.NewNegotiationInput)
		want   error
	}{
		{"nil referral id", func(in *referral.NewNegotiationInput) { in.ReferralID = uuid.Nil }, referral.ErrNotAuthorized},
		{"nil actor id", func(in *referral.NewNegotiationInput) { in.ActorID = uuid.Nil }, referral.ErrNotAuthorized},
		{"invalid actor role", func(in *referral.NewNegotiationInput) { in.ActorRole = "admin" }, referral.ErrNotAuthorized},
		{"invalid action", func(in *referral.NewNegotiationInput) { in.Action = "merged" }, referral.ErrInvalidTransition},
		{"version below one", func(in *referral.NewNegotiationInput) { in.Version = 0 }, referral.ErrInvalidTransition},
		{"rate negative", func(in *referral.NewNegotiationInput) { in.RatePct = -1 }, referral.ErrRateOutOfRange},
		{"rate too high", func(in *referral.NewNegotiationInput) { in.RatePct = 75 }, referral.ErrRateOutOfRange},
		{"message too long", func(in *referral.NewNegotiationInput) { in.Message = strings.Repeat("m", 5000) }, referral.ErrMessageTooLong},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validNegotiationInput()
			tt.mutate(&in)
			n, err := referral.NewNegotiation(in)
			require.ErrorIs(t, err, tt.want)
			assert.Nil(t, n)
		})
	}
}

func TestNegotiationAction_IsValid(t *testing.T) {
	assert.True(t, referral.NegoActionProposed.IsValid())
	assert.True(t, referral.NegoActionCountered.IsValid())
	assert.True(t, referral.NegoActionAccepted.IsValid())
	assert.True(t, referral.NegoActionRejected.IsValid())
	assert.False(t, referral.NegotiationAction("ghosted").IsValid())
}
