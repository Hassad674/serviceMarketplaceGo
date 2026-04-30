package request

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"marketplace-backend/pkg/validator"
)

func TestRegisterDeviceTokenRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(RegisterDeviceTokenRequest{Token: "", Platform: "ios"}))
	require.Error(t, validator.Validate(RegisterDeviceTokenRequest{Token: "abc", Platform: "windows"}))
	require.Error(t, validator.Validate(RegisterDeviceTokenRequest{Token: strings.Repeat("a", 5000), Platform: "ios"}))
	require.NoError(t, validator.Validate(RegisterDeviceTokenRequest{Token: "abc", Platform: "ios"}))
}

func TestUpdateNotificationPreferencesRequest_Validation(t *testing.T) {
	require.Error(t, validator.Validate(UpdateNotificationPreferencesRequest{
		Preferences: []NotificationPreferenceItem{{Type: ""}},
	}))
	require.NoError(t, validator.Validate(UpdateNotificationPreferencesRequest{
		Preferences: []NotificationPreferenceItem{{Type: "x", InApp: true}},
	}))
}
