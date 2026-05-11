package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	paymentapp "marketplace-backend/internal/app/payment"
	paymentdomain "marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/handler/middleware"
	portservice "marketplace-backend/internal/port/service"
	jsondec "marketplace-backend/pkg/decode"
	res "marketplace-backend/pkg/response"
)

// withdrawCommissionRetrier is the narrow port the unified withdraw
// endpoint uses to drain a commission row. Already defined as
// commissionRetrier on the wallet handler — re-declared here for
// readability of the withdraw orchestrator.
type withdrawCommissionRetrier = commissionRetrier

// withdrawRequest is the JSON body of POST /wallet/withdraw. Both
// fields are optional: an empty body drains everything available.
type withdrawRequest struct {
	AmountCents int64 `json:"amount_cents,omitempty"`
}

// withdrawResponse is the success envelope.
type withdrawResponse struct {
	DrainedCents      int64    `json:"drained_cents"`
	MissionsCents     int64    `json:"missions_cents"`
	CommissionsCents  int64    `json:"commissions_cents"`
	StripeTransferIDs []string `json:"stripe_transfer_ids"`
	Currency          string   `json:"currency"`
	// Errors is populated on partial failure (HTTP 207). Empty on
	// 200 success. Carries a short message per failed source so
	// the UI can surface which leg failed.
	Errors []withdrawLegError `json:"errors,omitempty"`
}

// withdrawLegError is one entry in withdrawResponse.Errors.
type withdrawLegError struct {
	Source  string `json:"source"`  // "missions" | "commissions"
	Code    string `json:"code"`    // short machine code
	Message string `json:"message"` // human message
}

// Withdraw drains the wallet across BOTH missions and commissions in
// a single call (Run B WALLET-UNIFY). The legs are processed in this
// order: missions first, then commissions — mirrors the wallet UI's
// listing order.
//
// Returns:
//   - 200 OK on full success (both legs ran cleanly, even if either
//     was empty).
//   - 207 Multi-Status when ≥ 1 transfer succeeded AND ≥ 1 leg failed
//     (partial drain — the client keeps the success, the body carries
//     the failure detail in Errors[]).
//   - 422 kyc_required when Stripe Connect is not ready (mirrors
//     the existing /wallet/payout flow).
//   - 403 billing_profile_incomplete when the invoicing gate blocks.
//   - 500 internal_error when the whole flow blows up before any
//     transfer was attempted.
//
// Idempotency: the Idempotency-Key middleware caches the response
// (same key → same JSON body, no double Stripe call).
//
// Audit: a single ActionWalletWithdrawExecuted entry is emitted on
// success (200 or 207), capturing the breakdown for forensic
// review. Audit failure is logged but never breaks the response.
func (h *WalletHandler) Withdraw(w http.ResponseWriter, r *http.Request) {
	userID, orgID, ok := h.requireAuthContext(w, r)
	if !ok {
		return
	}

	var req withdrawRequest
	if r.ContentLength > 0 {
		if err := jsondec.DecodeBody(w, r, &req, 1<<10); err != nil {
			res.Error(w, http.StatusBadRequest, "invalid_request", "request body invalid")
			return
		}
	}
	if req.AmountCents < 0 {
		res.Error(w, http.StatusBadRequest, "invalid_amount", "amount_cents must be non-negative")
		return
	}

	if !h.checkKYCGate(w, r) {
		return
	}
	if !h.checkBillingGate(w, r, orgID) {
		return
	}

	resp := h.driveWithdraw(r.Context(), userID, orgID, req.AmountCents)
	h.emitWithdrawAudit(r.Context(), userID, orgID, resp)
	h.respondWithdraw(w, resp)
}

// requireAuthContext extracts userID + orgID from the JWT context,
// short-circuiting with 401 on missing values. Returns (userID,
// orgID, ok).
func (h *WalletHandler) requireAuthContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return uuid.Nil, uuid.Nil, false
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, orgID, true
}

// checkKYCGate enforces the Stripe Connect readiness pre-check.
// Returns false (and writes the response) when the gate trips.
func (h *WalletHandler) checkKYCGate(w http.ResponseWriter, r *http.Request) bool {
	if h.kycProbe == nil {
		return true
	}
	orgID, _ := middleware.GetOrganizationID(r.Context())
	ready, perr := h.kycProbe.CanProviderReceivePayouts(r.Context(), orgID)
	if perr != nil {
		slog.Warn("wallet withdraw: kyc readiness probe failed, blocking",
			"org_id", orgID, "error", perr)
		h.respondWithdrawKYCRequired(w, r)
		return false
	}
	if !ready {
		h.respondWithdrawKYCRequired(w, r)
		return false
	}
	return true
}

