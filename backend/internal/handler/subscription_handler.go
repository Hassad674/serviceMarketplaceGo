package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	appsub "marketplace-backend/internal/app/subscription"
	domain "marketplace-backend/internal/domain/subscription"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// SubscriptionHandler groups the seven REST endpoints that drive the
// Premium flow: subscribe, get status, toggle auto-renew, change cycle,
// fetch stats, open portal. Auth is enforced by the router; every
// handler below assumes a non-empty userID in the context.
type SubscriptionHandler struct {
	svc *appsub.Service
}

func NewSubscriptionHandler(svc *appsub.Service) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

// ---------- Request / response DTOs ----------

type subscribeRequest struct {
	Plan         string `json:"plan"`
	BillingCycle string `json:"billing_cycle"`
	AutoRenew    bool   `json:"auto_renew"`
}

type subscribeResponse struct {
	CheckoutURL string `json:"checkout_url"`
}

type subscriptionResponse struct {
	ID                 string  `json:"id"`
	Plan               string  `json:"plan"`
	BillingCycle       string  `json:"billing_cycle"`
	Status             string  `json:"status"`
	CurrentPeriodStart string  `json:"current_period_start"`
	CurrentPeriodEnd   string  `json:"current_period_end"`
	CancelAtPeriodEnd  bool    `json:"cancel_at_period_end"`
	StartedAt          string  `json:"started_at"`
	GracePeriodEndsAt  *string `json:"grace_period_ends_at,omitempty"`
	CanceledAt         *string `json:"canceled_at,omitempty"`
	// PendingBillingCycle + PendingCycleEffectiveAt describe a scheduled
	// downgrade. Populated together or both omitted (mirroring the DB
	// CHECK constraint). The UI renders "Annuel jusqu'au JJ/MM/YYYY →
	// Mensuel ensuite" when pending_billing_cycle is present.
	PendingBillingCycle     *string `json:"pending_billing_cycle,omitempty"`
	PendingCycleEffectiveAt *string `json:"pending_cycle_effective_at,omitempty"`
}

// cyclePreviewResponse mirrors service.InvoicePreview on the wire.
type cyclePreviewResponse struct {
	AmountDueCents int64  `json:"amount_due_cents"`
	Currency       string `json:"currency"`
	PeriodStart    string `json:"period_start"`
	PeriodEnd      string `json:"period_end"`
	// ProrateImmediately flags whether the amount is charged today
	// (upgrade) or at the next renewal (downgrade → always 0 today).
	ProrateImmediately bool `json:"prorate_immediately"`
}

type toggleAutoRenewRequest struct {
	// AutoRenew = true means "keep charging me at each renewal". Maps to
	// cancel_at_period_end = false on the Stripe subscription.
	AutoRenew bool `json:"auto_renew"`
}

type changeCycleRequest struct {
	BillingCycle string `json:"billing_cycle"`
}

type statsResponse struct {
	SavedFeeCents int64  `json:"saved_fee_cents"`
	SavedCount    int    `json:"saved_count"`
	Since         string `json:"since"`
}

type portalResponse struct {
	URL string `json:"url"`
}

// ---------- Handlers ----------

// Subscribe — POST /api/v1/subscriptions
func (h *SubscriptionHandler) Subscribe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "malformed JSON payload")
		return
	}

	out, err := h.svc.Subscribe(r.Context(), appsub.SubscribeInput{
		UserID:       userID,
		Plan:         domain.Plan(req.Plan),
		BillingCycle: domain.BillingCycle(req.BillingCycle),
		AutoRenew:    req.AutoRenew,
	})
	if err != nil {
		mapSubscribeError(w, err)
		return
	}
	res.JSON(w, http.StatusCreated, subscribeResponse{CheckoutURL: out.CheckoutURL})
}

// GetMine — GET /api/v1/subscriptions/me
// Returns 404 when the user is on the free tier so the UI can
// differentiate "no subscription" from a real error.
func (h *SubscriptionHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	sub, err := h.svc.GetStatus(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			res.Error(w, http.StatusNotFound, "no_subscription", "user has no active subscription")
			return
		}
		res.Error(w, http.StatusInternalServerError, "subscription_read_error", err.Error())
		return
	}
	res.JSON(w, http.StatusOK, toSubscriptionResponse(sub))
}

// ToggleAutoRenew — PATCH /api/v1/subscriptions/me/auto-renew
func (h *SubscriptionHandler) ToggleAutoRenew(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req toggleAutoRenewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "malformed JSON payload")
		return
	}

	sub, err := h.svc.ToggleAutoRenew(r.Context(), userID, req.AutoRenew)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			res.Error(w, http.StatusNotFound, "no_subscription", "user has no active subscription")
			return
		}
		res.Error(w, http.StatusInternalServerError, "subscription_update_error", err.Error())
		return
	}
	res.JSON(w, http.StatusOK, toSubscriptionResponse(sub))
}

