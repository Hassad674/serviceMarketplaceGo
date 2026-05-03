package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	domain "marketplace-backend/internal/domain/invoicing"
	"marketplace-backend/internal/handler/middleware"
	jsondec "marketplace-backend/pkg/decode"
	res "marketplace-backend/pkg/response"
)

// BillingProfileHandler exposes the four REST endpoints driving the
// "Mes informations de facturation" page: read / update / sync from
// Stripe KYC / VIES validate. Auth is enforced by the router; every
// handler reads the org id from JWT context — the user never supplies
// it as a path or body parameter (defense against cross-org leaks).
type BillingProfileHandler struct {
	svc *invoicingapp.Service
}

// NewBillingProfileHandler wires the handler to the invoicing app
// service. The service may be nil in tests that exercise the rest of
// the router; main.go wires a real service when the invoicing module
// is enabled (issuer config + PDF renderer both healthy).
func NewBillingProfileHandler(svc *invoicingapp.Service) *BillingProfileHandler {
	return &BillingProfileHandler{svc: svc}
}

// ---- DTOs (kept inline, matching subscription_handler.go convention) ----

type billingProfileResponse struct {
	Profile       billingProfileBody     `json:"profile"`
	MissingFields []domain.MissingField  `json:"missing_fields"`
	IsComplete    bool                   `json:"is_complete"`
}

type billingProfileBody struct {
	OrganizationID  string  `json:"organization_id"`
	ProfileType     string  `json:"profile_type"`
	LegalName       string  `json:"legal_name"`
	TradingName     string  `json:"trading_name"`
	LegalForm       string  `json:"legal_form"`
	TaxID           string  `json:"tax_id"`
	VATNumber       string  `json:"vat_number"`
	VATValidatedAt  *string `json:"vat_validated_at,omitempty"`
	AddressLine1    string  `json:"address_line1"`
	AddressLine2    string  `json:"address_line2"`
	PostalCode      string  `json:"postal_code"`
	City            string  `json:"city"`
	Country         string  `json:"country"`
	InvoicingEmail  string  `json:"invoicing_email"`
	SyncedFromKYCAt *string `json:"synced_from_kyc_at,omitempty"`
}

type updateBillingProfileRequest struct {
	ProfileType    string `json:"profile_type"`
	LegalName      string `json:"legal_name"`
	TradingName    string `json:"trading_name"`
	LegalForm      string `json:"legal_form"`
	TaxID          string `json:"tax_id"`
	VATNumber      string `json:"vat_number"`
	AddressLine1   string `json:"address_line1"`
	AddressLine2   string `json:"address_line2"`
	PostalCode     string `json:"postal_code"`
	City           string `json:"city"`
	Country        string `json:"country"`
	InvoicingEmail string `json:"invoicing_email"`
}

type viesValidationResponse struct {
	Valid          bool   `json:"valid"`
	RegisteredName string `json:"registered_name"`
	CheckedAt      string `json:"checked_at"`
}

// ---- handler methods ----

// GetMine — GET /api/v1/me/billing-profile
func (h *BillingProfileHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "invoicing_disabled", "invoicing feature not configured")
		return
	}
	orgID, ok := h.requireOrg(w, r)
	if !ok {
		return
	}
	snap, err := h.svc.GetBillingProfile(r.Context(), orgID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "billing_profile_read_error", err.Error())
		return
	}
	res.JSON(w, http.StatusOK, toBillingProfileResponse(snap))
}

// Update — PUT /api/v1/me/billing-profile
func (h *BillingProfileHandler) Update(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "invoicing_disabled", "invoicing feature not configured")
		return
	}
	orgID, ok := h.requireOrg(w, r)
	if !ok {
		return
	}
	var req updateBillingProfileRequest
	// F.5 B1: bound + reject unknown fields. Address + tax IDs payload
	// is small — 32 KiB is generous for the legitimate flow and rejects
	// the unbounded-body DoS surface.
	if err := jsondec.DecodeBody(w, r, &req, 32<<10); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "malformed JSON payload")
		return
	}
	snap, err := h.svc.UpdateBillingProfile(r.Context(), orgID, invoicingapp.UpdateBillingProfileInput{
		ProfileType:    req.ProfileType,
		LegalName:      req.LegalName,
		TradingName:    req.TradingName,
		LegalForm:      req.LegalForm,
		TaxID:          req.TaxID,
		VATNumber:      req.VATNumber,
		AddressLine1:   req.AddressLine1,
		AddressLine2:   req.AddressLine2,
		PostalCode:     req.PostalCode,
		City:           req.City,
		Country:        req.Country,
		InvoicingEmail: req.InvoicingEmail,
	})
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "billing_profile_update_error", err.Error())
		return
	}
	res.JSON(w, http.StatusOK, toBillingProfileResponse(snap))
}

