package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	referralapp "marketplace-backend/internal/app/referral"
	domaininv "marketplace-backend/internal/domain/invoicing"
	paymentdomain "marketplace-backend/internal/domain/payment"
	referraldomain "marketplace-backend/internal/domain/referral"
	"marketplace-backend/internal/handler/middleware"
	portservice "marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
)

// payoutReadinessProbe is the narrow contract the wallet payout gate
// needs to ask "is this provider's Stripe Connect account ready to
// receive a transfer?". The full payment service satisfies it, but the
// handler only depends on this segregated method so tests can pass a
// 5-line fake without standing up the entire payment stack.
type payoutReadinessProbe interface {
	CanProviderReceivePayouts(ctx context.Context, providerOrgID uuid.UUID) (bool, error)
}

// transferRetrier is the narrow contract the wallet retry endpoint
// depends on so tests can drive every error branch (404 / 409 / 412 /
// 502) without standing up the full payment service. The real
// *paymentapp.Service satisfies it natively.
type transferRetrier interface {
	RetryFailedTransfer(ctx context.Context, userID, orgID, recordID uuid.UUID) (*paymentapp.PayoutResult, error)
}

// commissionRetrier is the narrow contract the wallet commission retry
// endpoint depends on so tests can drive every error branch (404 /
// 403 / 409 / 422 / 502) without standing up the full referral service.
// The real *referralapp.Service satisfies it via
// service.ReferralCommissionRetryService.
type commissionRetrier interface {
	RetryCommission(ctx context.Context, requestingUserID, commissionID uuid.UUID) (portservice.ReferralCommissionRetryOutcome, error)
}

// kycOnboardingURLResolver returns the Stripe Connect onboarding URL
// the apporteur should visit to finish KYC before retrying. Optional —
// when nil, the 422 response carries no `onboarding_url` field and
// the UI falls back to a generic /payment-info redirect.
type kycOnboardingURLResolver interface {
	GetOnboardingURL(ctx context.Context, userID uuid.UUID) (string, error)
}

// walletAuditLogger is the narrow append-only audit contract used by
// the unified /wallet/withdraw endpoint. Defined locally so the
// handler does not pull the entire audit.Entry constructor / repo
// from the test surface — the real *audit.AuditRepository implements
// the contract by delegating to its existing Log method.
type walletAuditLogger interface {
	Log(ctx context.Context, entry *auditEntry) error
}

// auditEntry is the field-level projection of audit.Entry used by
// the wallet handler so wiring stays minimal. The bootstrap maps it
// to domain audit.Entry before persisting (see WithAuditLogger).
type auditEntry struct {
	UserID       uuid.UUID
	OrgID        uuid.UUID
	Action       string
	ResourceType string
	Metadata     map[string]any
}

type WalletHandler struct {
	paymentSvc  *paymentapp.Service
	proposalSvc *proposalapp.Service
	// invoicingSvc is the optional gate the RequestPayout endpoint
	// uses to enforce billing-profile completeness BEFORE handing off
	// to Stripe. Nil = invoicing module disabled, in which case the
	// gate degrades open (the action is allowed) — invoicing is a
	// removable feature and removing it must never block payouts.
	invoicingSvc *invoicingapp.Service
	// kycProbe gates the RequestPayout endpoint on Stripe Connect
	// readiness BEFORE the billing-profile gate. The two failures are
	// surfaced with mutually-exclusive 403 codes so the frontend can
	// route the user to the right page (KYC missing → /payment-info,
	// billing missing → completion modal). Nil → KYC gate degrades
	// open and the request falls through to the billing gate.
	kycProbe payoutReadinessProbe
	// retrier is the narrow contract used by RetryFailedTransfer. We
	// hold it as an interface so handler tests can drive every error
	// branch without instantiating the real payment service. Defaulted
	// to paymentSvc in NewWalletHandler.
	retrier transferRetrier

	// commissionRetrier is the narrow contract used by RetryCommission
	// (D1+D2). Wired via WithCommissionRetrier — when nil the route
	// is registered but every request returns 503 retry_unavailable.
	commissionRetrier commissionRetrier

	// kycOnboardingURL resolves the Stripe Connect onboarding URL so
	// the 422 kyc_required response can carry a deep-link for the
	// frontend. Wired via WithKYCOnboardingURLResolver. When nil the
	// 422 response omits the URL and the UI falls back to its
	// generic /payment-info redirect.
	kycOnboardingURL kycOnboardingURLResolver

	// Run B (WALLET-UNIFY) — extra deps for the new /wallet/summary
	// and /wallet/withdraw endpoints. Each is wired via a builder
	// method so worktrees without the referral feature still boot —
	// degraded mode is "no projections / no commission drain"
	// rather than 500.
	commissionProjector commissionProjector
	commissionRecorder  commissionRecorder
	auditLogger         walletAuditLogger
}

