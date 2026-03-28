package request

type UpdateNotificationPreferencesRequest struct {
	Preferences []NotificationPreferenceItem `json:"preferences"`
}

type NotificationPreferenceItem struct {
	Type  string `json:"type"`
	InApp bool   `json:"in_app"`
	Push  bool   `json:"push"`
	Email bool   `json:"email"`
}

type RegisterDeviceTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}
