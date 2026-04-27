package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	appsub "marketplace-backend/internal/app/subscription"
	domain "marketplace-backend/internal/domain/subscription"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// SubscriptionHandler groups the seven REST endpoints that drive the
// Premium flow: subscribe, get status, toggle auto-renew, change cycle,
// fetch stats, open portal. Auth is enforced by the router; every
// handler below resolves the caller's organization from the JWT user
// claim and passes organization_id — never user_id — to the app service.
type SubscriptionHandler struct {
	svc *appsub.Service
}

func NewSubscriptionHandler(svc *appsub.Service) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

// resolveActorOrg reads the JWT user_id from context, then asks the app
// service for that user's organization_id. Writes the appropriate HTTP
// error response on failure and returns ok=false so the caller can
// short-circuit. Centralised here so every endpoint handles "unauth" and
// "user without org" identically.
func (h *SubscriptionHandler) resolveActorOrg(w http.ResponseWriter, r *http.Request) (userID, orgID uuid.UUID, ok bool) {
	userID, authed := middleware.GetUserID(r.Context())
	if !authed {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return uuid.Nil, uuid.Nil, false
	}
	orgID, err := h.svc.ResolveActorOrganization(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidOrganization) {
			res.Error(w, http.StatusForbidden, "no_organization", "user is not yet a member of any organization")
			return uuid.Nil, uuid.Nil, false
		}
		res.Error(w, http.StatusInternalServerError, "org_resolve_error", err.Error())
		return uuid.Nil, uuid.Nil, false
	}
	return userID, orgID, true
}

// ---------- Request / response DTOs ----------

type subscribeRequest struct {
	Plan         string `json:"plan"`
	BillingCycle string `json:"billing_cycle"`
	AutoRenew    bool   `json:"auto_renew"`
}

type subscribeResponse struct {
	// ClientSecret is the Stripe Embedded Checkout session client_secret.
	// The web/mobile client mounts it via @stripe/react-stripe-js to
	// render the inline payment form.
	ClientSecret string `json:"client_secret"`
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
	userID, orgID, ok := h.resolveActorOrg(w, r)
	if !ok {
		return
	}

	var req subscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "malformed JSON payload")
		return
	}

	// NOTE: the billing-profile completeness gate that used to block
	// subscribe lives in the Embedded Checkout modal now (step 1 of
	// the inline UX collects + validates the profile via our
	// BillingProfileForm before transitioning to the Stripe payment
	// step). The wallet handler keeps its own server-side gate because
	// withdrawals don't go through any UI form.

	out, err := h.svc.Subscribe(r.Context(), appsub.SubscribeInput{
		OrganizationID: orgID,
		ActorUserID:    userID,
		Plan:           domain.Plan(req.Plan),
		BillingCycle:   domain.BillingCycle(req.BillingCycle),
		AutoRenew:      req.AutoRenew,
	})
	if err != nil {
		mapSubscribeError(w, err)
		return
	}
	if out == nil || out.ClientSecret == "" {
		// Defensive: surface a clean error instead of returning an
		// empty client_secret that leaves the embedded checkout
		// frozen. Should never happen under normal Stripe flows
		// (session.New either returns a populated session or errors).
		slog.Error("subscribe: empty client_secret from Stripe",
			"org_id", orgID,
			"user_id", userID,
			"plan", req.Plan,
			"cycle", req.BillingCycle)
		res.Error(w, http.StatusBadGateway, "stripe_empty_session", "Stripe returned an empty session — retry")
		return
	}
	slog.Info("subscribe: created embedded checkout session",
		"org_id", orgID,
		"client_secret_prefix", clientSecretPrefix(out.ClientSecret))
	res.JSON(w, http.StatusCreated, subscribeResponse{ClientSecret: out.ClientSecret})
}

// clientSecretPrefix returns a short public-safe prefix of the secret
// (first 12 chars) for diagnostic logging without leaking the full
// value.
func clientSecretPrefix(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12] + "..."
}

// GetMine — GET /api/v1/subscriptions/me
// Returns 404 when the org is on the free tier so the UI can
// differentiate "no subscription" from a real error.
func (h *SubscriptionHandler) GetMine(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := h.resolveActorOrg(w, r)
	if !ok {
		return
	}

	sub, err := h.svc.GetStatus(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			res.Error(w, http.StatusNotFound, "no_subscription", "organization has no active subscription")
			return
		}
		res.Error(w, http.StatusInternalServerError, "subscription_read_error", err.Error())
		return
	}
	res.JSON(w, http.StatusOK, toSubscriptionResponse(sub))
}

