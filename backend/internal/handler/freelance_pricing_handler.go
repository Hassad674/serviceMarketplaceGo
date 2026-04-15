package handler

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"

	freelancepricingapp "marketplace-backend/internal/app/freelancepricing"
	domainpricing "marketplace-backend/internal/domain/freelancepricing"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/search"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// FreelanceProfileLookup is the minimal read contract this handler
// needs to resolve the freelance profile id for the authenticated
// user's organization — pricing endpoints receive an org id from
// the JWT context but the freelance_pricing table is keyed on
// profile_id.
//
// Defined locally (not in port/) so the handler does not carry a
// direct dependency on the freelance profile app package at the
// struct level. cmd/api/main.go supplies a thin shim that matches
// this contract.
type FreelanceProfileLookup interface {
	GetFreelanceProfileIDByOrgID(ctx context.Context, orgID uuid.UUID) (uuid.UUID, error)
}

// FreelancePricingHandler owns the freelance pricing HTTP endpoints.
type FreelancePricingHandler struct {
	pricing       *freelancepricingapp.Service
	profiles      FreelanceProfileLookup
	searchPublish SearchIndexPublisher
}

// NewFreelancePricingHandler wires the handler with the pricing
// service and a profile lookup.
func NewFreelancePricingHandler(pricing *freelancepricingapp.Service, profiles FreelanceProfileLookup) *FreelancePricingHandler {
	return &FreelancePricingHandler{pricing: pricing, profiles: profiles}
}

// WithSearchIndexPublisher attaches an optional Typesense publisher
// so every successful pricing mutation triggers a best-effort
// reindex of the freelance document.
func (h *FreelancePricingHandler) WithSearchIndexPublisher(p SearchIndexPublisher) *FreelancePricingHandler {
	h.searchPublish = p
	return h
}

// GetMy returns the current authenticated user's freelance pricing
// row, or a null pricing field when none is declared.
func (h *FreelancePricingHandler) GetMy(w http.ResponseWriter, r *http.Request) {
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
		handleFreelancePricingError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewFreelancePricingSummary(p),
	})
}

// UpsertMy writes or updates the freelance pricing row.
func (h *FreelancePricingHandler) UpsertMy(w http.ResponseWriter, r *http.Request) {
	orgID, profileID, ok := h.resolveOrgAndProfile(w, r)
	if !ok {
		return
	}

	var req request.UpsertFreelancePricingRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	p, err := h.pricing.Upsert(r.Context(), freelancepricingapp.UpsertInput{
		ProfileID:  profileID,
		Type:       domainpricing.PricingType(req.Type),
		MinAmount:  req.MinAmount,
		MaxAmount:  req.MaxAmount,
		Currency:   req.Currency,
		Note:       req.Note,
		Negotiable: req.Negotiable,
	})
	if err != nil {
		handleFreelancePricingError(w, err)
		return
	}
	publishReindexBestEffort(r.Context(), h.searchPublish, orgID, search.PersonaFreelance, "freelance_pricing.upsert")
	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewFreelancePricingSummary(p),
	})
}

// DeleteMy removes the freelance pricing row.
func (h *FreelancePricingHandler) DeleteMy(w http.ResponseWriter, r *http.Request) {
	orgID, profileID, ok := h.resolveOrgAndProfile(w, r)
	if !ok {
		return
	}

	if err := h.pricing.Delete(r.Context(), profileID); err != nil {
		handleFreelancePricingError(w, err)
		return
	}
	publishReindexBestEffort(r.Context(), h.searchPublish, orgID, search.PersonaFreelance, "freelance_pricing.delete")
	res.NoContent(w)
}

// resolveProfile resolves the freelance profile id for the caller's
// org, writing the appropriate HTTP error and returning ok=false
// when anything goes wrong. Extracted so the three endpoints share
// a single failure path.
func (h *FreelancePricingHandler) resolveProfile(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	_, profileID, ok := h.resolveOrgAndProfile(w, r)
	return profileID, ok
}

// resolveOrgAndProfile returns both ids so the search publisher
// (which keys on org id) can fire after a successful mutation.
func (h *FreelancePricingHandler) resolveOrgAndProfile(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return uuid.Nil, uuid.Nil, false
	}
	profileID, err := h.profiles.GetFreelanceProfileIDByOrgID(r.Context(), orgID)
	if err != nil {
		handleFreelancePricingError(w, err)
		return uuid.Nil, uuid.Nil, false
	}
	return orgID, profileID, true
}

// handleFreelancePricingError maps domain-level errors to HTTP
// statuses. Kept pure so the mapping is unit-testable.
func handleFreelancePricingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domainpricing.ErrPricingNotFound):
		res.Error(w, http.StatusNotFound, "freelance_pricing_not_found", err.Error())
	case errors.Is(err, domainpricing.ErrInvalidType),
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
