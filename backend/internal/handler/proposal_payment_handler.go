package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	invoicingapp "marketplace-backend/internal/app/invoicing"
	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	domaininv "marketplace-backend/internal/domain/invoicing"
	paymentdomain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	portservice "marketplace-backend/internal/port/service"

	res "marketplace-backend/pkg/response"
)

// billingProfileGate is the narrow read-only contract the proposal
// payment handler uses to prove the client organization has filled in
// its billing profile BEFORE we trigger a Stripe PaymentIntent. Kept as
// a small interface (Interface Segregation) so handler tests can drive
// every branch with a 5-line fake without standing up the entire
// invoicing service.
//
// The real *invoicingapp.Service satisfies it natively.
type billingProfileGate interface {
	IsBillingProfileComplete(ctx context.Context, organizationID uuid.UUID) (bool, []domaininv.MissingField, error)
}

// ProposalPaymentHandler owns the funding endpoints — the surface that
// transitions a proposal from accepted to active by initiating a Stripe
// PaymentIntent and confirming its settlement.
//
// SRP rationale: every method here writes to the payment side of a
// proposal. Lifecycle (create/accept/cancel) lives on
// ProposalLifecycleHandler; completion (request/approve/reject) lives
// on ProposalCompletionHandler.
//
// Dependencies:
//   - proposalSvc: drives the InitiatePayment / ConfirmPaymentAndActivate /
//     AuthorizeClientOrg use cases.
//   - paymentSvc:  needed for MarkPaymentSucceeded (SEC-02 Stripe
//     verification). Nil when Stripe is not configured — the
//     ConfirmPayment handler degrades to "trust the local record" with
//     a logged warning, mirroring the pre-Phase-3 behaviour.
//   - invoicingSvc: optional gate that blocks PayProposal / FundMilestone
//     when the client organization has an incomplete billing profile.
//     Nil = invoicing module disabled and the gate degrades open
//     (invoicing is a removable feature and must never block the rest of
//     the platform). Wired post-construction via WithInvoicing inside
//     wireInvoicing — same builder pattern as WalletHandler.
type ProposalPaymentHandler struct {
	proposalSvc  *proposalapp.Service
	paymentSvc   *paymentapp.Service // nil if Stripe not configured
	invoicingSvc billingProfileGate  // nil if invoicing not configured
}

// NewProposalPaymentHandler wires the funding handler.
func NewProposalPaymentHandler(svc *proposalapp.Service, paymentSvc *paymentapp.Service) *ProposalPaymentHandler {
	return &ProposalPaymentHandler{proposalSvc: svc, paymentSvc: paymentSvc}
}

// WithInvoicing wires the billing-profile gate. Builder pattern keeps
// the constructor signature stable so a worktree without invoicing wired
// in still boots — and removing the invoicing feature is a single-line
// edit in main.go. Same shape as WalletHandler.WithInvoicing.
func (h *ProposalPaymentHandler) WithInvoicing(svc *invoicingapp.Service) *ProposalPaymentHandler {
	if svc != nil {
		h.invoicingSvc = svc
	}
	return h
}

// withBillingGate is the test seam that lets us inject a fake
// billingProfileGate without standing up the real invoicing service.
// Production code goes through WithInvoicing; tests use this directly.
func (h *ProposalPaymentHandler) withBillingGate(g billingProfileGate) *ProposalPaymentHandler {
	h.invoicingSvc = g
	return h
}

// requireClientBillingComplete enforces the client organization has a
// complete billing profile before initiating any payment. Returns false
// (and writes the response) when the gate blocks the call. Returns true
// to indicate the caller may proceed.
//
// Fail-open posture matches the wallet handler: when the invoicing
// module is disabled (gate nil) or the probe itself errors, the request
// is allowed through. The probe error is logged via slog.Warn so it
// surfaces in observability dashboards. We never want a transient gate
// failure to block real money flows on a near-final production app.
//
// Audit trail: every block writes a structured slog.Info line with
// action `payment.blocked_billing_incomplete` so the security audit
// pipeline can correlate (org_id, proposal_id, missing_fields). Slog is
// the canonical audit channel at the handler layer in this codebase —
// the audit_logs table is append-only via app services.
func (h *ProposalPaymentHandler) requireClientBillingComplete(
	w http.ResponseWriter,
	r *http.Request,
	orgID, proposalID uuid.UUID,
) bool {
	if h.invoicingSvc == nil {
		return true
	}
	complete, missing, err := h.invoicingSvc.IsBillingProfileComplete(r.Context(), orgID)
	if err != nil {
		slog.Warn("proposal payment: billing profile gate probe failed, allowing through",
			"org_id", orgID, "proposal_id", proposalID, "error", err)
		return true
	}
	if complete {
		return true
	}
	slog.Info("payment.blocked_billing_incomplete",
		"org_id", orgID,
		"proposal_id", proposalID,
		"missing_fields_count", len(missing),
	)
	respondClientBillingProfileIncomplete(w, missing)
	return false
}

// respondClientBillingProfileIncomplete writes the canonical 412
// Precondition Required envelope when the client organization has not
// yet filled in its billing profile. The shape mirrors the wallet
// handler's billing_profile_incomplete response so the frontend can
// reuse a single completion modal for both flows. The discriminator
// `code` lets the modal differentiate the two surfaces.
//
// 412 (RFC 7232 §4.2 + RFC 6585 §3) is the right status because the
// resource state is fine — the precondition that the caller must
// supply complete billing info before triggering a charge is what
// fails. The wallet handler historically uses 403 here; the user-
// facing brief explicitly asked for 412 on this surface to better
// match the semantics, and the discriminator code stays identical so
// the existing UX stays consistent.
func respondClientBillingProfileIncomplete(w http.ResponseWriter, missing []domaininv.MissingField) {
	if missing == nil {
		missing = []domaininv.MissingField{}
	}
	res.JSON(w, http.StatusPreconditionRequired, map[string]any{
		"error": map[string]string{
			"code":    "billing_profile_incomplete",
			"message": "Complète tes informations de facturation avant de payer cette proposition.",
		},
		"missing_fields": missing,
	})
}

