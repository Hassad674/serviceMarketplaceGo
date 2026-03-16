package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	profileapp "marketplace-backend/internal/app/profile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

type ProfileHandler struct {
	profileService *profileapp.Service
}

func NewProfileHandler(profileService *profileapp.Service) *ProfileHandler {
	return &ProfileHandler{profileService: profileService}
}

func (h *ProfileHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	p, err := h.profileService.GetProfile(r.Context(), userID)
	if err != nil {
		handleProfileError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewProfileResponse(p))
}

func (h *ProfileHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req struct {
		Title                string `json:"title"`
		PhotoURL             string `json:"photo_url"`
		PresentationVideoURL string `json:"presentation_video_url"`
		ReferrerVideoURL     string `json:"referrer_video_url"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input := profileapp.UpdateProfileInput{
		Title:                req.Title,
		PhotoURL:             req.PhotoURL,
		PresentationVideoURL: req.PresentationVideoURL,
		ReferrerVideoURL:     req.ReferrerVideoURL,
	}

	p, err := h.profileService.UpdateProfile(r.Context(), userID, input)
	if err != nil {
		handleProfileError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewProfileResponse(p))
}

func (h *ProfileHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	userIDParam := chi.URLParam(r, "userId")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_user_id", "user ID must be a valid UUID")
		return
	}

	p, err := h.profileService.GetProfile(r.Context(), userID)
	if err != nil {
		handleProfileError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewProfileResponse(p))
}

func handleProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, profile.ErrProfileNotFound):
		res.Error(w, http.StatusNotFound, "profile_not_found", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
