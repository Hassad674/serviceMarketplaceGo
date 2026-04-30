package request

import (
	"testing"

	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestUpdateOrganizationLocationRequest_Validation(t *testing.T) {
	bad := -91.0
	require.Error(t, validator.Validate(UpdateOrganizationLocationRequest{Latitude: &bad}))

	bad = 181.0
	require.Error(t, validator.Validate(UpdateOrganizationLocationRequest{Longitude: &bad}))

	require.Error(t, validator.Validate(UpdateOrganizationLocationRequest{CountryCode: "FRA"}))

	require.NoError(t, validator.Validate(UpdateOrganizationLocationRequest{}))
}

func TestUpdateOrganizationLanguagesRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(UpdateOrganizationLanguagesRequest{
		Professional: make([]string, 21),
	}))
	require.NoError(t, validator.Validate(UpdateOrganizationLanguagesRequest{
		Professional: []string{"en"},
	}))
}

func TestUpdateOrganizationPhotoRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(UpdateOrganizationPhotoRequest{PhotoURL: "not-a-url"}))
	require.NoError(t, validator.Validate(UpdateOrganizationPhotoRequest{}))
}
