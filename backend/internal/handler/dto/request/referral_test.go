package request

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestCreateReferralRequest_Validation(t *testing.T) {
	valid := CreateReferralRequest{
		ProviderID:     uuid.NewString(),
		ClientID:       uuid.NewString(),
		RatePct:        10,
		DurationMonths: 6,
	}
	require.NoError(t, validator.Validate(valid))

	t.Run("rate out of range", func(t *testing.T) {
		r := valid
		r.RatePct = 150
		require.Error(t, validator.Validate(r))
	})
	t.Run("invalid uuid", func(t *testing.T) {
		r := valid
		r.ProviderID = "not-uuid"
		require.Error(t, validator.Validate(r))
	})
	t.Run("duration too long", func(t *testing.T) {
		r := valid
		r.DurationMonths = 200
		require.Error(t, validator.Validate(r))
	})
}

func TestRespondReferralRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(RespondReferralRequest{Action: "delete"}))
	require.NoError(t, validator.Validate(RespondReferralRequest{Action: "accept"}))
}
