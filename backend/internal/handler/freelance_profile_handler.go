package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	freelanceprofileapp "marketplace-backend/internal/app/freelanceprofile"
	domainfreelancepricing "marketplace-backend/internal/domain/freelancepricing"
	"marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// FreelancePricingReader is the minimal read contract the freelance
// profile handler needs to decorate its response with the
// persona-specific pricing row. Defined locally (not in port/) so
// the handler does not force a direct dependency on the pricing
// app package on every caller — cmd/api/main.go supplies a concrete
// value that matches the shape.
//
// A nil reader is tolerated by the handler: GetMy / UpdateMy /
// GetPublic simply render the response with Pricing=nil.
type FreelancePricingReader interface {
	Get(ctx context.Context, profileID uuid.UUID) (*domainfreelancepricing.Pricing, error)
}

// FreelanceProfileHandler wires the freelance profile HTTP
// endpoints to the freelance profile app service. Skills and
// pricing are optional collaborators wired via fluent builders
// after construction — passing nil is safe and yields empty
// decorations.
type FreelanceProfileHandler struct {
	svc           *freelanceprofileapp.Service
	skillsReader  SkillsReader
	pricingReader FreelancePricingReader
}

// NewFreelanceProfileHandler constructs the handler with the
// freelance profile service.
func NewFreelanceProfileHandler(svc *freelanceprofileapp.Service) *FreelanceProfileHandler {
	return &FreelanceProfileHandler{svc: svc}
}

// WithSkillsReader attaches the skills decorator. Nil is a no-op.
func (h *FreelanceProfileHandler) WithSkillsReader(reader SkillsReader) *FreelanceProfileHandler {
	if reader != nil {
		h.skillsReader = reader
	}
	return h
}

// WithPricingReader attaches the pricing decorator. Nil is a no-op.
func (h *FreelanceProfileHandler) WithPricingReader(reader FreelancePricingReader) *FreelanceProfileHandler {
	if reader != nil {
		h.pricingReader = reader
	}
	return h
}

// GetMy returns the freelance profile of the authenticated user's
// current organization. Decorated with the per-org skill list and
// the declared pricing row.
func (h *FreelanceProfileHandler) GetMy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	view, err := h.svc.GetByOrgID(r.Context(), orgID)
	if err != nil {
		handleFreelanceProfileError(w, err)
		return
	}

	h.writeViewResponse(w, r, view)
}

// UpdateMy writes the title / about / video_url triplet and
// returns the refreshed response.
func (h *FreelanceProfileHandler) UpdateMy(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req request.UpdateFreelanceProfileRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	view, err := h.svc.UpdateCore(r.Context(), orgID, freelanceprofileapp.UpdateCoreInput{
		Title:    req.Title,
		About:    req.About,
		VideoURL: req.VideoURL,
	})
	if err != nil {
		handleFreelanceProfileError(w, err)
		return
	}

	h.writeViewResponse(w, r, view)
}

// UpdateMyAvailability writes a single availability value.
func (h *FreelanceProfileHandler) UpdateMyAvailability(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req request.UpdateFreelanceAvailabilityRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	view, err := h.svc.UpdateAvailability(r.Context(), orgID, req.AvailabilityStatus)
	if err != nil {
		handleFreelanceProfileError(w, err)
		return
	}

	h.writeViewResponse(w, r, view)
}

// UpdateMyExpertise replaces the expertise list atomically.
func (h *FreelanceProfileHandler) UpdateMyExpertise(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req request.UpdateFreelanceExpertiseRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	// Hard upper bound on request size — obviously bogus payloads
	// rejected before any allocation.
	const maxRequestSize = 20
	if len(req.Domains) > maxRequestSize {
		res.Error(w, http.StatusBadRequest, "validation_error", "too many domains in request")
		return
	}

	view, err := h.svc.UpdateExpertise(r.Context(), orgID, req.Domains)
	if err != nil {
		handleFreelanceProfileError(w, err)
		return
	}

	h.writeViewResponse(w, r, view)
}

// GetPublic returns any organization's freelance profile. Route
// param is the organization id, preserving the existing public URL
// scheme the frontend uses (/freelancers/[orgID]).
func (h *FreelanceProfileHandler) GetPublic(w http.ResponseWriter, r *http.Request) {
	orgIDParam := chi.URLParam(r, "orgID")
	orgID, err := uuid.Parse(orgIDParam)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "organization ID must be a valid UUID")
		return
	}

	view, err := h.svc.GetByOrgID(r.Context(), orgID)
	if err != nil {
		handleFreelanceProfileError(w, err)
		return
	}

	h.writeViewResponse(w, r, view)
}

// writeViewResponse performs the skill + pricing decoration and
// writes the full DTO. Extracted so each endpoint stays under the
// 50-line cap and the decoration logic is tested in one place.
func (h *FreelanceProfileHandler) writeViewResponse(w http.ResponseWriter, r *http.Request, view *repository.FreelanceProfileView) {
	skills := h.loadSkills(r, view.Profile.OrganizationID)
	pricing := h.loadPricing(r, view.Profile.ID)
	res.JSON(w, http.StatusOK, response.NewFreelanceProfileResponse(view, pricing, skills))
}

// loadSkills fetches the org skills via the optional SkillsReader.
// A nil reader or any error yields an empty (non-nil) slice so
// the outer profile read never fails because of a skill hiccup.
func (h *FreelanceProfileHandler) loadSkills(r *http.Request, orgID uuid.UUID) []response.ProfileSkillSummary {
	if h.skillsReader == nil {
		return []response.ProfileSkillSummary{}
	}
	skills, err := h.skillsReader.GetProfileSkills(r.Context(), orgID)
	if err != nil {
		return []response.ProfileSkillSummary{}
	}
	return response.NewProfileSkillSummaryList(skills)
}

// loadPricing fetches the profile pricing via the optional
// FreelancePricingReader. Nil reader or any error yields nil so
// the response's "pricing" key becomes null.
func (h *FreelanceProfileHandler) loadPricing(r *http.Request, profileID uuid.UUID) *domainfreelancepricing.Pricing {
	if h.pricingReader == nil {
		return nil
	}
	p, err := h.pricingReader.Get(r.Context(), profileID)
	if err != nil {
		return nil
	}
	return p
}

// handleFreelanceProfileError maps domain-level errors to stable
// HTTP status codes. Kept pure so the mapping is unit-testable.
func handleFreelanceProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, freelanceprofile.ErrProfileNotFound):
		res.Error(w, http.StatusNotFound, "freelance_profile_not_found", err.Error())
	case errors.Is(err, profile.ErrInvalidAvailabilityStatus):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