// SyncFromStripe — POST /api/v1/me/billing-profile/sync-from-stripe
func (h *BillingProfileHandler) SyncFromStripe(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "invoicing_disabled", "invoicing feature not configured")
		return
	}
	orgID, ok := h.requireOrg(w, r)
	if !ok {
		return
	}
	snap, err := h.svc.SyncBillingProfileFromStripeKYC(r.Context(), orgID)
	if err != nil {
		switch {
		case errors.Is(err, invoicingapp.ErrBillingProfileFeatureDisabled):
			res.Error(w, http.StatusServiceUnavailable, "stripe_sync_disabled", "stripe sync not configured")
		default:
			res.Error(w, http.StatusInternalServerError, "stripe_sync_error", err.Error())
		}
		return
	}
	res.JSON(w, http.StatusOK, toBillingProfileResponse(snap))
}

// ValidateVAT — POST /api/v1/me/billing-profile/validate-vat
func (h *BillingProfileHandler) ValidateVAT(w http.ResponseWriter, r *http.Request) {
	if h.svc == nil {
		res.Error(w, http.StatusServiceUnavailable, "invoicing_disabled", "invoicing feature not configured")
		return
	}
	orgID, ok := h.requireOrg(w, r)
	if !ok {
		return
	}
	snap, err := h.svc.ValidateBillingProfileVAT(r.Context(), orgID)
	if err != nil {
		switch {
		case errors.Is(err, invoicingapp.ErrBillingProfileNoVAT):
			res.Error(w, http.StatusBadRequest, "vat_number_required", "the billing profile has no VAT number to validate")
		case errors.Is(err, domain.ErrNotFound):
			res.Error(w, http.StatusNotFound, "billing_profile_not_found", "billing profile not found")
		case errors.Is(err, invoicingapp.ErrBillingProfileFeatureDisabled):
			res.Error(w, http.StatusServiceUnavailable, "vies_disabled", "VIES validator not configured")
		default:
			// VIES network errors map to 502 — the upstream is what failed.
			res.Error(w, http.StatusBadGateway, "vies_unavailable", err.Error())
		}
		return
	}
	res.JSON(w, http.StatusOK, viesValidationResponse{
		Valid:          snap.Valid,
		RegisteredName: snap.RegisteredName,
		CheckedAt:      snap.CheckedAt.UTC().Format(time.RFC3339),
	})
}

// ---- helpers ----

func (h *BillingProfileHandler) requireOrg(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return uuid.Nil, false
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusForbidden, "no_organization", "user is not yet a member of any organization")
		return uuid.Nil, false
	}
	return orgID, true
}

func toBillingProfileResponse(snap invoicingapp.BillingProfileSnapshot) billingProfileResponse {
	body := billingProfileBody{
		OrganizationID: snap.Profile.OrganizationID.String(),
		ProfileType:    string(snap.Profile.ProfileType),
		LegalName:      snap.Profile.LegalName,
		TradingName:    snap.Profile.TradingName,
		LegalForm:      snap.Profile.LegalForm,
		TaxID:          snap.Profile.TaxID,
		VATNumber:      snap.Profile.VATNumber,
		AddressLine1:   snap.Profile.AddressLine1,
		AddressLine2:   snap.Profile.AddressLine2,
		PostalCode:     snap.Profile.PostalCode,
		City:           snap.Profile.City,
		Country:        snap.Profile.Country,
		InvoicingEmail: snap.Profile.InvoicingEmail,
	}
	if snap.Profile.VATValidatedAt != nil {
		t := snap.Profile.VATValidatedAt.UTC().Format(time.RFC3339)
		body.VATValidatedAt = &t
	}
	if snap.Profile.SyncedFromKYCAt != nil {
		t := snap.Profile.SyncedFromKYCAt.UTC().Format(time.RFC3339)
		body.SyncedFromKYCAt = &t
	}
	missing := snap.MissingFields
	if missing == nil {
		missing = []domain.MissingField{}
	}
	return billingProfileResponse{
		Profile:       body,
		MissingFields: missing,
		IsComplete:    snap.IsComplete,
	}
}
