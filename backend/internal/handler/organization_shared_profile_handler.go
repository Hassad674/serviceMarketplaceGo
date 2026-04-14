package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// OrganizationSharedProfileHandler owns the endpoints that write
// the shared-profile columns directly on the organizations row:
// location, languages, photo. Lives in its own handler file (not
// on OrganizationHandler) to preserve the feature-isolation
// principle — deleting the shared-profile split means deleting
// this file + its wiring, nothing else.
//
// The handler uses the OrganizationSharedProfileWriter port, which
// the postgres OrganizationRepository satisfies directly. The
// optional Geocoder is reused from the legacy profile flow so the
// behavior matches byte-for-byte: when the client sends lat+lng
// they are trusted verbatim, otherwise the server best-effort
// geocodes from city/country.
type OrganizationSharedProfileHandler struct {
	writer   repository.OrganizationSharedProfileWriter
	geocoder service.Geocoder
}

// NewOrganizationSharedProfileHandler constructs the handler with
// the writer port. The geocoder is attached via WithGeocoder after
// construction (matches the pattern used by the legacy profile
// handler).
func NewOrganizationSharedProfileHandler(writer repository.OrganizationSharedProfileWriter) *OrganizationSharedProfileHandler {
	return &OrganizationSharedProfileHandler{writer: writer}
}

// WithGeocoder attaches the optional geocoder. Nil is a no-op.
func (h *OrganizationSharedProfileHandler) WithGeocoder(g service.Geocoder) *OrganizationSharedProfileHandler {
	if g != nil {
		h.geocoder = g
	}
	return h
}

// GetSharedProfile returns the shared-profile block for the
// authenticated user's organization.
func (h *OrganizationSharedProfileHandler) GetSharedProfile(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	shared, err := h.writer.GetSharedProfile(r.Context(), orgID)
	if err != nil {
		handleSharedProfileError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewOrganizationSharedProfileResponse(shared),
	})
}

// UpdateLocation writes the location block atomically. Mirrors the
// legacy profile UpdateLocation semantics: trust client-supplied
// lat/lng when both are present, otherwise best-effort geocode.
func (h *OrganizationSharedProfileHandler) UpdateLocation(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req request.UpdateOrganizationLocationRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	city := strings.TrimSpace(req.City)
	country := strings.ToUpper(strings.TrimSpace(req.CountryCode))
	if err := profile.ValidateCountryCode(country); err != nil {
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
		return
	}
	workMode := profile.NormalizeWorkModes(req.WorkMode)
	lat, lng := h.resolveCoordinates(r.Context(), orgID, city, country, req.Latitude, req.Longitude)

	if err := h.writer.UpdateSharedLocation(r.Context(), orgID, repository.SharedProfileLocationInput{
		City:           city,
		CountryCode:    country,
		Latitude:       lat,
		Longitude:      lng,
		WorkMode:       workMode,
		TravelRadiusKm: req.TravelRadiusKm,
	}); err != nil {
		handleSharedProfileError(w, err)
		return
	}
	h.writeCurrentShared(w, r, orgID)
}

// UpdateLanguages replaces the two language arrays atomically.
func (h *OrganizationSharedProfileHandler) UpdateLanguages(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req request.UpdateOrganizationLanguagesRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	pro := profile.NormalizeLanguageCodes(req.Professional)
	conv := profile.NormalizeLanguageCodes(req.Conversational)

	if err := h.writer.UpdateSharedLanguages(r.Context(), orgID, pro, conv); err != nil {
		handleSharedProfileError(w, err)
		return
	}
	h.writeCurrentShared(w, r, orgID)
}

// UpdatePhoto writes the photo_url column. Empty string is
// accepted — the caller is responsible for the upstream storage
// delete when clearing the photo.
func (h *OrganizationSharedProfileHandler) UpdatePhoto(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var req request.UpdateOrganizationPhotoRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if err := h.writer.UpdateSharedPhotoURL(r.Context(), orgID, strings.TrimSpace(req.PhotoURL)); err != nil {
		handleSharedProfileError(w, err)
		return
	}
	h.writeCurrentShared(w, r, orgID)
}

// writeCurrentShared fetches and writes the shared block after a
// mutation so the client receives the canonical post-write state
// in one roundtrip.
func (h *OrganizationSharedProfileHandler) writeCurrentShared(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) {
	shared, err := h.writer.GetSharedProfile(r.Context(), orgID)
	if err != nil {
		handleSharedProfileError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewOrganizationSharedProfileResponse(shared),
	})
}

// resolveCoordinates trusts client-supplied lat/lng when both are
// non-nil, otherwise falls back to the optional geocoder with a
// bounded sub-context. Any geocoding failure is logged at WARN and
// surfaces as nil pointers so the save proceeds without coordinates.
func (h *OrganizationSharedProfileHandler) resolveCoordinates(
	ctx context.Context,
	orgID uuid.UUID,
	city, country string,
	clientLat, clientLng *float64,
) (*float64, *float64) {
	if clientLat != nil && clientLng != nil {
		return clientLat, clientLng
	}
	if h.geocoder == nil || city == "" || country == "" {
		return nil, nil
	}
	gctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	la, ln, err := h.geocoder.Geocode(gctx, city, country)
	if err != nil {
		slog.Warn("geocoding failed for organization shared location",
			"org_id", orgID.String(),
			"city", city,
			"country", country,
			"error", err)
		return nil, nil
	}
	return &la, &ln
}

// handleSharedProfileError maps the small set of errors the shared-
// profile writer can surface.
func handleSharedProfileError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, organization.ErrOrgNotFound):
		res.Error(w, http.StatusNotFound, "organization_not_found", err.Error())
	case errors.Is(err, profile.ErrInvalidCountryCode):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
