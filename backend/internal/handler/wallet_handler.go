package handler

import (
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
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

type WalletHandler struct {
	paymentSvc  *paymentapp.Service
	proposalSvc *proposalapp.Service
	// invoicingSvc is the optional gate the RequestPayout endpoint
	// uses to enforce billing-profile completeness BEFORE handing off
	// to Stripe. Nil = invoicing module disabled, in which case the
	// gate degrades open (the action is allowed) — invoicing is a
	// removable feature and removing it must never block payouts.
	invoicingSvc *invoicingapp.Service
}

func NewWalletHandler(paymentSvc *paymentapp.Service, proposalSvc *proposalapp.Service) *WalletHandler {
	return &WalletHandler{paymentSvc: paymentSvc, proposalSvc: proposalSvc}
}

// WithInvoicing wires the invoicing gate. Builder pattern keeps the
// constructor signature stable so a worktree without invoicing wired in
// still boots — and removing the invoicing feature is a single-line edit
// in main.go.
func (h *WalletHandler) WithInvoicing(svc *invoicingapp.Service) *WalletHandler {
	h.invoicingSvc = svc
	return h
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

	result, err := h.paymentSvc.RetryFailedTransfer(r.Context(), userID, orgID, recordID)
	if err != nil {
		switch {
		case errors.Is(err, paymentdomain.ErrTransferNotRetriable):
			res.Error(w, http.StatusConflict, "transfer_not_retriable", "This transfer cannot be retried. The mission must be completed and the previous transfer must have failed.")
			return
		case errors.Is(err, paymentdomain.ErrStripeAccountNotFound):
			res.Error(w, http.StatusForbidden, "stripe_account_missing", "You must complete your payment setup before retrying a transfer.")
			return
		}
		slog.Error("wallet retry transfer failed", "record_id", recordID, "user_id", userID, "error", err)
		res.Error(w, http.StatusInternalServerError, "retry_error", "Could not retry transfer")
		return
	}

	res.JSON(w, http.StatusOK, result)
}