// ChangeCycle — PATCH /api/v1/subscriptions/me/billing-cycle
// Body carries the NEW billing_cycle; both directions monthly↔annual are
// supported with immediate Stripe-handled proration.
func (h *SubscriptionHandler) ChangeCycle(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req changeCycleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "malformed JSON payload")
		return
	}

	sub, err := h.svc.ChangeCycle(r.Context(), userID, domain.BillingCycle(req.BillingCycle))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			res.Error(w, http.StatusNotFound, "no_subscription", "user has no active subscription")
		case errors.Is(err, domain.ErrInvalidCycle):
			res.Error(w, http.StatusBadRequest, "invalid_cycle", err.Error())
		case errors.Is(err, domain.ErrSameCycle):
			res.Error(w, http.StatusConflict, "same_cycle", err.Error())
		case errors.Is(err, domain.ErrInvalidTransition):
			res.Error(w, http.StatusConflict, "invalid_state", err.Error())
		default:
			res.Error(w, http.StatusInternalServerError, "subscription_update_error", err.Error())
		}
		return
	}
	res.JSON(w, http.StatusOK, toSubscriptionResponse(sub))
}

// GetStats — GET /api/v1/subscriptions/me/stats
func (h *SubscriptionHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	stats, err := h.svc.GetStats(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			res.Error(w, http.StatusNotFound, "no_subscription", "user has no active subscription")
			return
		}
		res.Error(w, http.StatusInternalServerError, "stats_error", err.Error())
		return
	}
	res.JSON(w, http.StatusOK, statsResponse{
		SavedFeeCents: stats.SavedFeeCents,
		SavedCount:    stats.SavedCount,
		Since:         stats.Since.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

// PreviewCycleChange — GET /api/v1/subscriptions/me/cycle-preview?billing_cycle=X
// Side-effect free — returns the invoice preview (amount charged today +
// next period) so the manage modal can render an accurate confirm step.
func (h *SubscriptionHandler) PreviewCycleChange(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	newCycleRaw := r.URL.Query().Get("billing_cycle")
	if newCycleRaw == "" {
		res.Error(w, http.StatusBadRequest, "missing_billing_cycle", "billing_cycle query parameter is required")
		return
	}

	preview, err := h.svc.PreviewCycleChange(r.Context(), userID, domain.BillingCycle(newCycleRaw))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			res.Error(w, http.StatusNotFound, "no_subscription", "user has no active subscription")
		case errors.Is(err, domain.ErrInvalidCycle):
			res.Error(w, http.StatusBadRequest, "invalid_cycle", err.Error())
		case errors.Is(err, domain.ErrSameCycle):
			res.Error(w, http.StatusConflict, "same_cycle", err.Error())
		default:
			res.Error(w, http.StatusInternalServerError, "preview_error", err.Error())
		}
		return
	}

	// AmountDueCents > 0 implies the call would charge today (upgrade).
	// Downgrades are always 0 today (scheduled for next renewal).
	res.JSON(w, http.StatusOK, cyclePreviewResponse{
		AmountDueCents:     preview.AmountDueCents,
		Currency:           preview.Currency,
		PeriodStart:        preview.PeriodStart.UTC().Format("2006-01-02T15:04:05Z"),
		PeriodEnd:          preview.PeriodEnd.UTC().Format("2006-01-02T15:04:05Z"),
		ProrateImmediately: preview.AmountDueCents > 0,
	})
}

// GetPortal — GET /api/v1/subscriptions/portal
// Returns a short-lived URL to the Stripe Customer Portal so the user
// can update their payment method or view invoices without ever leaving
// Stripe's PCI-compliant environment.
func (h *SubscriptionHandler) GetPortal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	url, err := h.svc.GetPortalURL(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			res.Error(w, http.StatusNotFound, "no_subscription", "user has no active subscription")
			return
		}
		res.Error(w, http.StatusInternalServerError, "portal_error", err.Error())
		return
	}
	res.JSON(w, http.StatusOK, portalResponse{URL: url})
}

// ---------- helpers ----------

func toSubscriptionResponse(s *domain.Subscription) subscriptionResponse {
	out := subscriptionResponse{
		ID:                 s.ID.String(),
		Plan:               string(s.Plan),
		BillingCycle:       string(s.BillingCycle),
		Status:             string(s.Status),
		CurrentPeriodStart: s.CurrentPeriodStart.UTC().Format("2006-01-02T15:04:05Z"),
		CurrentPeriodEnd:   s.CurrentPeriodEnd.UTC().Format("2006-01-02T15:04:05Z"),
		CancelAtPeriodEnd:  s.CancelAtPeriodEnd,
		StartedAt:          s.StartedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
	if s.GracePeriodEndsAt != nil {
		formatted := s.GracePeriodEndsAt.UTC().Format("2006-01-02T15:04:05Z")
		out.GracePeriodEndsAt = &formatted
	}
	if s.CanceledAt != nil {
		formatted := s.CanceledAt.UTC().Format("2006-01-02T15:04:05Z")
		out.CanceledAt = &formatted
	}
	if s.PendingBillingCycle != nil {
		cycle := string(*s.PendingBillingCycle)
		out.PendingBillingCycle = &cycle
	}
	if s.PendingCycleEffectiveAt != nil {
		formatted := s.PendingCycleEffectiveAt.UTC().Format("2006-01-02T15:04:05Z")
		out.PendingCycleEffectiveAt = &formatted
	}
	return out
}

func mapSubscribeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidPlan):
		res.Error(w, http.StatusBadRequest, "invalid_plan", err.Error())
	case errors.Is(err, domain.ErrInvalidCycle):
		res.Error(w, http.StatusBadRequest, "invalid_cycle", err.Error())
	case errors.Is(err, domain.ErrAlreadySubscribed):
		res.Error(w, http.StatusConflict, "already_subscribed", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "subscribe_error", err.Error())
	}
}
