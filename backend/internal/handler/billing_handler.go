package handler

import (
	"net/http"
	"strconv"

	paymentapp "marketplace-backend/internal/app/payment"
	"marketplace-backend/internal/domain/billing"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// BillingHandler exposes read-only billing endpoints used by the proposal
// creation flow on web and mobile. Write paths (subscribe, cancel, invoice
// download) will land in Phase B and will reuse this handler struct.
type BillingHandler struct {
	paymentSvc *paymentapp.Service
}

func NewBillingHandler(paymentSvc *paymentapp.Service) *BillingHandler {
	return &BillingHandler{paymentSvc: paymentSvc}
}

// feePreviewTier is the JSON shape of a single tier bracket.
// MaxCents is omitted (sent as `null` in JSON when pointer is nil) for the
// open-ended last tier so the client can render "Plus de X €" without
// hardcoding sentinel values.
type feePreviewTier struct {
	Label    string `json:"label"`
	MaxCents *int64 `json:"max_cents"`
	FeeCents int64  `json:"fee_cents"`
}

// feePreviewResponse mirrors billing.Result with JSON-friendly field names
// so the web/mobile clients can consume it without transformation.
type feePreviewResponse struct {
	AmountCents     int64            `json:"amount_cents"`
	FeeCents        int64            `json:"fee_cents"`
	NetCents        int64            `json:"net_cents"`
	Role            string           `json:"role"`
	ActiveTierIndex int              `json:"active_tier_index"`
	Tiers           []feePreviewTier `json:"tiers"`
}

// GetFeePreview returns the fee schedule for the authenticated user along
// with the specific fee that applies to a milestone of the given amount.
//
// Query parameters:
//   - amount (required, integer cents, >= 0): the milestone amount.
//
// The user's role is read from the JWT context, not the query string —
// never trust a client-supplied role. An enterprise or admin querying this
// endpoint will see the freelance grid (the `RoleFromUser` fallback),
// which is harmless: they cannot create proposals as a prestataire anyway.
func (h *BillingHandler) GetFeePreview(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	amountRaw := r.URL.Query().Get("amount")
	if amountRaw == "" {
		res.Error(w, http.StatusBadRequest, "missing_amount", "amount query parameter is required")
		return
	}
	amountCents, err := strconv.ParseInt(amountRaw, 10, 64)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_amount", "amount must be a valid integer in cents")
		return
	}
	if amountCents < 0 {
		res.Error(w, http.StatusBadRequest, "invalid_amount", "amount must be zero or positive")
		return
	}

	result, err := h.paymentSvc.PreviewFee(r.Context(), userID, amountCents)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "fee_preview_error", "could not compute fee preview")
		return
	}

	res.JSON(w, http.StatusOK, toFeePreviewResponse(result))
}

func toFeePreviewResponse(r *billing.Result) feePreviewResponse {
	tiers := make([]feePreviewTier, 0, len(r.Tiers))
	for _, t := range r.Tiers {
		tiers = append(tiers, feePreviewTier{
			Label:    t.Label,
			MaxCents: t.MaxCents,
			FeeCents: t.FeeCents,
		})
	}
	return feePreviewResponse{
		AmountCents:     r.AmountCents,
		FeeCents:        r.FeeCents,
		NetCents:        r.NetCents,
		Role:            string(r.Role),
		ActiveTierIndex: r.ActiveTierIndex,
		Tiers:           tiers,
	}
}