func NewWalletHandler(paymentSvc *paymentapp.Service, proposalSvc *proposalapp.Service) *WalletHandler {
	h := &WalletHandler{paymentSvc: paymentSvc, proposalSvc: proposalSvc}
	// Default the KYC probe and the retry transport to the payment
	// service when wired — the payment service satisfies both narrow
	// interfaces natively. Tests that pass nil here can still inject
	// custom fakes via WithPayoutReadinessProbe / WithTransferRetrier.
	if paymentSvc != nil {
		h.kycProbe = paymentSvc
		h.retrier = paymentSvc
	}
	return h
}

// WithInvoicing wires the invoicing gate. Builder pattern keeps the
// constructor signature stable so a worktree without invoicing wired in
// still boots — and removing the invoicing feature is a single-line edit
// in main.go.
func (h *WalletHandler) WithInvoicing(svc *invoicingapp.Service) *WalletHandler {
	h.invoicingSvc = svc
	return h
}

// WithPayoutReadinessProbe overrides the default KYC readiness probe.
// Used by tests to inject a fake without the real payment service —
// production code path is auto-wired in NewWalletHandler.
func (h *WalletHandler) WithPayoutReadinessProbe(probe payoutReadinessProbe) *WalletHandler {
	h.kycProbe = probe
	return h
}

// WithTransferRetrier overrides the default payment-service-backed
// retry handler. Used by tests to drive every error branch of
// RetryFailedTransfer without standing up the full payment stack.
func (h *WalletHandler) WithTransferRetrier(r transferRetrier) *WalletHandler {
	h.retrier = r
	return h
}

// WithCommissionRetrier wires the D1+D2 commission retry service.
// Builder pattern keeps the constructor signature stable — a worktree
// without the referral feature still boots, the commission retry
// endpoint simply returns 503 retry_unavailable until wired.
func (h *WalletHandler) WithCommissionRetrier(r commissionRetrier) *WalletHandler {
	h.commissionRetrier = r
	return h
}

// WithKYCOnboardingURLResolver wires the helper that returns the
// Stripe Connect onboarding URL so the 422 kyc_required response can
// embed a deep-link for the frontend modal. Optional — when not wired
// the response omits the URL and the UI falls back to a generic redirect.
func (h *WalletHandler) WithKYCOnboardingURLResolver(r kycOnboardingURLResolver) *WalletHandler {
	h.kycOnboardingURL = r
	return h
}

// WithAuditLogger wires the narrow audit logger used by the unified
// /wallet/withdraw endpoint to record successful drains. Optional —
// when nil the endpoint still works, audit emission is silently
// skipped. Builder pattern keeps the constructor stable.
func (h *WalletHandler) WithAuditLogger(l walletAuditLogger) *WalletHandler {
	h.auditLogger = l
	return h
}

// respondKYCIncomplete writes the 403 envelope for the KYC pre-check
// on the wallet payout flow. The `redirect` field tells the frontend
// which page to send the user to so they can finish their Stripe
// onboarding before retrying the withdrawal. The `code` is the
// discriminator the frontend uses to differentiate this gate from
// the billing-profile gate.
func respondKYCIncomplete(w http.ResponseWriter) {
	res.JSON(w, http.StatusForbidden, map[string]any{
		"error": map[string]string{
			"code":    "kyc_incomplete",
			"message": "Termine d'abord ton onboarding Stripe sur la page Infos paiement avant de pouvoir retirer.",
		},
		"redirect": "/payment-info",
	})
}

