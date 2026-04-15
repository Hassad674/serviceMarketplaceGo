package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	appskill "marketplace-backend/internal/app/skill"
	domainskill "marketplace-backend/internal/domain/skill"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// skillService is the narrow interface the handler depends on. Keeping
// the dependency as an interface (rather than *appskill.Service) lets
// the unit tests mock the service with a tiny stub, and keeps the
// handler decoupled from any service-layer implementation details.
// The concrete *appskill.Service in cmd/api/main.go satisfies this
// contract by definition.
type skillService interface {
	GetCuratedForExpertise(ctx context.Context, key string, limit int) ([]*domainskill.CatalogEntry, error)
	CountCuratedForExpertise(ctx context.Context, key string) (int, error)
	Autocomplete(ctx context.Context, q string, limit int) ([]*domainskill.CatalogEntry, error)
	GetProfileSkills(ctx context.Context, orgID uuid.UUID) ([]*domainskill.ProfileSkill, error)
	ReplaceProfileSkills(ctx context.Context, in appskill.ReplaceProfileSkillsInput) error
	CreateUserSkill(ctx context.Context, in appskill.CreateUserSkillInput) (*domainskill.CatalogEntry, error)
}

// MultiPersonaSearchPublisher is the narrow port the skill handler
// uses to trigger a reindex across every persona an org could
// expose. Skills are persona-agnostic, so the handler does not
// know whether the org is freelance, agency, or referrer — it
// emits one event for each persona and lets the publisher's
// debounce + the worker's "no profile" handling deduplicate.
type MultiPersonaSearchPublisher interface {
	PublishReindexAllPersonas(ctx context.Context, orgID uuid.UUID) error
}

// SkillHandler groups every HTTP endpoint exposed by the skill feature.
type SkillHandler struct {
	svc           skillService
	searchPublish MultiPersonaSearchPublisher
}

// NewSkillHandler constructs the HTTP handler for skills. The interface
// parameter accepts both the production *appskill.Service and any test
// mock that implements the same contract.
func NewSkillHandler(svc skillService) *SkillHandler {
	return &SkillHandler{svc: svc}
}

// WithSearchIndexPublisher attaches an optional Typesense publisher
// that fires a multi-persona reindex after a successful skills
// mutation.
func (h *SkillHandler) WithSearchIndexPublisher(p MultiPersonaSearchPublisher) *SkillHandler {
	h.searchPublish = p
	return h
}

// ---- Public catalog reads (no auth required — browsing) ----

// GetCuratedByExpertise handles GET /api/v1/skills/catalog?expertise=development&limit=50
// The response shape is { "skills": [...], "total": N } so the panel
// header can show "N curated skills for Development" without doing a
// second round trip.
func (h *SkillHandler) GetCuratedByExpertise(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("expertise")
	limit := parseSkillIntQuery(r, "limit", 50)

	entries, err := h.svc.GetCuratedForExpertise(r.Context(), key, limit)
	if err != nil {
		handleSkillError(w, err)
		return
	}

	count, err := h.svc.CountCuratedForExpertise(r.Context(), key)
	if err != nil {
		handleSkillError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"skills": response.NewSkillsListResponse(entries),
		"total":  count,
	})
}

// Autocomplete handles GET /api/v1/skills/autocomplete?q=re&limit=20
// An empty or whitespace-only q returns a 200 with an empty array —
// the frontend's autocomplete debounce sometimes fires on an empty
// string and this behaviour keeps the UX simple.
func (h *SkillHandler) Autocomplete(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	limit := parseSkillIntQuery(r, "limit", 20)

	entries, err := h.svc.Autocomplete(r.Context(), q, limit)
	if err != nil {
		handleSkillError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewSkillsListResponse(entries))
}

// ---- Profile skills (authenticated — requires Auth middleware) ----

