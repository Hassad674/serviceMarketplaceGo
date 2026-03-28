package handler

import (
	"io"
	"log/slog"
	"net/http"

	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	"marketplace-backend/internal/handler/dto/response"
	res "marketplace-backend/pkg/response"
)

type StripeHandler struct {
	paymentSvc     *paymentapp.Service
	proposalSvc    *proposalapp.Service
	publishableKey string
}

func NewStripeHandler(paymentSvc *paymentapp.Service, proposalSvc *proposalapp.Service, publishableKey string) *StripeHandler {
	return &StripeHandler{
		paymentSvc:     paymentSvc,
		proposalSvc:    proposalSvc,
		publishableKey: publishableKey,
	}
}

// GetConfig returns the Stripe publishable key for frontend initialization.
func (h *StripeHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	res.JSON(w, http.StatusOK, response.StripeConfigResponse{
		PublishableKey: h.publishableKey,
	})
}

// HandleWebhook processes Stripe webhook events.
// No auth middleware — Stripe sends directly, verified by signature.
func (h *StripeHandler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "read_error", "cannot read request body")
		return
	}

	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		res.Error(w, http.StatusBadRequest, "missing_signature", "Stripe-Signature header required")
		return
	}

	event, err := h.paymentSvc.VerifyWebhook(body, signature)
	if err != nil {
		slog.Error("stripe webhook signature verification failed", "error", err)
		res.Error(w, http.StatusBadRequest, "invalid_signature", "webhook signature verification failed")
		return
	}

	switch event.Type {
	case "payment_intent.succeeded":
		h.handlePaymentSucceeded(r, event.PaymentIntentID)
	case "account.updated":
		h.handleAccountUpdated(r, event.AccountID)
	default:
		slog.Debug("unhandled stripe event", "type", event.Type)
	}

	// Always return 200 to Stripe to acknowledge receipt
	w.WriteHeader(http.StatusOK)
}

func (h *StripeHandler) handlePaymentSucceeded(r *http.Request, piID string) {
	proposalID, err := h.paymentSvc.HandlePaymentSucceeded(r.Context(), piID)
	if err != nil {
		slog.Error("handle payment succeeded", "payment_intent", piID, "error", err)
		return
	}

	if err := h.proposalSvc.ConfirmPaymentAndActivate(r.Context(), proposalID); err != nil {
		slog.Error("confirm payment and activate", "proposal_id", proposalID, "error", err)
	}
}

func (h *StripeHandler) handleAccountUpdated(r *http.Request, accountID string) {
	if err := h.paymentSvc.HandleAccountUpdated(r.Context(), accountID); err != nil {
		slog.Error("handle account updated", "account_id", accountID, "error", err)
	}
}
