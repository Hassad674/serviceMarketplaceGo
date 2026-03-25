package request

type UpdateProfileRequest struct {
	Title                string `json:"title"`
	About                string `json:"about"`
	PhotoURL             string `json:"photo_url"`
	PresentationVideoURL string `json:"presentation_video_url"`
	ReferrerAbout        string `json:"referrer_about"`
	ReferrerVideoURL     string `json:"referrer_video_url"`
}
