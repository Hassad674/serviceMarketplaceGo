package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	paymentapp "marketplace-backend/internal/app/payment"
	proposalapp "marketplace-backend/internal/app/proposal"
	paymentdomain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	portservice "marketplace-backend/internal/port/service"

	res "marketplace-backend/pkg/response"
)

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
type ProposalPaymentHandler struct {
	proposalSvc *proposalapp.Service
	paymentSvc  *paymentapp.Service // nil if Stripe not configured
}

// NewProposalPaymentHandler wires the funding handler.
func NewProposalPaymentHandler(svc *proposalapp.Service, paymentSvc *paymentapp.Service) *ProposalPaymentHandler {
	return &ProposalPaymentHandler{proposalSvc: svc, paymentSvc: paymentSvc}
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
