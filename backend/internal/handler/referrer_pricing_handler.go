package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	referrerpricingapp "marketplace-backend/internal/app/referrerpricing"
	domainpricing "marketplace-backend/internal/domain/referrerpricing"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// ReferrerProfileLookup is the minimal read contract this handler
// needs to resolve the referrer profile id from an org id. Mirrors
// FreelanceProfileLookup shape — the lookup may create a default
// row lazily before returning the id, because referrer profiles
// do not exist for every provider_personal org out of the gate.
type ReferrerProfileLookup interface {
	GetReferrerProfileIDByOrgID(ctx context.Context, orgID uuid.UUID) (uuid.UUID, error)
}

// ReferrerPricingHandler owns the referrer pricing HTTP endpoints.
type ReferrerPricingHandler struct {
	pricing  *referrerpricingapp.Service
	profiles ReferrerProfileLookup
}

// NewReferrerPricingHandler wires the handler.
func NewReferrerPricingHandler(pricing *referrerpricingapp.Service, profiles ReferrerProfileLookup) *ReferrerPricingHandler {
	return &ReferrerPricingHandler{pricing: pricing, profiles: profiles}
}

// GetMy returns the authenticated user's referrer pricing row.
func (h *ReferrerPricingHandler) GetMy(w http.ResponseWriter, r *http.Request) {
	profileID, ok := h.resolveProfile(w, r)
	if !ok {
		return
	}

	p, err := h.pricing.Get(r.Context(), profileID)
	if err != nil {
		if errors.Is(err, domainpricing.ErrPricingNotFound) {
			res.JSON(w, http.StatusOK, map[string]any{"data": nil})
			return
		}
		handleReferrerPricingError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewReferrerPricingSummary(p),
	})
}

// UpsertMy writes or updates the referrer pricing row.
func (h *ReferrerPricingHandler) UpsertMy(w http.ResponseWriter, r *http.Request) {
	profileID, ok := h.resolveProfile(w, r)
	if !ok {
		return
	}

	var req request.UpsertReferrerPricingRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	p, err := h.pricing.Upsert(r.Context(), referrerpricingapp.UpsertInput{
		ProfileID:  profileID,
		Type:       domainpricing.PricingType(req.Type),
		MinAmount:  req.MinAmount,
		MaxAmount:  req.MaxAmount,
		Currency:   req.Currency,
		Note:       req.Note,
		Negotiable: req.Negotiable,
	})
	if err != nil {
		handleReferrerPricingError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewReferrerPricingSummary(p),
	})
}

// DeleteMy removes the referrer pricing row.
func (h *ReferrerPricingHandler) DeleteMy(w http.ResponseWriter, r *http.Request) {
	profileID, ok := h.resolveProfile(w, r)
	if !ok {
		return
	}

	if err := h.pricing.Delete(r.Context(), profileID); err != nil {
		handleReferrerPricingError(w, err)
		return
	}
	res.NoContent(w)
}

// resolveProfile resolves the referrer profile id or writes the
// appropriate HTTP error and returns ok=false.
func (h *ReferrerPricingHandler) resolveProfile(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return uuid.Nil, false
	}
	profileID, err := h.profiles.GetReferrerProfileIDByOrgID(r.Context(), orgID)
	if err != nil {
		handleReferrerPricingError(w, err)
		return uuid.Nil, false
	}
	return profileID, true
}

// handleReferrerPricingError maps domain-level errors to HTTP
// statuses.
func handleReferrerPricingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domainpricing.ErrPricingNotFound):
		res.Error(w, http.StatusNotFound, "referrer_pricing_not_found", err.Error())
	case errors.Is(err, domainpricing.ErrInvalidType),
		errors.Is(err, domainpricing.ErrNegativeAmount),
		errors.Is(err, domainpricing.ErrMaxLessThanMin),
		errors.Is(err, domainpricing.ErrRangeNotAllowedForType),
		errors.Is(err, domainpricing.ErrRangeRequiredForType),
		errors.Is(err, domainpricing.ErrInvalidCurrency),
		errors.Is(err, domainpricing.ErrInvalidCurrencyForType),
		errors.Is(err, domainpricing.ErrCommissionPctOutOfRange):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