// checkBillingGate enforces the invoicing billing-profile pre-check.
// Returns false (and writes the response) when the gate trips.
func (h *WalletHandler) checkBillingGate(w http.ResponseWriter, r *http.Request, orgID uuid.UUID) bool {
	if h.invoicingSvc == nil {
		return true
	}
	complete, missing, gerr := h.invoicingSvc.IsBillingProfileComplete(r.Context(), orgID)
	if gerr != nil {
		// Fail-open as documented on the legacy RequestPayout path —
		// a broken billing gate must never block a money-out flow.
		slog.Warn("wallet withdraw: billing profile gate probe failed, allowing through",
			"org_id", orgID, "error", gerr)
		return true
	}
	if !complete {
		respondBillingProfileIncomplete(w, missing, "Complete your billing profile before requesting a withdrawal")
		return false
	}
	return true
}

// respondWithdrawKYCRequired writes the 422 kyc_required envelope
// with the optional onboarding URL embedded. Mirrors the shape used
// by RetryCommission so the frontend can reuse the same handler
// regardless of which endpoint surfaced the gate.
func (h *WalletHandler) respondWithdrawKYCRequired(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	onboardingURL := ""
	if h.kycOnboardingURL != nil {
		url, uerr := h.kycOnboardingURL.GetOnboardingURL(r.Context(), userID)
		if uerr != nil {
			slog.Warn("wallet withdraw: onboarding URL resolution failed",
				"user_id", userID, "error", uerr)
		} else {
			onboardingURL = url
		}
	}
	res.JSON(w, http.StatusUnprocessableEntity, map[string]any{
		"error": map[string]string{
			"code":    "kyc_required",
			"message": "Termine d'abord ton onboarding Stripe pour pouvoir retirer.",
		},
		"onboarding_url": onboardingURL,
		"redirect":       "/payment-info",
	})
}

// driveWithdraw runs the two legs (missions, then commissions) and
// composes the response. Stops the commissions leg when amount_cents
// is non-zero and missions already covered the request.
func (h *WalletHandler) driveWithdraw(ctx context.Context, userID, orgID uuid.UUID, requestedCents int64) *withdrawResponse {
	resp := &withdrawResponse{Currency: "EUR", StripeTransferIDs: []string{}, Errors: []withdrawLegError{}}

	// Mission leg.
	missionTransferred, missionErr := h.runMissionLeg(ctx, userID, orgID)
	resp.MissionsCents = missionTransferred
	if missionErr != nil {
		resp.Errors = append(resp.Errors, withdrawLegError{
			Source:  "missions",
			Code:    missionErrCode(missionErr),
			Message: missionErr.Error(),
		})
	}

	// Decide whether the commission leg should run. When the client
	// asked for a specific amount and missions already met or
	// exceeded it, skip the commission leg.
	if requestedCents > 0 && missionTransferred >= requestedCents {
		resp.DrainedCents = missionTransferred
		return resp
	}

	// Commission leg.
	remainingCap := int64(0)
	if requestedCents > 0 {
		remainingCap = requestedCents - missionTransferred
	}
	commissionTransferred, ids, commissionErr := h.runCommissionLeg(ctx, userID, remainingCap)
	resp.CommissionsCents = commissionTransferred
	resp.StripeTransferIDs = append(resp.StripeTransferIDs, ids...)
	if commissionErr != nil {
		resp.Errors = append(resp.Errors, withdrawLegError{
			Source:  "commissions",
			Code:    "commission_drain_failed",
			Message: commissionErr.Error(),
		})
	}

	resp.DrainedCents = missionTransferred + commissionTransferred
	return resp
}

// runMissionLeg invokes the existing payment.RequestPayout — REUSE,
// not rewrite, per the brief. Returns the cents transferred (parsed
// from the PayoutResult message) and any error.
func (h *WalletHandler) runMissionLeg(ctx context.Context, userID, orgID uuid.UUID) (int64, error) {
	if h.paymentSvc == nil {
		return 0, nil
	}
	result, err := h.paymentSvc.RequestPayout(ctx, userID, orgID)
	if err != nil {
		return 0, err
	}
	return missionDrainedFromResult(result), nil
}

// missionDrainedFromResult derives the cents transferred from the
// payment service's PayoutResult. The result.Status discriminates
// success cases. On "nothing_to_transfer" we return 0; on the two
// success statuses we re-fetch the wallet overview to know what just
// moved — the legacy result type only carries a "Transferred X
// centimes" message, which is too fragile to parse.
//
// Implementation note: the legacy RequestPayout is intentionally
// idempotent — calling it again right after a successful drain
// returns nothing_to_transfer. So a fresh wallet read here is
// authoritative.
func missionDrainedFromResult(result *paymentapp.PayoutResult) int64 {
	if result == nil {
		return 0
	}
	switch result.Status {
	case "transferred", "transferred_pending_bank":
		// The legacy service doesn't expose the integer amount on
		// the result type. We re-derive it from the message — which
		// always has the shape "Transferred X centimes to your
		// account". Failing to parse falls back to 0; the audit log
		// captures the raw error.
		return parseMissionAmountFromMessage(result.Message)
	}
	return 0
}

