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
	domainpricing "marketplace-backend/internal/domain/profilepricing"
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

// PricingReader is the minimal read contract the profile handler
// needs to decorate public profile responses with the org's
// declared pricing rows. Same "local interface" pattern as
// SkillsReader — keeps the profile handler independent of the
// profile pricing app package's public surface.
//
// A nil PricingReader is tolerated by every read path: the
// handler returns an empty pricing slice, never fails the outer
// profile read. In production cmd/api/main.go injects
// *profilepricingapp.Service.
type PricingReader interface {
	GetForOrg(ctx context.Context, orgID uuid.UUID) ([]*domainpricing.Pricing, error)
	GetForOrgsBatch(ctx context.Context, orgIDs []uuid.UUID) (map[uuid.UUID][]*domainpricing.Pricing, error)
}

// ProfileHandler wires the profile-related HTTP endpoints to the
// profile application services. The expertise service, skills
// reader, and pricing reader are optional at the struct level so
// existing unit tests that only care about the main profile flow
// can pass nil — in production wiring (cmd/api/main.go) they are
// always non-nil.
type ProfileHandler struct {
	profileService   *profileapp.Service
	expertiseService *profileapp.ExpertiseService
	skillsReader     SkillsReader
	pricingReader    PricingReader
}

// NewProfileHandler constructs the handler with the profile service
// and an optional expertise service. Skills and pricing are wired
// via WithSkillsReader / WithPricingReader after construction to
// keep existing call sites (older unit tests) intact without
// forcing a signature change.
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

