package request

type UpdateProfileRequest struct {
	Title                string `json:"title"`
	PhotoURL             string `json:"photo_url"`
	PresentationVideoURL string `json:"presentation_video_url"`
	ReferrerVideoURL     string `json:"referrer_video_url"`
}