// PayProposal handles POST /api/v1/proposals/{id}/pay. Legacy one-time
// mode endpoint — delegates to InitiatePayment which routes to the
// Stripe charge flow or simulation depending on configuration.
func (h *ProposalPaymentHandler) PayProposal(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	// Billing profile gate: a client paying a proposal must have a
	// complete billing profile so the resulting receipt snapshot
	// (PR #165) carries the legally required recipient identity. The
	// gate runs BEFORE InitiatePayment so we never charge a card and
	// then fail to issue a usable receipt. Authorization (caller's
	// org owns the client side) is enforced inside InitiatePayment;
	// since the gate uses the caller's org id from the JWT context,
	// a non-client caller that would fail authorization downstream
	// would only see a benign "your billing profile is incomplete"
	// 412, not a confidential leak — the message is identical for
	// every authenticated org.
	if !h.requireClientBillingComplete(w, r, orgID, proposalID) {
		return
	}

	result, err := h.proposalSvc.InitiatePayment(r.Context(), proposalapp.PayProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	// Simulation mode: payment completed immediately
	if result == nil {
		res.JSON(w, http.StatusOK, map[string]string{"status": "paid"})
		return
	}

	// Stripe mode: return client_secret for Elements
	res.JSON(w, http.StatusOK, paymentIntentResponse(result))
}

// ConfirmPayment handles POST /api/v1/proposals/{id}/confirm-payment.
// Called by the frontend after stripe.confirmPayment() succeeds. Acts
// as a fallback to the webhook — verifies the PaymentIntent has
// actually settled with Stripe before flipping the local record.
//
// SEC-02 / BUG-01: a client with DevTools could otherwise bypass the
// real Stripe charge.
func (h *ProposalPaymentHandler) ConfirmPayment(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	// Verify the caller's org is on the client side — any operator of
	// the client org can confirm payment on behalf of their team. Only
	// the client side is authorized because they are the party who
	// released the funds. The provider-side org must NOT be able to
	// bounce the payment record to "succeeded" on its own.
	if err := h.proposalSvc.AuthorizeClientOrg(r.Context(), proposalID, orgID); err != nil {
		handleProposalError(w, err)
		return
	}

	// Mark the payment record as succeeded — but ONLY if Stripe
	// confirms the PaymentIntent has actually settled. SEC-02 / BUG-01:
	// a client could otherwise call this endpoint with no real charge
	// and have the proposal flipped to `active`, draining escrow on
	// completion.
	if h.paymentSvc != nil {
		if err := h.paymentSvc.MarkPaymentSucceeded(r.Context(), proposalID); err != nil {
			if errors.Is(err, paymentdomain.ErrPaymentNotConfirmed) {
				slog.Warn("payment confirm denied: stripe verification failed",
					"proposal_id", proposalID, "org_id", orgID)
				res.Error(w, http.StatusPaymentRequired, "payment_not_confirmed",
					"payment intent has not settled with stripe")
				return
			}
			slog.Error("mark payment succeeded", "proposal_id", proposalID, "error", err)
			res.Error(w, http.StatusInternalServerError, "payment_verification_failed",
				"could not verify payment intent")
			return
		}
	}

	// Confirm payment and activate the proposal
	if err := h.proposalSvc.ConfirmPaymentAndActivate(r.Context(), proposalID); err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "active"})
}

// FundMilestone handles POST /proposals/{id}/milestones/{mid}/fund.
// Validates that {mid} matches the current active milestone, then
// calls InitiatePayment which routes through the milestone state
// machine to create a Stripe PaymentIntent or simulate the payment.
func (h *ProposalPaymentHandler) FundMilestone(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := requireAuthContext(w, r)
	if !ok {
		return
	}
	proposalID, milestoneID, ok := parseProposalAndMilestoneID(w, r)
	if !ok {
		return
	}
	// Same billing-profile gate as PayProposal — milestone-mode
	// payments still require the client org's billing identity so
	// the receipt snapshot is regeneratable. Run BEFORE the
	// milestone-matches-current check so a user with an incomplete
	// profile sees the "fix your billing first" 412 instead of an
	// unrelated milestone-state 409.
	if !h.requireClientBillingComplete(w, r, orgID, proposalID) {
		return
	}
	if !validateMilestoneMatchesCurrent(w, r, h.proposalSvc, proposalID, milestoneID) {
		return
	}
	result, err := h.proposalSvc.InitiatePayment(r.Context(), proposalapp.PayProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
		OrgID:      orgID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}
	if result == nil {
		res.JSON(w, http.StatusOK, map[string]string{"status": "funded"})
		return
	}
	res.JSON(w, http.StatusOK, paymentIntentResponse(result))
}

// paymentIntentResponse builds the Stripe-mode response payload from a
// proposal-service InitiatePayment result. Centralised so both
// PayProposal and FundMilestone produce the same shape.
func paymentIntentResponse(result *portservice.PaymentIntentOutput) response.PaymentIntentResponse {
	return response.PaymentIntentResponse{
		ClientSecret:    result.ClientSecret,
		PaymentRecordID: result.PaymentRecordID.String(),
		Amounts: response.PaymentAmounts{
			ProposalAmount: result.ProposalAmount,
			StripeFee:      result.StripeFee,
			PlatformFee:    result.PlatformFee,
			ClientTotal:    result.ClientTotal,
			ProviderPayout: result.ProviderPayout,
		},
	}
}
