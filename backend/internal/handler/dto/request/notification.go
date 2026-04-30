package request

// UpdateNotificationPreferencesRequest is the body of
// PUT /api/v1/notifications/preferences.
type UpdateNotificationPreferencesRequest struct {
	Preferences []NotificationPreferenceItem `json:"preferences" validate:"required,min=0,max=200,dive"`
}

type NotificationPreferenceItem struct {
	Type  string `json:"type" validate:"required,min=1,max=100"`
	InApp bool   `json:"in_app"`
	Push  bool   `json:"push"`
	Email bool   `json:"email"`
}

// BulkEmailPreferencesRequest toggles all email notifications at once.
type BulkEmailPreferencesRequest struct {
	Enabled bool `json:"enabled"`
}

// RegisterDeviceTokenRequest is the body of POST /api/v1/notifications/devices.
type RegisterDeviceTokenRequest struct {
	Token    string `json:"token" validate:"required,min=1,max=4096"`
	Platform string `json:"platform" validate:"required,oneof=ios android web"`
}