// WithPricingReader sets the pricing reader used to decorate public
// profile responses with the org's declared pricing rows. Returns
// the same handler for fluent wiring. Passing nil is a no-op.
func (h *ProfileHandler) WithPricingReader(reader PricingReader) *ProfileHandler {
	if reader != nil {
		h.pricingReader = reader
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
	pricing := h.loadPricing(r, orgID)
	res.JSON(w, http.StatusOK, response.NewProfileResponseWithExtras(p, domains, skills, pricing))
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
	pricing := h.loadPricing(r, orgID)
	res.JSON(w, http.StatusOK, response.NewProfileResponseWithExtras(p, domains, skills, pricing))
}

// ---------------------------------------------------------------
// Tier 1 completion endpoints (migration 083)
// ---------------------------------------------------------------

// UpdateMyLocation writes the org's location block (city, country
// code, work modes, travel radius). Coordinates are derived
// server-side via the geocoder — the request body never carries
// lat/lng to prevent clients from forging coordinates.
func (h *ProfileHandler) UpdateMyLocation(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req struct {
		City           string   `json:"city"`
		CountryCode    string   `json:"country_code"`
		WorkMode       []string `json:"work_mode"`
		TravelRadiusKm *int     `json:"travel_radius_km"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	err := h.profileService.UpdateLocation(r.Context(), orgID, profileapp.UpdateLocationInput{
		City:           req.City,
		CountryCode:    req.CountryCode,
		WorkMode:       req.WorkMode,
		TravelRadiusKm: req.TravelRadiusKm,
	})
	if err != nil {
		handleProfileError(w, err)
		return
	}
	h.writeProfileFromOrg(w, r, orgID)
}

// UpdateMyLanguages replaces the two language arrays.
func (h *ProfileHandler) UpdateMyLanguages(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req struct {
		Professional   []string `json:"professional"`
		Conversational []string `json:"conversational"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := h.profileService.UpdateLanguages(r.Context(), orgID, req.Professional, req.Conversational); err != nil {
		handleProfileError(w, err)
		return
	}
	h.writeProfileFromOrg(w, r, orgID)
}

// UpdateMyAvailability writes the direct + optional referrer
// availability statuses. Referrer may be omitted (nil pointer in
// the request) to clear the column.
func (h *ProfileHandler) UpdateMyAvailability(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req struct {
		AvailabilityStatus         string  `json:"availability_status"`
		ReferrerAvailabilityStatus *string `json:"referrer_availability_status"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	direct, err := profile.ParseAvailabilityStatus(req.AvailabilityStatus)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	var referrer *profile.AvailabilityStatus
	if req.ReferrerAvailabilityStatus != nil {
		parsed, err := profile.ParseAvailabilityStatus(*req.ReferrerAvailabilityStatus)
		if err != nil {
			res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
			return
		}
		referrer = &parsed
	}

	if err := h.profileService.UpdateAvailability(r.Context(), orgID, direct, referrer); err != nil {
		handleProfileError(w, err)
		return
	}
	h.writeProfileFromOrg(w, r, orgID)
}

// writeProfileFromOrg fetches and writes the full profile DTO for
// the given org — used by every Tier 1 mutation endpoint so the
// client always receives the canonical post-write shape in one
// roundtrip.
func (h *ProfileHandler) writeProfileFromOrg(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	p, err := h.profileService.GetProfile(r.Context(), orgID)
	if err != nil {
		handleProfileError(w, err)
		return
	}
	domains := h.loadExpertise(r, orgID)
	skills := h.loadSkills(r, orgID)
	pricing := h.loadPricing(r, orgID)
	res.JSON(w, http.StatusOK, response.NewProfileResponseWithExtras(p, domains, skills, pricing))
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

	// Batch-load skills AND pricing for the entire search result
	// page in a single database roundtrip each. Keeps the listing
	// endpoint O(1) queries for decoration (vs N+1 that would kick
	// in with per-card fetch). Expertise is intentionally NOT
	// loaded here — the detail page still fetches it lazily.
	skillsByOrg := h.loadSkillsBatch(r, profiles)
	pricingByOrg := h.loadPricingBatch(r, profiles)
	res.JSON(w, http.StatusOK, map[string]any{
		"data":        response.NewPublicProfileSummaryListWithExtras(profiles, skillsByOrg, pricingByOrg),
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
	pricing := h.loadPricing(r, orgID)
	res.JSON(w, http.StatusOK, response.NewProfileResponseWithExtras(p, domains, skills, pricing))
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

// loadPricing fetches the org's pricing rows (0, 1 or 2) for
// embedding in a profile response. Graceful degradation matches
// loadSkills: any failure yields an empty slice so the profile
// read never fails because of a pricing glitch.
func (h *ProfileHandler) loadPricing(r *http.Request, orgID uuid.UUID) []*domainpricing.Pricing {
	if h.pricingReader == nil {
		return []*domainpricing.Pricing{}
	}
	pricing, err := h.pricingReader.GetForOrg(r.Context(), orgID)
	if err != nil {
		return []*domainpricing.Pricing{}
	}
	return pricing
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

// loadPricingBatch batches pricing fetches across a search result
// page — mirrors loadSkillsBatch semantics and tolerates the same
// graceful-degradation paths (nil reader, error, empty input).
func (h *ProfileHandler) loadPricingBatch(r *http.Request, profiles []*profile.PublicProfile) map[uuid.UUID][]*domainpricing.Pricing {
	empty := map[uuid.UUID][]*domainpricing.Pricing{}
	if h.pricingReader == nil || len(profiles) == 0 {
		return empty
	}
	orgIDs := make([]uuid.UUID, 0, len(profiles))
	for _, p := range profiles {
		orgIDs = append(orgIDs, p.OrganizationID)
	}
	pricing, err := h.pricingReader.GetForOrgsBatch(r.Context(), orgIDs)
	if err != nil {
		return empty
	}
	return pricing
}

func handleProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, profile.ErrProfileNotFound):
		res.Error(w, http.StatusNotFound, "profile_not_found", err.Error())
	case errors.Is(err, profile.ErrInvalidCountryCode),
		errors.Is(err, profile.ErrInvalidAvailabilityStatus):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
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