// ToggleAutoRenew — PATCH /api/v1/subscriptions/me/auto-renew
func (h *SubscriptionHandler) ToggleAutoRenew(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := h.resolveActorOrg(w, r)
	if !ok {
		return
	}

	var req toggleAutoRenewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "malformed JSON payload")
		return
	}

	sub, err := h.svc.ToggleAutoRenew(r.Context(), orgID, req.AutoRenew)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			res.Error(w, http.StatusNotFound, "no_subscription", "organization has no active subscription")
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
	_, orgID, ok := h.resolveActorOrg(w, r)
	if !ok {
		return
	}

	var req changeCycleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", "malformed JSON payload")
		return
	}

	sub, err := h.svc.ChangeCycle(r.Context(), orgID, domain.BillingCycle(req.BillingCycle))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			res.Error(w, http.StatusNotFound, "no_subscription", "organization has no active subscription")
		case errors.Is(err, domain.ErrInvalidCycle):
			res.Error(w, http.StatusBadRequest, "invalid_cycle", err.Error())
		case errors.Is(err, domain.ErrSameCycle):
			res.Error(w, http.StatusConflict, "same_cycle", err.Error())
		case errors.Is(err, domain.ErrAutoRenewOffBlocksDowngrade):
			res.Error(w, http.StatusConflict, "auto_renew_required", err.Error())
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
	userID, orgID, ok := h.resolveActorOrg(w, r)
	if !ok {
		return
	}

	stats, err := h.svc.GetStats(r.Context(), orgID, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			res.Error(w, http.StatusNotFound, "no_subscription", "organization has no active subscription")
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
	_, orgID, ok := h.resolveActorOrg(w, r)
	if !ok {
		return
	}

	newCycleRaw := r.URL.Query().Get("billing_cycle")
	if newCycleRaw == "" {
		res.Error(w, http.StatusBadRequest, "missing_billing_cycle", "billing_cycle query parameter is required")
		return
	}

	preview, err := h.svc.PreviewCycleChange(r.Context(), orgID, domain.BillingCycle(newCycleRaw))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			res.Error(w, http.StatusNotFound, "no_subscription", "organization has no active subscription")
		case errors.Is(err, domain.ErrInvalidCycle):
			res.Error(w, http.StatusBadRequest, "invalid_cycle", err.Error())
		case errors.Is(err, domain.ErrSameCycle):
			res.Error(w, http.StatusConflict, "same_cycle", err.Error())
		case errors.Is(err, domain.ErrAutoRenewOffBlocksDowngrade):
			res.Error(w, http.StatusConflict, "auto_renew_required", err.Error())
		default:
			res.Error(w, http.StatusInternalServerError, "preview_error", err.Error())
		}
		return
	}

	// ProrateImmediately comes from the service (direction-derived),
	// NOT inferred from the amount: Stripe's invoices.upcoming returns
	// the NEXT invoice for downgrades (e.g. the first monthly charge
	// that will hit at the phase boundary), which would mislead the UI
	// into labelling it as "charged today" if we used amount > 0.
	res.JSON(w, http.StatusOK, cyclePreviewResponse{
		AmountDueCents:     preview.AmountDueCents,
		Currency:           preview.Currency,
		PeriodStart:        preview.PeriodStart.UTC().Format("2006-01-02T15:04:05Z"),
		PeriodEnd:          preview.PeriodEnd.UTC().Format("2006-01-02T15:04:05Z"),
		ProrateImmediately: preview.ProrateImmediately,
	})
}

// GetPortal — GET /api/v1/subscriptions/portal
// Returns a short-lived URL to the Stripe Customer Portal so the user
// can update their payment method or view invoices without ever leaving
// Stripe's PCI-compliant environment.
func (h *SubscriptionHandler) GetPortal(w http.ResponseWriter, r *http.Request) {
	_, orgID, ok := h.resolveActorOrg(w, r)
	if !ok {
		return
	}

	url, err := h.svc.GetPortalURL(r.Context(), orgID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			res.Error(w, http.StatusNotFound, "no_subscription", "organization has no active subscription")
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