// respondBillingProfileIncomplete writes the canonical 403 envelope
// shared between the wallet payout and the subscription subscribe
// gates. The shape mirrors what the frontend's "completion modal"
// expects: a discriminator code + the missing-fields list.
func respondBillingProfileIncomplete(w http.ResponseWriter, missing []domaininv.MissingField, message string) {
	if missing == nil {
		missing = []domaininv.MissingField{}
	}
	res.JSON(w, http.StatusForbidden, map[string]any{
		"error": map[string]string{
			"code":    "billing_profile_incomplete",
			"message": message,
		},
		"missing_fields": missing,
	})
}

// GetWallet returns wallet overview with proposal statuses.
func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	wallet, err := h.paymentSvc.GetWalletOverview(r.Context(), userID, orgID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	// Enrich records with proposal status and recompute available vs escrow
	wallet.EscrowAmount = 0
	wallet.AvailableAmount = 0
	for i := range wallet.Records {
		rec := &wallet.Records[i]
		proposalID, parseErr := uuid.Parse(rec.ProposalID)
		if parseErr != nil {
			continue
		}
		p, pErr := h.proposalSvc.GetProposalByID(r.Context(), proposalID)
		if pErr == nil && p != nil {
			rec.MissionStatus = string(p.Status)
		}
		if rec.PaymentStatus == "succeeded" && rec.TransferStatus == "pending" {
			if rec.MissionStatus == "completed" {
				wallet.AvailableAmount += rec.ProviderPayout
			} else {
				wallet.EscrowAmount += rec.ProviderPayout
			}
		}
	}

	res.JSON(w, http.StatusOK, wallet)
}

// RequestPayout triggers transfers only for completed missions.
func (h *WalletHandler) RequestPayout(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	// KYC gate: a payout cannot succeed unless the provider's Stripe
	// Connect account has payouts enabled. We block here BEFORE the
	// billing-profile gate so the user fixes their actual blocker —
	// the alternative is to make them complete a billing profile only
	// to fail later on the Stripe side with no actionable message.
	//
	// Errors during the probe are logged and treated as not-ready
	// (fail-closed): if we cannot prove the account is payable, the
	// payout would have failed downstream anyway, and a clear "finish
	// your KYC" message is friendlier than a 500.
	//
	// When the probe is nil (e.g. payment service not wired in a
	// minimal worktree), the gate degrades open and the request falls
	// through to the billing-profile gate.
	if h.kycProbe != nil {
		ready, perr := h.kycProbe.CanProviderReceivePayouts(r.Context(), orgID)
		if perr != nil {
			slog.Warn("wallet payout: kyc readiness probe failed, blocking",
				"org_id", orgID, "error", perr)
			respondKYCIncomplete(w)
			return
		}
		if !ready {
			respondKYCIncomplete(w)
			return
		}
	}

	// Phase 6 gate: every payout requires a complete billing profile.
	// If the invoicing module is disabled (svc nil), the gate degrades
	// open — invoicing is a removable feature and must never block the
	// rest of the platform. Errors during the probe are logged and the
	// payout is allowed (fail-open is the safer default for a
	// money-out flow when the gate itself is broken).
	if h.invoicingSvc != nil {
		complete, missing, gerr := h.invoicingSvc.IsBillingProfileComplete(r.Context(), orgID)
		if gerr != nil {
			slog.Warn("wallet payout: billing profile gate probe failed, allowing through",
				"org_id", orgID, "error", gerr)
		} else if !complete {
			respondBillingProfileIncomplete(w, missing, "Complete your billing profile before requesting a payout")
			return
		}
	}

	result, err := h.paymentSvc.RequestPayout(r.Context(), userID, orgID)
	if err != nil {
		if errors.Is(err, paymentdomain.ErrStripeAccountNotFound) {
			res.Error(w, http.StatusForbidden, "stripe_account_missing", "You must complete your payment setup before requesting a payout.")
			return
		}
		res.Error(w, http.StatusInternalServerError, "payout_error", err.Error())
		return
	}

	res.JSON(w, http.StatusOK, result)
}

