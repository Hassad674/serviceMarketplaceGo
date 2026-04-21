package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	clientprofileapp "marketplace-backend/internal/app/clientprofile"
	profileapp "marketplace-backend/internal/app/profile"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// ClientProfileHandler wires the client-profile endpoints. It stays
// separate from ProfileHandler so the feature is fully removable —
// deleting this file plus its three wiring lines (main.go + router.go)
// takes the client-profile surface down without touching the legacy
// provider-profile flow.
type ClientProfileHandler struct {
	write *profileapp.ClientProfileService
	read  *clientprofileapp.Service
}

// NewClientProfileHandler constructs the handler with its write (app/
// profile.ClientProfileService) and read (app/clientprofile.Service)
// dependencies. Both are required — there is no "read-only" or
// "write-only" deployment of the feature.
func NewClientProfileHandler(write *profileapp.ClientProfileService, read *clientprofileapp.Service) *ClientProfileHandler {
	return &ClientProfileHandler{write: write, read: read}
}

// UpdateMyClientProfile serves PUT /api/v1/profile/client. Gated by
// middleware.RequirePermission(PermOrgClientProfileEdit) at the
// router level — this method trusts the context and validates the
// payload at the service layer.
func (h *ClientProfileHandler) UpdateMyClientProfile(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	var body request.UpdateClientProfileRequest
	if err := validator.DecodeJSON(r, &body); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	updated, err := h.write.UpdateClientProfile(r.Context(), orgID, profileapp.UpdateClientProfileInput{
		CompanyName:       body.CompanyName,
		ClientDescription: body.ClientDescription,
	})
	if err != nil {
		handleClientProfileWriteError(w, err)
		return
	}

	// Build the response without the Client stats block — the caller
	// already holds those via the separate GET /api/v1/profile path.
	// Keeping this response shape tight avoids extra DB round-trips
	// on every write.
	res.JSON(w, http.StatusOK, response.NewProfileResponseWithExtras(updated, nil, nil, nil))
}

// GetPublicClientProfile serves GET /api/v1/clients/{orgId}. Public
// endpoint — no authentication middleware. A missing or
// non-exposed org (e.g. provider_personal in v1) returns 404 so a
// probe cannot discover which orgs are agencies vs. freelancers
// without hitting the canonical search endpoints.
func (h *ClientProfileHandler) GetPublicClientProfile(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "orgId"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "orgId must be a valid UUID")
		return
	}

	p, err := h.read.GetPublicClientProfile(r.Context(), orgID)
	if err != nil {
		handleClientProfileReadError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewPublicClientProfileResponse(p))
}

// handleClientProfileWriteError maps domain-level errors to the
// stable HTTP status table.
func handleClientProfileWriteError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, profile.ErrForbiddenOrgType):
		res.Error(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, profile.ErrClientDescriptionTooLong):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, organization.ErrNameRequired):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	case errors.Is(err, profile.ErrProfileNotFound),
		errors.Is(err, organization.ErrOrgNotFound):
		res.Error(w, http.StatusNotFound, "profile_not_found", err.Error())
	default:
		slog.Error("update client profile", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

// handleClientProfileReadError maps the narrow error surface of the
// public read — in v1 the only expected non-500 outcome is "this org
// is not exposed", which we map to 404 per the package doc comment.
func handleClientProfileReadError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, profile.ErrProfileNotFound),
		errors.Is(err, organization.ErrOrgNotFound):
		res.Error(w, http.StatusNotFound, "profile_not_found", "client profile not found")
	default:
		slog.Error("get public client profile", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
