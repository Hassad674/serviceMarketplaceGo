package handler

import (
	"io"
	"log/slog"
	"net/http"

	embeddedapp "marketplace-backend/internal/app/embedded"
	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	"marketplace-backend/internal/handler/dto/response"
	portservice "marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
)

type StripeHandler struct {
	paymentSvc       *paymentapp.Service
	proposalSvc      *proposalapp.Service
	embeddedNotifier *embeddedapp.Notifier // optional, nil = classic path only
	publishableKey   string
}

func NewStripeHandler(paymentSvc *paymentapp.Service, proposalSvc *proposalapp.Service, publishableKey string) *StripeHandler {
	return &StripeHandler{
		paymentSvc:     paymentSvc,
		proposalSvc:    proposalSvc,
		publishableKey: publishableKey,
	}
}

// WithEmbeddedNotifier attaches an embedded notifier to the handler so
// account.* webhooks emit rich notifications via the shared notification
// service. Call once at wiring time.
func (h *StripeHandler) WithEmbeddedNotifier(n *embeddedapp.Notifier) *StripeHandler {
	h.embeddedNotifier = n
	return h
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
		h.dispatchEmbeddedNotif(r, event)
	case "capability.updated",
		"account.application.authorized",
		"account.application.deauthorized",
		"account.external_account.created",
		"account.external_account.updated",
		"account.external_account.deleted":
		h.dispatchEmbeddedNotif(r, event)
	default:
		slog.Debug("unhandled stripe event", "type", event.Type)
	}

	// Always return 200 to Stripe to acknowledge receipt
	w.WriteHeader(http.StatusOK)
}

// dispatchEmbeddedNotif fans out a Stripe account snapshot to the embedded
// notifier (when wired). Best-effort: logs errors, never returns them to
// Stripe, otherwise Stripe retries our webhook which could spam users.
func (h *StripeHandler) dispatchEmbeddedNotif(r *http.Request, event *portservice.StripeWebhookEvent) {
	if h.embeddedNotifier == nil || event == nil || event.AccountSnapshot == nil {
		return
	}
	if err := h.embeddedNotifier.HandleAccountSnapshot(r.Context(), event.AccountSnapshot); err != nil {
		slog.Warn("embedded notifier: handle snapshot",
			"account_id", event.AccountSnapshot.AccountID,
			"event_type", event.Type,
			"error", err)
	}
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