// RetryFailedTransfer re-issues a Stripe transfer for a single payment
// record stuck in TransferFailed. Bound to
// POST /api/v1/wallet/transfers/{record_id}/retry under the same auth
// + wallet.withdraw permission as /wallet/payout.
//
// Takes the payment record id (NOT the proposal id) because a proposal
// can own multiple records (one per milestone) and only the record id
// is unambiguous.
func (h *WalletHandler) RetryFailedTransfer(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	recordIDRaw := chi.URLParam(r, "record_id")
	recordID, parseErr := uuid.Parse(recordIDRaw)
	if parseErr != nil {
		res.Error(w, http.StatusBadRequest, "invalid_record_id", "record id must be a valid UUID")
		return
	}

	if h.retrier == nil {
		res.Error(w, http.StatusServiceUnavailable, "retry_unavailable", "Retry is not currently available.")
		return
	}
	result, err := h.retrier.RetryFailedTransfer(r.Context(), userID, orgID, recordID)
	if err != nil {
		switch {
		case errors.Is(err, paymentdomain.ErrPaymentRecordNotFound):
			res.Error(w, http.StatusNotFound, "payment_record_not_found", "Ce transfert est introuvable.")
			return
		case errors.Is(err, paymentdomain.ErrTransferNotRetriable):
			res.Error(w, http.StatusConflict, "transfer_not_retriable", "Ce transfert ne peut pas être relancé. La mission doit être terminée et le précédent transfert doit avoir échoué.")
			return
		case errors.Is(err, paymentdomain.ErrStripeAccountNotFound):
			res.Error(w, http.StatusForbidden, "stripe_account_missing", "Tu dois d'abord configurer tes informations de paiement avant de pouvoir relancer ce transfert.")
			return
		case errors.Is(err, paymentdomain.ErrProviderPayoutsDisabled):
			// Account exists but payouts are not yet enabled (KYC pending /
			// capability throttled). Pre-checking here avoids burning the
			// Stripe idempotency key on a doomed CreateTransfer call.
			res.Error(w, http.StatusPreconditionFailed, "provider_kyc_incomplete", "Termine d'abord ton onboarding Stripe pour pouvoir recevoir le virement.")
			return
		}
		slog.Error("wallet retry transfer failed", "record_id", recordID, "user_id", userID, "error", err)
		// Anything else is an upstream Stripe error or a transient infra
		// blip — return 502 so the client can offer "try again later"
		// instead of treating it as a permanent failure.
		res.Error(w, http.StatusBadGateway, "retry_failed", "Le virement a de nouveau échoué côté Stripe. Réessaie dans quelques minutes ou contacte le support.")
		return
	}

	res.JSON(w, http.StatusOK, result)
}

// RetryCommission re-attempts the Stripe transfer for a commission row
// stuck in pending_kyc or failed. Bound to
// POST /api/v1/wallet/commissions/{id}/retry under the same auth +
// wallet.withdraw permission as /wallet/payout (D1+D2).
//
// Status mapping:
//   - 200 OK + paid_at on success
//   - 401 unauthorized when no user/org in context
//   - 403 forbidden when caller is not the apporteur on the parent referral
//   - 404 not_found when no commission row has that id
//   - 409 conflict when the row is already paid or terminal (cancelled/clawed_back)
//   - 422 kyc_required when the Connect gate trips — body carries an
//     onboarding_url so the UI can deep-link the apporteur to KYC
//   - 502 bad_gateway when Stripe returned an error on the transfer
func (h *WalletHandler) RetryCommission(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	// We do not need orgID for the commission retry — ownership is
	// scoped on the apporteur user id (the parent referral row stores
	// referrer_id, not referrer_org_id) — but we still require the
	// org context so the JWT middleware shape is consistent across the
	// wallet routes.
	if _, ok := middleware.GetOrganizationID(r.Context()); !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	commissionIDRaw := chi.URLParam(r, "id")
	commissionID, parseErr := uuid.Parse(commissionIDRaw)
	if parseErr != nil {
		res.Error(w, http.StatusBadRequest, "invalid_commission_id", "commission id must be a valid UUID")
		return
	}

	if h.commissionRetrier == nil {
		res.Error(w, http.StatusServiceUnavailable, "retry_unavailable", "Retry is not currently available.")
		return
	}

	outcome, err := h.commissionRetrier.RetryCommission(r.Context(), userID, commissionID)
	if err != nil {
		h.mapCommissionRetryError(w, err, commissionID, userID)
		return
	}

	switch outcome.Result {
	case portservice.ReferralCommissionRetryPaid:
		res.JSON(w, http.StatusOK, map[string]any{
			"status":         "paid",
			"message":        "Retrait en cours.",
			"stripe_account": outcome.StripeAccount,
		})
	case portservice.ReferralCommissionRetryKYCRequired:
		h.respondCommissionKYCRequired(w, r, userID, outcome.StripeAccount)
	case portservice.ReferralCommissionRetryAlreadyPaid:
		res.Error(w, http.StatusConflict, "already_paid", "Cette commission a déjà été versée.")
	case portservice.ReferralCommissionRetryNotRetriable:
		res.Error(w, http.StatusConflict, "not_retriable", "Cette commission ne peut pas être relancée.")
	case portservice.ReferralCommissionRetryFailed:
		res.JSON(w, http.StatusBadGateway, map[string]any{
			"error": map[string]string{
				"code":    "retry_failed",
				"message": "Le virement a de nouveau échoué côté Stripe.",
			},
			"failure_reason": outcome.FailureReason,
		})
	default:
		// Defensive — every result above is enumerated. An unknown
		// value indicates a port contract drift that should never reach
		// production.
		slog.Error("wallet commission retry: unknown outcome", "outcome", outcome.Result, "commission_id", commissionID)
		res.Error(w, http.StatusInternalServerError, "internal_error", "Une erreur inattendue est survenue.")
	}
}

