package request

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestUpdateFreelanceProfileRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(UpdateFreelanceProfileRequest{
		Title: strings.Repeat("a", 201),
	}))
	require.Error(t, validator.Validate(UpdateFreelanceProfileRequest{
		VideoURL: "not a url",
	}))
	require.NoError(t, validator.Validate(UpdateFreelanceProfileRequest{}))
}

func TestUpdateFreelanceAvailabilityRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(UpdateFreelanceAvailabilityRequest{AvailabilityStatus: ""}))
	require.NoError(t, validator.Validate(UpdateFreelanceAvailabilityRequest{AvailabilityStatus: "active"}))
}

func TestUpsertFreelancePricingRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(UpsertFreelancePricingRequest{Type: "", Currency: "EUR"}))
	require.Error(t, validator.Validate(UpsertFreelancePricingRequest{Type: "fixed", Currency: "INVALID"}))
	require.Error(t, validator.Validate(UpsertFreelancePricingRequest{Type: "fixed", Currency: "EUR", MinAmount: -1}))
	require.NoError(t, validator.Validate(UpsertFreelancePricingRequest{Type: "fixed", Currency: "EUR"}))
}
