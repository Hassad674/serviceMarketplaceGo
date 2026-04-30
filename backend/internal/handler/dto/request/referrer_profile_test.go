package request

import (
	"testing"

	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestUpdateReferrerProfileRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(UpdateReferrerProfileRequest{VideoURL: "not-a-url"}))
	require.NoError(t, validator.Validate(UpdateReferrerProfileRequest{}))
}

func TestUpsertReferrerPricingRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(UpsertReferrerPricingRequest{Type: "", Currency: "EUR"}))
	require.Error(t, validator.Validate(UpsertReferrerPricingRequest{Type: "fixed", Currency: "INVALID"}))
	require.Error(t, validator.Validate(UpsertReferrerPricingRequest{Type: "fixed", Currency: "EUR", MinAmount: -1}))
	require.NoError(t, validator.Validate(UpsertReferrerPricingRequest{Type: "fixed", Currency: "EUR"}))
}

func TestUpdateReferrerAvailabilityRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(UpdateReferrerAvailabilityRequest{AvailabilityStatus: ""}))
}
