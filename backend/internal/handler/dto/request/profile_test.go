package request

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestUpdateProfileRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(UpdateProfileRequest{
		Title: strings.Repeat("a", 201),
	}))
	require.Error(t, validator.Validate(UpdateProfileRequest{
		PhotoURL: "not a url",
	}))
	require.NoError(t, validator.Validate(UpdateProfileRequest{}))
}

func TestUpdateClientProfileRequest_Validation(t *testing.T) {
	tooLong := strings.Repeat("a", 201)
	require.Error(t, validator.Validate(UpdateClientProfileRequest{CompanyName: &tooLong}))
	require.NoError(t, validator.Validate(UpdateClientProfileRequest{}))
}