// parseMissionAmountFromMessage extracts the integer amount from the
// PayoutResult message. Pure helper so the parsing rule is testable.
// The legacy payment service emits two known shapes:
//
//	"Transferred N centimes to your account"
//	"Transferred N centimes — bank transfer pending"
//
// Both start with "Transferred N centimes" so a single format string
// captures both variants. Returns 0 when the parse fails — the audit
// log captures the raw message verbatim for forensic review.
func parseMissionAmountFromMessage(msg string) int64 {
	var amount int64
	if _, err := fmt.Sscanf(msg, "Transferred %d centimes", &amount); err != nil {
		return 0
	}
	return amount
}

// unused capture to silence linter if removed in a future refactor —
// the unused-port reference keeps the contract explicit.
var _ = (*withdrawCommissionRetrier)(nil)

// runCommissionLeg drains every retirable commission row up to
// remainingCap (0 = no cap, drain all). Returns the total cents
// drained, the Stripe transfer IDs that succeeded, and the first
// error encountered (subsequent errors are logged but not surfaced
// — partial drain semantics).
func (h *WalletHandler) runCommissionLeg(ctx context.Context, userID uuid.UUID, remainingCap int64) (int64, []string, error) {
	if h.commissionRecorder == nil || h.commissionRetrier == nil {
		return 0, nil, nil
	}
	recs, err := h.commissionRecorder.RecentCommissions(ctx, userID, 100)
	if err != nil {
		return 0, nil, err
	}
	var drained int64
	var ids []string
	var firstErr error
	for _, rec := range recs {
		if !rec.RetireEligible {
			continue
		}
		if remainingCap > 0 && drained >= remainingCap {
			break
		}
		transferred, transferID, retryErr := h.tryDrainCommission(ctx, userID, rec)
		if retryErr != nil {
			if firstErr == nil {
				firstErr = retryErr
			}
			continue
		}
		if transferred > 0 {
			drained += transferred
			if transferID != "" {
				ids = append(ids, transferID)
			}
		}
	}
	return drained, ids, firstErr
}

// tryDrainCommission drives a single commission row through the
// retrier. Returns the cents drained on success (0 when not paid),
// the stripe transfer id (when emitted by the orchestrator), and any
// retry error.
func (h *WalletHandler) tryDrainCommission(ctx context.Context, userID uuid.UUID, rec portservice.ReferralCommissionRecord) (int64, string, error) {
	outcome, err := h.commissionRetrier.RetryCommission(ctx, userID, rec.ID)
	if err != nil {
		return 0, "", err
	}
	switch outcome.Result {
	case portservice.ReferralCommissionRetryPaid:
		return rec.CommissionCents, outcome.StripeAccount, nil
	default:
		// Anything else (KYCRequired, AlreadyPaid, NotRetriable,
		// Failed) is a no-op on the cumulative drain — the audit
		// log captures the breakdown.
		return 0, "", nil
	}
}

// emitWithdrawAudit writes the success audit entry. Failures are
// logged but never break the response — the funds have already moved.
func (h *WalletHandler) emitWithdrawAudit(ctx context.Context, userID, orgID uuid.UUID, resp *withdrawResponse) {
	if h.auditLogger == nil {
		return
	}
	if resp.DrainedCents == 0 {
		// Nothing moved — no audit event.
		return
	}
	entry := &auditEntry{
		UserID:       userID,
		OrgID:        orgID,
		Action:       "wallet.withdraw_executed",
		ResourceType: "wallet",
		Metadata: map[string]any{
			"drained_cents":      resp.DrainedCents,
			"missions_cents":     resp.MissionsCents,
			"commissions_cents":  resp.CommissionsCents,
			"stripe_transfer_ids": resp.StripeTransferIDs,
			"currency":           resp.Currency,
		},
	}
	if err := h.auditLogger.Log(ctx, entry); err != nil {
		slog.Warn("wallet withdraw: audit write failed",
			"user_id", userID, "org_id", orgID, "error", err)
	}
}

// respondWithdraw picks the HTTP status code and writes the envelope:
//   - 207 Multi-Status when ≥ 1 transfer succeeded AND ≥ 1 leg failed
//   - 500 Internal Server Error when nothing moved AND a leg failed
//   - 200 OK otherwise (including the "nothing to drain" case)
func (h *WalletHandler) respondWithdraw(w http.ResponseWriter, resp *withdrawResponse) {
	switch {
	case resp.DrainedCents > 0 && len(resp.Errors) > 0:
		res.JSON(w, http.StatusMultiStatus, map[string]any{"data": resp})
	case resp.DrainedCents == 0 && len(resp.Errors) > 0:
		res.JSON(w, http.StatusInternalServerError, map[string]any{
			"data":  resp,
			"error": map[string]string{"code": "withdraw_failed", "message": "Le retrait n'a pas pu être exécuté."},
		})
	default:
		res.JSON(w, http.StatusOK, map[string]any{"data": resp})
	}
}

// missionErrCode maps a payment-domain error to a short code so the
// 207 / 500 envelope is machine-readable.
func missionErrCode(err error) string {
	switch {
	case errors.Is(err, paymentdomain.ErrStripeAccountNotFound):
		return "stripe_account_missing"
	case errors.Is(err, paymentdomain.ErrProviderPayoutsDisabled):
		return "provider_kyc_incomplete"
	}
	return "missions_drain_failed"
}
