package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	profileapp "marketplace-backend/internal/app/profile"
	"marketplace-backend/internal/domain/expertise"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// ProfileHandler wires the profile-related HTTP endpoints to the
// profile application services. The expertise service is optional
// at the struct level so existing unit tests that only care about
// the main profile flow can pass nil — in production wiring
// (cmd/api/main.go) it is always non-nil.
type ProfileHandler struct {
	profileService   *profileapp.Service
	expertiseService *profileapp.ExpertiseService
}

// NewProfileHandler constructs the handler with both services wired.
// A nil expertiseService is tolerated: read endpoints will return an
// empty expertise list and the write endpoint will respond with 503.
// This keeps older unit tests valid without forcing them to stub a
// second service they don't exercise.
func NewProfileHandler(
	profileService *profileapp.Service,
	expertiseService *profileapp.ExpertiseService,
) *ProfileHandler {
	return &ProfileHandler{
		profileService:   profileService,
		expertiseService: expertiseService,
	}
}

// GetMyProfile returns the org profile of the authenticated user's
// current organization. All operators in the same org see the same
// profile — this is the Stripe Dashboard shared-workspace model.
func (h *ProfileHandler) GetMyProfile(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	p, err := h.profileService.GetProfile(r.Context(), orgID)
	if err != nil {
		handleProfileError(w, err)
		return
	}

	domains := h.loadExpertise(r, orgID)
	res.JSON(w, http.StatusOK, response.NewProfileResponse(p, domains))
}

// UpdateMyProfile updates the authenticated user's org profile.
func (h *ProfileHandler) UpdateMyProfile(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req struct {
		Title                string `json:"title"`
		About                string `json:"about"`
		PhotoURL             string `json:"photo_url"`
		PresentationVideoURL string `json:"presentation_video_url"`
		ReferrerAbout        string `json:"referrer_about"`
		ReferrerVideoURL     string `json:"referrer_video_url"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	input := profileapp.UpdateProfileInput{
		Title:                req.Title,
		About:                req.About,
		PhotoURL:             req.PhotoURL,
		PresentationVideoURL: req.PresentationVideoURL,
		ReferrerAbout:        req.ReferrerAbout,
		ReferrerVideoURL:     req.ReferrerVideoURL,
	}

	p, err := h.profileService.UpdateProfile(r.Context(), orgID, input)
	if err != nil {
		handleProfileError(w, err)
		return
	}

	domains := h.loadExpertise(r, orgID)
	res.JSON(w, http.StatusOK, response.NewProfileResponse(p, domains))
}

// SearchProfiles surfaces org-level public profiles for discovery.
// `type` query param filters by org type: freelancer = provider_personal,
// agency = agency, enterprise = enterprise, referrer = provider_personal
// with referrer flag enabled.
func (h *ProfileHandler) SearchProfiles(w http.ResponseWriter, r *http.Request) {
	typeFilter := r.URL.Query().Get("type")

	var orgTypeFilter string
	var referrerOnly bool

	switch typeFilter {
	case "freelancer":
		orgTypeFilter = "provider_personal"
	case "agency":
		orgTypeFilter = "agency"
	case "enterprise":
		orgTypeFilter = "enterprise"
	case "referrer":
		orgTypeFilter = "provider_personal"
		referrerOnly = true
	default:
		orgTypeFilter = ""
	}

	limit := 20
	if limitParam := r.URL.Query().Get("limit"); limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	cursor := r.URL.Query().Get("cursor")

	profiles, nextCursor, err := h.profileService.SearchPublic(r.Context(), orgTypeFilter, referrerOnly, cursor, limit)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}

	// TODO(expertise): batch-load expertise for search results via
	// ExpertiseRepository.ListByOrganizationIDs once the search results
	// card UI surfaces the expertise chips. For now, the summary DTO
	// in discovery does not carry expertise — the detail page fetches
	// it lazily. Implementing it requires plumbing the expertise
	// service (or repo) into SearchProfiles and passing a map[orgID][]key
	// into NewPublicProfileSummaryList so each summary gets its own
	// slice without an N+1 pattern. Already supported by the repo.
	res.JSON(w, http.StatusOK, map[string]any{
		"data":        response.NewPublicProfileSummaryList(profiles),
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

// GetPublicProfile returns any organization's public profile.
// Route param is the organization id.
func (h *ProfileHandler) GetPublicProfile(w http.ResponseWriter, r *http.Request) {
	orgIDParam := chi.URLParam(r, "orgId")
	orgID, err := uuid.Parse(orgIDParam)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "organization ID must be a valid UUID")
		return
	}

	p, err := h.profileService.GetProfile(r.Context(), orgID)
	if err != nil {
		handleProfileError(w, err)
		return
	}

	domains := h.loadExpertise(r, orgID)
	res.JSON(w, http.StatusOK, response.NewProfileResponse(p, domains))
}

// UpdateMyExpertise replaces the authenticated organization's full
// list of declared expertise domains in a single atomic write. The
// request body is {"domains": ["development", "design_ui_ux"]}; the
// array order is preserved as the display order. See the domain
// package for validation rules and the service layer for the error
// → HTTP mapping.
func (h *ProfileHandler) UpdateMyExpertise(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	if h.expertiseService == nil {
		res.Error(w, http.StatusServiceUnavailable, "expertise_unavailable", "expertise service is not configured")
		return
	}

	var req struct {
		Domains []string `json:"domains"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	// Hard upper bound on request size — an obviously bogus payload
	// (e.g. 10k keys) is rejected before we hit the per-org-type
	// check so the service layer never allocates oversized slices.
	const maxRequestSize = 20
	if len(req.Domains) > maxRequestSize {
		res.Error(w, http.StatusBadRequest, "validation_error", "too many domains in request")
		return
	}

	domains, err := h.expertiseService.SetExpertise(r.Context(), orgID, req.Domains)
	if err != nil {
		handleExpertiseError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"expertise_domains": domains,
		},
	})
}

// loadExpertise fetches the org's expertise list for embedding in a
// profile response. Any error or a nil expertise service is
// interpreted as "no declared expertise" so a transient expertise
// read failure never fails the whole profile endpoint. The caller
// gets a guaranteed non-nil empty slice when nothing is declared.
func (h *ProfileHandler) loadExpertise(r *http.Request, orgID uuid.UUID) []string {
	if h.expertiseService == nil {
		return []string{}
	}
	domains, err := h.expertiseService.ListByOrganization(r.Context(), orgID)
	if err != nil {
		// Do not surface — the profile read succeeded, and the expertise
		// section is decorative. The error is already logged deep in the
		// repository via fmt.Errorf %w. Returning an empty slice keeps
		// the envelope shape stable.
		return []string{}
	}
	return domains
}

func handleProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, profile.ErrProfileNotFound):
		res.Error(w, http.StatusNotFound, "profile_not_found", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

// handleExpertiseError maps domain-level expertise errors to the
// stable error code / HTTP status table defined in the API contract.
// Keeping this function pure (no logging, no side effects) lets the
// handler stay thin and the tests focused on the mapping itself.
func handleExpertiseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, expertise.ErrForbiddenOrgType):
		res.Error(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, expertise.ErrUnknownKey),
		errors.Is(err, expertise.ErrDuplicate),
		errors.Is(err, expertise.ErrOverMax):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