// mapCommissionRetryError translates a domain error from the commission
// retry orchestrator into the proper HTTP response.
func (h *WalletHandler) mapCommissionRetryError(
	w http.ResponseWriter,
	err error,
	commissionID, userID uuid.UUID,
) {
	switch {
	case errors.Is(err, referraldomain.ErrCommissionNotFound):
		res.Error(w, http.StatusNotFound, "commission_not_found", "Cette commission est introuvable.")
		return
	case errors.Is(err, referralapp.ErrCommissionNotOwned):
		res.Error(w, http.StatusForbidden, "forbidden", "Cette commission ne t'appartient pas.")
		return
	}
	slog.Error("wallet commission retry: orchestrator failed",
		"commission_id", commissionID, "user_id", userID, "error", err)
	res.Error(w, http.StatusBadGateway, "retry_failed", "Le retrait n'a pas pu être lancé. Réessaie dans quelques minutes.")
}

// RetryCommissionDeprecated wraps RetryCommission with the Deprecation
// + Sunset headers per the back-compat brief (Run B WALLET-UNIFY).
// The unified /wallet/withdraw endpoint replaces this — frontend
// (Run C) migrates within the 30-day window.
func (h *WalletHandler) RetryCommissionDeprecated(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Deprecation", "true")
	w.Header().Set("Sunset", deprecationSunset())
	w.Header().Set("Link", `</api/v1/wallet/withdraw>; rel="successor-version"`)
	h.RetryCommission(w, r)
}

// deprecationSunset returns the HTTP-Date sunset value (30 days out)
// for the back-compat headers on the legacy commission retry
// endpoint. Computed at request time so a long-running process never
// serves a stale sunset.
func deprecationSunset() string {
	return time.Now().UTC().Add(30 * 24 * time.Hour).Format(http.TimeFormat)
}

// respondCommissionKYCRequired writes the 422 envelope the UI uses to
// open the KYC modal. The onboarding URL is best-effort — failure to
// resolve it still returns 422 so the UI can fall back to the static
// /payment-info redirect.
func (h *WalletHandler) respondCommissionKYCRequired(
	w http.ResponseWriter,
	r *http.Request,
	userID uuid.UUID,
	stripeAccount string,
) {
	onboardingURL := ""
	if h.kycOnboardingURL != nil {
		url, uerr := h.kycOnboardingURL.GetOnboardingURL(r.Context(), userID)
		if uerr != nil {
			slog.Warn("wallet commission retry: onboarding URL resolution failed",
				"user_id", userID, "error", uerr)
		} else {
			onboardingURL = url
		}
	}
	res.JSON(w, http.StatusUnprocessableEntity, map[string]any{
		"error": map[string]string{
			"code":    "kyc_required",
			"message": "Termine d'abord ton onboarding Stripe pour pouvoir recevoir cette commission.",
		},
		"onboarding_url": onboardingURL,
		"stripe_account": stripeAccount,
		"redirect":       "/payment-info",
	})
}
