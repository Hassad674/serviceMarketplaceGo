package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	profileapp "marketplace-backend/internal/app/profile"
	"marketplace-backend/internal/domain/expertise"
	"marketplace-backend/internal/domain/profile"
	domainskill "marketplace-backend/internal/domain/skill"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// SkillsReader is the minimal read contract the profile handler needs
// to decorate public profile responses with the organization's declared
// skills. Defined locally (not in port/) so the profile handler does
// not carry a direct dependency on the skill app package in its
// public surface — cmd/api/main.go supplies any concrete value that
// matches this shape. In production that value is *skillapp.Service.
//
// A nil SkillsReader is tolerated by every read path: the handler
// returns an empty skill list in that case, exactly like the
// expertise read path.
type SkillsReader interface {
	GetProfileSkills(ctx context.Context, orgID uuid.UUID) ([]*domainskill.ProfileSkill, error)
	GetProfileSkillsBatch(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]*domainskill.ProfileSkill, error)
}

// ProfileHandler wires the profile-related HTTP endpoints to the
// profile application services. The expertise service and skills
// reader are optional at the struct level so existing unit tests
// that only care about the main profile flow can pass nil — in
// production wiring (cmd/api/main.go) they are always non-nil.
type ProfileHandler struct {
	profileService   *profileapp.Service
	expertiseService *profileapp.ExpertiseService
	skillsReader     SkillsReader
}

// NewProfileHandler constructs the handler with the profile service
// and an optional expertise service. Skills are wired via
// WithSkillsReader after construction to keep existing call sites
// (older unit tests) intact without forcing a signature change.
func NewProfileHandler(
	profileService *profileapp.Service,
	expertiseService *profileapp.ExpertiseService,
) *ProfileHandler {
	return &ProfileHandler{
		profileService:   profileService,
		expertiseService: expertiseService,
	}
}

// WithSkillsReader sets the skills reader used to decorate public
// profile responses with the org's declared skills. Returns the
// same handler for fluent wiring: NewProfileHandler(...).WithSkillsReader(svc).
// Passing nil is a no-op (preserves the existing reader, if any).
func (h *ProfileHandler) WithSkillsReader(reader SkillsReader) *ProfileHandler {
	if reader != nil {
		h.skillsReader = reader
	}
	return h
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
	skills := h.loadSkills(r, orgID)
	res.JSON(w, http.StatusOK, response.NewProfileResponse(p, domains, skills))
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
	skills := h.loadSkills(r, orgID)
	res.JSON(w, http.StatusOK, response.NewProfileResponse(p, domains, skills))
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

	// Batch-load skills for the entire search result page in a single
	// database roundtrip. Keeps the listing endpoint O(1) queries for
	// skill decoration (vs N+1 that would kick in with per-card fetch).
	// Expertise is intentionally NOT loaded here — the detail page
	// still fetches it lazily. The TODO on the expertise side can be
	// actioned by mirroring this exact pattern against
	// ExpertiseRepository.ListByOrganizationIDs.
	skillsByOrg := h.loadSkillsBatch(r, profiles)
	res.JSON(w, http.StatusOK, map[string]any{
		"data":        response.NewPublicProfileSummaryListWithSkills(profiles, skillsByOrg),
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
	skills := h.loadSkills(r, orgID)
	res.JSON(w, http.StatusOK, response.NewProfileResponse(p, domains, skills))
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

// loadSkills fetches the org's declared skills for embedding in a
// profile response. Same graceful-degradation semantics as
// loadExpertise: nil reader, repository errors, or any unexpected
// state yields an empty (non-nil) slice rather than failing the
// outer profile read. Skills are decorative on the public profile
// — a transient skill fetch failure must not block the rest of
// the page from rendering.
func (h *ProfileHandler) loadSkills(r *http.Request, orgID uuid.UUID) []*domainskill.ProfileSkill {
	if h.skillsReader == nil {
		return []*domainskill.ProfileSkill{}
	}
	skills, err := h.skillsReader.GetProfileSkills(r.Context(), orgID)
	if err != nil {
		return []*domainskill.ProfileSkill{}
	}
	return skills
}

// loadSkillsBatch fetches skills for every org in a search result
// page in a single database roundtrip. Returns a map keyed by org
// ID with a guaranteed empty slice for orgs that have no skills so
// callers never need nil-checks.
//
// The method tolerates:
//   - nil skillsReader → empty map
//   - repository error → empty map
//   - empty profiles slice → empty map
//
// Decorative semantics match loadSkills: a skill read failure must
// not block the listing endpoint from returning profile rows.
func (h *ProfileHandler) loadSkillsBatch(r *http.Request, profiles []*profile.PublicProfile) map[uuid.UUID][]*domainskill.ProfileSkill {
	empty := map[uuid.UUID][]*domainskill.ProfileSkill{}
	if h.skillsReader == nil || len(profiles) == 0 {
		return empty
	}
	orgIDs := make([]uuid.UUID, 0, len(profiles))
	for _, p := range profiles {
		orgIDs = append(orgIDs, p.OrganizationID)
	}
	skills, err := h.skillsReader.GetProfileSkillsBatch(r.Context(), orgIDs)
	if err != nil {
		return empty
	}
	return skills
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
