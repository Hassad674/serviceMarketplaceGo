package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	referrerprofileapp "marketplace-backend/internal/app/referrerprofile"
	"marketplace-backend/internal/domain/profile"
	domainreferrerpricing "marketplace-backend/internal/domain/referrerpricing"
	"marketplace-backend/internal/domain/referrerprofile"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// ReferrerPricingReader is the minimal read contract the referrer
// profile handler needs to decorate its response with the
// persona-specific pricing row. Defined locally so the handler
// does not carry a direct dependency on the pricing app package.
type ReferrerPricingReader interface {
	Get(ctx context.Context, profileID uuid.UUID) (*domainreferrerpricing.Pricing, error)
}

// ReferrerProfileHandler wires the referrer profile HTTP endpoints
// to the referrer profile app service. Pricing is an optional
// collaborator wired via a fluent builder after construction.
// Unlike the freelance handler, there is no skills decoration —
// skills describe what a person does themselves, not the deals
// they bring in as an apporteur.
type ReferrerProfileHandler struct {
	svc           *referrerprofileapp.Service
	pricingReader ReferrerPricingReader
}

// NewReferrerProfileHandler constructs the handler with the
// referrer profile service.
func NewReferrerProfileHandler(svc *referrerprofileapp.Service) *ReferrerProfileHandler {
	return &ReferrerProfileHandler{svc: svc}
}

// WithPricingReader attaches the pricing decorator. Nil is a no-op.
func (h *ReferrerProfileHandler) WithPricingReader(reader ReferrerPricingReader) *ReferrerProfileHandler {
	if reader != nil {
		h.pricingReader = reader
	}
	return h
}

// GetMy returns the referrer profile of the authenticated user's
// current organization. The service auto-creates a default row
// on first access so providers who just toggled referrer_enabled
// see a clean blank profile instead of a 404.
func (h *ReferrerProfileHandler) GetMy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	view, err := h.svc.GetByOrgID(r.Context(), orgID)
	if err != nil {
		handleReferrerProfileError(w, err)
		return
	}

	h.writeViewResponse(w, r, view)
}

// UpdateMy writes the title / about / video_url triplet.
func (h *ReferrerProfileHandler) UpdateMy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req request.UpdateReferrerProfileRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	view, err := h.svc.UpdateCore(r.Context(), orgID, referrerprofileapp.UpdateCoreInput{
		Title:    req.Title,
		About:    req.About,
		VideoURL: req.VideoURL,
	})
	if err != nil {
		handleReferrerProfileError(w, err)
		return
	}

	h.writeViewResponse(w, r, view)
}

// UpdateMyAvailability writes a single availability value.
func (h *ReferrerProfileHandler) UpdateMyAvailability(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req request.UpdateReferrerAvailabilityRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	view, err := h.svc.UpdateAvailability(r.Context(), orgID, req.AvailabilityStatus)
	if err != nil {
		handleReferrerProfileError(w, err)
		return
	}

	h.writeViewResponse(w, r, view)
}

// UpdateMyExpertise replaces the expertise list atomically.
func (h *ReferrerProfileHandler) UpdateMyExpertise(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req request.UpdateReferrerExpertiseRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	const maxRequestSize = 20
	if len(req.Domains) > maxRequestSize {
		res.Error(w, http.StatusBadRequest, "validation_error", "too many domains in request")
		return
	}

	view, err := h.svc.UpdateExpertise(r.Context(), orgID, req.Domains)
	if err != nil {
		handleReferrerProfileError(w, err)
		return
	}

	h.writeViewResponse(w, r, view)
}

// GetPublic returns any organization's referrer profile. Route
// param is the organization id, preserving the existing public
// URL scheme the frontend uses (/referrers/[orgID]).
func (h *ReferrerProfileHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	orgIDParam := chi.URLParam(r, "orgID")
	orgID, err := uuid.Parse(orgIDParam)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "organization ID must be a valid UUID")
		return
	}

	view, err := h.svc.GetByOrgID(r.Context(), orgID)
	if err != nil {
		handleReferrerProfileError(w, err)
		return
	}

	h.writeViewResponse(w, r, view)
}

// writeViewResponse decorates and writes the full DTO.
func (h *ReferrerProfileHandler) writeViewResponse(w http.ResponseWriter, r *http.Request, view *repository.ReferrerProfileView) {
	pricing := h.loadPricing(r, view.Profile.ID)
	res.JSON(w, http.StatusOK, response.NewReferrerProfileResponse(view, pricing))
}

// loadPricing fetches the profile pricing via the optional
// ReferrerPricingReader. Nil reader or any error yields nil.
func (h *ReferrerProfileHandler) loadPricing(r *http.Request, profileID uuid.UUID) *domainreferrerpricing.Pricing {
	if h.pricingReader == nil {
		return nil
	}
	p, err := h.pricingReader.Get(r.Context(), profileID)
	if err != nil {
		return nil
	}
	return p
}

// handleReferrerProfileError maps domain-level errors to stable
// HTTP status codes.
func handleReferrerProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, referrerprofile.ErrProfileNotFound):
		res.Error(w, http.StatusNotFound, "referrer_profile_not_found", err.Error())
	case errors.Is(err, profile.ErrInvalidAvailabilityStatus):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
