package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	referrerprofileapp "marketplace-backend/internal/app/referrerprofile"
	"marketplace-backend/internal/domain/organization"
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
	orgOwner      OrgOwnerLookup
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

// OrgOwnerLookup is the narrow contract the handler uses to resolve
// the apporteur's user_id from the organization id. A provider_personal
// org has exactly one owner — the referrer is that owner — and
// referrals reference users, so we translate here at the edge so the
// public URL stays keyed on the stable orgID that the rest of the
// profile surface already uses.
type OrgOwnerLookup interface {
	OwnerUserIDForOrg(ctx context.Context, orgID uuid.UUID) (uuid.UUID, error)
}

// WithOrgOwnerLookup wires the org→owner lookup used by the reputation
// endpoint. Nil is a no-op — the endpoint will return 404 for any
// orgID until the lookup is wired.
func (h *ReferrerProfileHandler) WithOrgOwnerLookup(lookup OrgOwnerLookup) *ReferrerProfileHandler {
	if lookup != nil {
		h.orgOwner = lookup
	}
	return h
}

// GetReputation returns the apporteur reputation aggregate for the
// given organization — a dedicated rating (distinct from the user's
// freelance rating) and a cursor-paginated history of attributed
// missions.
//
// Public endpoint — no auth required, same pattern as other public
// profile reads. Keyed on orgID for URL symmetry with
// /referrer-profiles/{orgID}; internally the handler translates to
// the owner user_id because referrals reference users.
//
// Empty-state contract: a referrer with zero attributed projects MUST
// receive a 200 OK with an empty `history` array — never a 404. The
// public profile read (GetByOrgID) already auto-creates a default row,
// so the reputation surface is the natural empty state for that org
// and a 404 here would render the load-error UI on a perfectly valid
// fresh referrer profile (the production bug observed on
// /fr/referrers/{uuid} when the org has no referrals yet).
func (h *ReferrerProfileHandler) GetReputation(w http.ResponseWriter, r *http.Request) {
	orgIDParam := chi.URLParam(r, "orgID")
	orgID, err := uuid.Parse(orgIDParam)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "organization ID must be a valid UUID")
		return
	}

	if h.orgOwner == nil {
		// Lookup is optional at wire time — surface an empty aggregate
		// rather than 500 so the feature is fully removable.
		res.JSON(w, http.StatusOK, response.NewReferrerReputationResponse(referrerprofileapp.ReferrerReputation{}))
		return
	}
	userID, err := h.orgOwner.OwnerUserIDForOrg(r.Context(), orgID)
	if err != nil {
		// Distinguish "the org row genuinely does not exist" from any
		// other repository error. The former is the normal "no such
		// referrer" case and we still return 200 + empty payload so
		// the public profile + reputation surfaces stay symmetrical
		// (the profile read auto-creates an empty row, so the
		// reputation read should not 404 just because no referrals
		// exist yet). Any other error is an actual infrastructure
		// failure — log it and surface a 500 so it is visible in the
		// browser network tab AND the structured server logs.
		if errors.Is(err, organization.ErrOrgNotFound) {
			res.JSON(w, http.StatusOK, response.NewReferrerReputationResponse(referrerprofileapp.ReferrerReputation{}))
			return
		}
		slog.Error("referrer reputation: org owner lookup failed",
			"org_id", orgID,
			"error", err.Error(),
		)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to resolve referrer organization")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	rep, err := h.svc.GetReferrerReputation(r.Context(), userID, cursor, limit)
	if err != nil {
		// Log the underlying error so prod-ops can correlate the
		// generic 500 the browser sees with a concrete root cause.
		// Without this, the only signal is the frontend toast.
		slog.Error("referrer reputation: aggregate load failed",
			"org_id", orgID,
			"user_id", userID,
			"error", err.Error(),
		)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to load reputation")
		return
	}

	res.JSON(w, http.StatusOK, response.NewReferrerReputationResponse(rep))
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
