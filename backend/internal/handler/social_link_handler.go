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

// SocialLinkHandler handles HTTP requests for social link CRUD.
type SocialLinkHandler struct {
	socialLinkSvc *profileapp.SocialLinkService
}

// NewSocialLinkHandler creates a new handler for social link endpoints.
func NewSocialLinkHandler(svc *profileapp.SocialLinkService) *SocialLinkHandler {
	return &SocialLinkHandler{socialLinkSvc: svc}
}

// ListMySocialLinks returns social links for the authenticated user.
func (h *SocialLinkHandler) ListMySocialLinks(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	links, err := h.socialLinkSvc.ListByUser(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}

	res.JSON(w, http.StatusOK, response.NewSocialLinkListResponse(links))
}

// UpsertSocialLink creates or updates a social link for the authenticated user.
func (h *SocialLinkHandler) UpsertSocialLink(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req struct {
		Platform string `json:"platform"`
		URL      string `json:"url"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input := profileapp.UpsertInput{Platform: req.Platform, URL: req.URL}
	if err := h.socialLinkSvc.Upsert(r.Context(), userID, input); err != nil {
		handleSocialLinkError(w, err)
		return
	}

	res.NoContent(w)
}

// DeleteSocialLink removes a social link for the authenticated user.
func (h *SocialLinkHandler) DeleteSocialLink(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	platform := chi.URLParam(r, "platform")
	if err := h.socialLinkSvc.Delete(r.Context(), userID, platform); err != nil {
		handleSocialLinkError(w, err)
		return
	}

	res.NoContent(w)
}

// ListPublicSocialLinks returns social links for any user (public).
func (h *SocialLinkHandler) ListPublicSocialLinks(w http.ResponseWriter, r *http.Request) {
	userIDParam := chi.URLParam(r, "userId")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_user_id", "user ID must be a valid UUID")
		return
	}

	links, err := h.socialLinkSvc.ListByUser(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}

	res.JSON(w, http.StatusOK, response.NewSocialLinkListResponse(links))
}

func handleSocialLinkError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, profile.ErrInvalidPlatform):
		res.Error(w, http.StatusBadRequest, "invalid_platform", err.Error())
	case errors.Is(err, profile.ErrInvalidURL):
		res.Error(w, http.StatusBadRequest, "invalid_url", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
