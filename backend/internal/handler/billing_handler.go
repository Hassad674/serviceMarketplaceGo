package handler

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"

	paymentapp "marketplace-backend/internal/app/payment"
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
//
// ViewerIsProvider is the authoritative "should I render the preview?"
// flag. The UI hides the whole component when this is false so a client
// or enterprise user never sees the prestataire's fee structure.
type feePreviewResponse struct {
	AmountCents      int64            `json:"amount_cents"`
	FeeCents         int64            `json:"fee_cents"`
	NetCents         int64            `json:"net_cents"`
	Role             string           `json:"role"`
	ActiveTierIndex  int              `json:"active_tier_index"`
	Tiers            []feePreviewTier `json:"tiers"`
	ViewerIsProvider bool             `json:"viewer_is_provider"`
}

// GetFeePreview returns the fee schedule for the authenticated user along
// with the specific fee that applies to a milestone of the given amount,
// and a `viewer_is_provider` flag that tells the UI whether to render the
// preview at all (clients and enterprises never see platform fees).
//
// Query parameters:
//   - amount (required, integer cents, >= 0): the milestone amount.
//   - recipient_id (optional, UUID): the other party on the hypothetical
//     proposal. When provided, the backend runs proposal.DetermineRoles to
//     compute the authoritative provider_id and sets viewer_is_provider
//     accordingly. Callers that create a proposal against a specific user
//     MUST pass this so agency-vs-provider and agency-vs-enterprise
//     disambiguation happens server-side.
//
// The caller's role is read from the JWT context, never from the query.
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

	var recipientID *uuid.UUID
	if raw := r.URL.Query().Get("recipient_id"); raw != "" {
		parsed, pErr := uuid.Parse(raw)
		if pErr != nil {
			res.Error(w, http.StatusBadRequest, "invalid_recipient_id", "recipient_id must be a valid UUID")
			return
		}
		recipientID = &parsed
	}

	result, err := h.paymentSvc.PreviewFee(r.Context(), userID, amountCents, recipientID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "fee_preview_error", "could not compute fee preview")
		return
	}

	res.JSON(w, http.StatusOK, toFeePreviewResponse(result))
}

func toFeePreviewResponse(r *paymentapp.FeePreviewResult) feePreviewResponse {
	b := r.Billing
	tiers := make([]feePreviewTier, 0, len(b.Tiers))
	for _, t := range b.Tiers {
		tiers = append(tiers, feePreviewTier{
			Label:    t.Label,
			MaxCents: t.MaxCents,
			FeeCents: t.FeeCents,
		})
	}
	return feePreviewResponse{
		AmountCents:      b.AmountCents,
		FeeCents:         b.FeeCents,
		NetCents:         b.NetCents,
		Role:             string(b.Role),
		ActiveTierIndex:  b.ActiveTierIndex,
		Tiers:            tiers,
		ViewerIsProvider: r.ViewerIsProvider,
	}
}
