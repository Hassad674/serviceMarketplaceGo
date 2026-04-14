package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	profilepricingapp "marketplace-backend/internal/app/profilepricing"
	domainpricing "marketplace-backend/internal/domain/profilepricing"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// ProfilePricingHandler owns the /api/v1/profile/pricing routes.
// Lives in its own handler (rather than on ProfileHandler) to
// preserve the feature-isolation principle: the pricing feature
// can be deleted by removing this file, its route wiring, and the
// domain / app / adapter packages without touching the classic
// profile flow.
type ProfilePricingHandler struct {
	svc *profilepricingapp.Service
}

// NewProfilePricingHandler constructs the handler with the app
// service. Both arguments are mandatory — there are no optional
// collaborators.
func NewProfilePricingHandler(svc *profilepricingapp.Service) *ProfilePricingHandler {
	return &ProfilePricingHandler{svc: svc}
}

// ListMyPricing returns every pricing row for the authenticated
// user's org. 0, 1 or 2 rows depending on declaration state.
func (h *ProfilePricingHandler) ListMyPricing(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	pricing, err := h.svc.GetForOrg(r.Context(), orgID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}

	res.JSON(w, http.StatusOK, response.NewPricingSummaryList(pricing))
}

// UpsertMyPricing writes or updates one pricing row for the
// authenticated user's org. The pricing_kind is part of the
// request body (not the URL) so a single endpoint covers both
// "create direct pricing" and "create referral pricing" without
// duplicating the payload schema.
func (h *ProfilePricingHandler) UpsertMyPricing(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req struct {
		Kind      string `json:"kind"`
		Type      string `json:"type"`
		MinAmount int64  `json:"min_amount"`
		MaxAmount *int64 `json:"max_amount"`
		Currency  string `json:"currency"`
		Note      string `json:"note"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	pricing, err := h.svc.Upsert(r.Context(), profilepricingapp.UpsertInput{
		OrganizationID: orgID,
		Kind:           domainpricing.PricingKind(req.Kind),
		Type:           domainpricing.PricingType(req.Type),
		MinAmount:      req.MinAmount,
		MaxAmount:      req.MaxAmount,
		Currency:       req.Currency,
		Note:           req.Note,
	})
	if err != nil {
		handlePricingError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewPricingSummary(pricing))
}

// DeleteMyPricingByKind deletes one pricing row. The kind is a
// URL parameter (not body) because DELETE requests conventionally
// carry no body in the marketplace API.
func (h *ProfilePricingHandler) DeleteMyPricingByKind(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	kind := domainpricing.PricingKind(chi.URLParam(r, "kind"))
	if err := h.svc.DeleteKind(r.Context(), orgID, kind); err != nil {
		handlePricingError(w, err)
		return
	}
	res.NoContent(w)
}

// handlePricingError maps domain-level pricing errors to the
// stable error code / HTTP status table. Kept pure (no logging,
// no side effects) so the error mapping is unit-testable in
// isolation.
func handlePricingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domainpricing.ErrPricingNotFound):
		res.Error(w, http.StatusNotFound, "pricing_not_found", err.Error())
	case errors.Is(err, domainpricing.ErrKindNotAllowedForRole):
		res.Error(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, domainpricing.ErrInvalidKind),
		errors.Is(err, domainpricing.ErrInvalidType),
		errors.Is(err, domainpricing.ErrTypeNotAllowedForKind),
		errors.Is(err, domainpricing.ErrNegativeAmount),
		errors.Is(err, domainpricing.ErrMaxLessThanMin),
		errors.Is(err, domainpricing.ErrRangeNotAllowedForType),
		errors.Is(err, domainpricing.ErrRangeRequiredForType),
		errors.Is(err, domainpricing.ErrInvalidCurrency),
		errors.Is(err, domainpricing.ErrInvalidCurrencyForType):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