// GetMyProfileSkills handles GET /api/v1/profile/skills. Returns the
// ordered list of skills declared on the caller's organization's
// public profile.
func (h *SkillHandler) GetMyProfileSkills(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization context required")
		return
	}

	skills, err := h.svc.GetProfileSkills(r.Context(), orgID)
	if err != nil {
		handleSkillError(w, err)
		return
	}

	// V1 note: display_text is denormalized from skill_text so a
	// deleted catalog entry does not break the UI. The frontend keeps
	// its own autocomplete cache of { skill_text -> display_text }
	// and falls back to the raw text if the cache misses.
	out := make([]response.ProfileSkillResponse, 0, len(skills))
	for _, s := range skills {
		out = append(out, response.ProfileSkillResponse{
			SkillText:   s.SkillText,
			DisplayText: s.SkillText,
			Position:    s.Position,
		})
	}

	res.JSON(w, http.StatusOK, out)
}

// PutMyProfileSkills handles PUT /api/v1/profile/skills. The payload
// is { "skill_texts": [...] } — array position IS the display order.
func (h *SkillHandler) PutMyProfileSkills(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization context required")
		return
	}

	var req request.PutProfileSkillsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	err := h.svc.ReplaceProfileSkills(r.Context(), appskill.ReplaceProfileSkillsInput{
		OrganizationID: orgID,
		SkillTexts:     req.SkillTexts,
	})
	if err != nil {
		handleSkillError(w, err)
		return
	}

	if h.searchPublish != nil {
		if pubErr := h.searchPublish.PublishReindexAllPersonas(r.Context(), orgID); pubErr != nil {
			// Best-effort. A degraded search engine must not
			// block a profile mutation.
			slog.Warn("skills: multi-persona reindex publish failed",
				"org_id", orgID, "error", pubErr)
		}
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// CreateUserSkill handles POST /api/v1/skills. Used by the "Create X"
// autocomplete option when the user types a brand-new skill that does
// not yet exist in the catalog.
//
// V1 limitation: the request does not carry expertise_keys. A later
// iteration can inject an expertise resolver into the service so the
// caller's declared expertise domains are inherited automatically.
func (h *SkillHandler) CreateUserSkill(w http.ResponseWriter, r *http.Request) {
	var req request.CreateSkillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "invalid request body")
		return
	}

	entry, err := h.svc.CreateUserSkill(r.Context(), appskill.CreateUserSkillInput{
		DisplayText:   req.DisplayText,
		ExpertiseKeys: nil, // V1 limitation — see comment above
	})
	if err != nil {
		handleSkillError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, response.NewSkillResponse(entry))
}

// ---- error mapping ----

// handleSkillError translates a domain-level sentinel into the stable
// HTTP error code / status table used across the rest of the API. Any
// unknown error surfaces as a generic 500 so internals never leak.
func handleSkillError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domainskill.ErrInvalidExpertiseKey):
		res.Error(w, http.StatusBadRequest, "invalid_expertise_key", err.Error())
	case errors.Is(err, domainskill.ErrInvalidSkillText):
		res.Error(w, http.StatusBadRequest, "invalid_skill_text", err.Error())
	case errors.Is(err, domainskill.ErrInvalidDisplayText):
		res.Error(w, http.StatusBadRequest, "invalid_display_text", err.Error())
	case errors.Is(err, domainskill.ErrDuplicateSkill):
		res.Error(w, http.StatusBadRequest, "duplicate_skill", err.Error())
	case errors.Is(err, domainskill.ErrSkillsDisabledForOrgType):
		res.Error(w, http.StatusForbidden, "skills_disabled", "skills feature is not available for this account type")
	case errors.Is(err, domainskill.ErrTooManySkills):
		res.Error(w, http.StatusBadRequest, "too_many_skills", "you declared more skills than allowed for your account type")
	case errors.Is(err, domainskill.ErrSkillNotFound):
		res.Error(w, http.StatusBadRequest, "skill_not_found", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

// parseSkillIntQuery returns the integer parsed from the named query
// parameter, or def if the parameter is missing, invalid, or negative.
func parseSkillIntQuery(r *http.Request, name string, def int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return def
	}
	return n
}
