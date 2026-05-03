// LEGACY profile handler — agency-only going forward.
//
// Since the split-profile refactor (migrations 096-104) the
// provider_personal path is served by FreelanceProfileHandler and
// ReferrerProfileHandler. The GET /api/v1/profile and related
// endpoints defined in this file continue to back the agency
// workflow until the agency refactor ships.
//
// A frontend that moves a provider_personal org onto the new
// /freelance-profile or /referrer-profile endpoints and keeps
// hitting /profile here will receive data from the legacy agency-
// scoped profiles table — which no longer contains rows for that
// org after migration 104 — and get a 404. That is intentional:
// the split flipped the ownership model and any client still
// using these URLs for provider_personal is broken on purpose.

package handler

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	clientprofileapp "marketplace-backend/internal/app/clientprofile"
	profileapp "marketplace-backend/internal/app/profile"
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

// ClientStatsReader is the narrow read contract the profile handler
// needs to decorate GET /api/v1/profile with the owner's client-side
// aggregates (total_spent, review_count, average_rating,
// projects_completed_as_client). Defined locally so the profile
// handler does not carry a wider dependency on the clientprofile
// app package's public surface. Nil is tolerated — the Client block
// is simply omitted from the response.
type ClientStatsReader interface {
	GetStats(ctx context.Context, orgID uuid.UUID) (*clientprofileapp.ClientStats, error)
}

// PublicProfileReader is the narrow read contract the handler uses
// for the cacheable public profile read paths. The concrete
// *profileapp.Service satisfies it directly; in production the
// wiring layer wraps the service with a Redis cache decorator that
// also satisfies this contract — the handler is then completely
// agnostic of whether a cache is present.
//
// Defined locally (not in port/) to keep the handler's
// dependency graph flat: the only consumer is this file.
type PublicProfileReader interface {
	GetProfile(ctx context.Context, orgID uuid.UUID) (*profile.Profile, error)
}

// ExpertiseReader is the narrow read contract the handler uses for
// the cacheable expertise list (org's declared specializations).
// The concrete *profileapp.ExpertiseService satisfies it directly;
// in production main.go wraps it with a Redis cache decorator.
type ExpertiseReader interface {
	ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]string, error)
}

// ProfileHandler wires the profile-related HTTP endpoints to the
// profile application services. The expertise service, skills
// reader, pricing reader, and client stats reader are optional at
// the struct level so existing unit tests that only care about the
// main profile flow can pass nil — in production wiring
// (cmd/api/main.go) they are always non-nil.
//
// publicReader is consulted first on read paths (GetPublicProfile /
// owner self-read after a successful write). It defaults to the
// profile service itself but can be overridden via WithPublicReader
// to point at a Redis-backed cache decorator. Writes always go
// through profileService — the cache invalidates itself via the
// service's WithCacheInvalidator hook.
type ProfileHandler struct {
	profileService    *profileapp.Service
	expertiseService  *profileapp.ExpertiseService
	publicReader      PublicProfileReader
	expertiseReader   ExpertiseReader
	skillsReader      SkillsReader
	pricingReader     PricingReader
	clientStatsReader ClientStatsReader
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
	h := &ProfileHandler{
		profileService:   profileService,
		expertiseService: expertiseService,
		// Default the public reader to the service itself so legacy
		// call sites (tests + main.go before the cache wiring lands)
		// keep working without a cache. WithPublicReader overrides
		// this with the Redis-backed decorator in production.
		publicReader: profileService,
	}
	// Default expertise reader: nil-safe assignment — assigning a
	// typed-nil *ExpertiseService into an interface field would
	// produce a non-nil interface holding a nil pointer, which then
	// nil-derefs on call. Only wrap when the concrete is actually
	// non-nil. Tests that pass nil intentionally still get a clean
	// "no expertise" path via loadExpertise's nil guard.
	if expertiseService != nil {
		h.expertiseReader = expertiseService
	}
	return h
}

// WithExpertiseReader overrides the default expertise reader.
// Pass a Redis cache decorator that wraps the expertise service.
// Nil is a no-op.
func (h *ProfileHandler) WithExpertiseReader(reader ExpertiseReader) *ProfileHandler {
	if reader != nil {
		h.expertiseReader = reader
	}
	return h
}

// WithPublicReader overrides the default reader used for public
// profile reads (GetPublicProfile + the owner self-read after a
// successful write). Pass a Redis cache decorator that wraps the
// underlying service, or nil to leave the existing reader in
// place. The override is applied in main.go after the cache is
// constructed; in tests this method is rarely needed because the
// service itself is fast enough.
func (h *ProfileHandler) WithPublicReader(reader PublicProfileReader) *ProfileHandler {
	if reader != nil {
		h.publicReader = reader
	}
	return h
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

// WithClientStatsReader sets the client-stats reader used to decorate
// the authenticated /api/v1/profile response with the owner's
// client-side aggregates. Returns the same handler for fluent
// wiring. Passing nil is a no-op — the Client block simply stays
// absent in the response.
func (h *ProfileHandler) WithClientStatsReader(reader ClientStatsReader) *ProfileHandler {
	if reader != nil {
		h.clientStatsReader = reader
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

	p, err := h.publicReader.GetProfile(r.Context(), orgID)
	if err != nil {
		handleProfileError(w, err)
		return
	}

	domains := h.loadExpertise(r, orgID)
	skills := h.loadSkills(r, orgID)
	pricing := h.loadPricing(r, orgID)
	client := h.loadClientStats(r, orgID)
	res.JSON(w, http.StatusOK,
		response.NewProfileResponseWithExtras(p, domains, skills, pricing).
			WithClientSection(client))
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
	client := h.loadClientStats(r, orgID)
	res.JSON(w, http.StatusOK,
		response.NewProfileResponseWithExtras(p, domains, skills, pricing).
			WithClientSection(client))
}

// ---------------------------------------------------------------
// Tier 1 completion endpoints (migration 083)
// ---------------------------------------------------------------

// UpdateMyLocation writes the org's location block (city, country
// code, work modes, travel radius). The web / mobile clients use a
// client-side city autocomplete (BAN + Photon) that ships canonical
// latitude / longitude alongside the selected municipality — when
// both are present the server trusts them verbatim and skips
// geocoding. When lat/lng are absent the server falls back to the
// optional Nominatim-backed geocoder so admin tooling and
// programmatic writes keep working without an embedded geocoder.
func (h *ProfileHandler) UpdateMyLocation(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req struct {
		City           string   `json:"city"`
		CountryCode    string   `json:"country_code"`
		Latitude       *float64 `json:"latitude"`
		Longitude      *float64 `json:"longitude"`
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
		Latitude:       req.Latitude,
		Longitude:      req.Longitude,
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
