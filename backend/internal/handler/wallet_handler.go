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
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

type WalletHandler struct {
	paymentSvc  *paymentapp.Service
	proposalSvc *proposalapp.Service
}

func NewWalletHandler(paymentSvc *paymentapp.Service, proposalSvc *proposalapp.Service) *WalletHandler {
	return &WalletHandler{paymentSvc: paymentSvc, proposalSvc: proposalSvc}
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
// POST /api/v1/wallet/transfers/{proposal_id}/retry under the same auth
// + wallet.withdraw permission as /wallet/payout.
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

	proposalIDRaw := chi.URLParam(r, "proposal_id")
	proposalID, parseErr := uuid.Parse(proposalIDRaw)
	if parseErr != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "proposal id must be a valid UUID")
		return
	}

	result, err := h.paymentSvc.RetryFailedTransfer(r.Context(), userID, orgID, proposalID)
	if err != nil {
		switch {
		case errors.Is(err, paymentdomain.ErrTransferNotRetriable):
			res.Error(w, http.StatusConflict, "transfer_not_retriable", "This transfer cannot be retried. The mission must be completed and the previous transfer must have failed.")
			return
		case errors.Is(err, paymentdomain.ErrStripeAccountNotFound):
			res.Error(w, http.StatusForbidden, "stripe_account_missing", "You must complete your payment setup before retrying a transfer.")
			return
		}
		slog.Error("wallet retry transfer failed", "proposal_id", proposalID, "user_id", userID, "error", err)
		res.Error(w, http.StatusInternalServerError, "retry_error", "Could not retry transfer")
		return
	}

	res.JSON(w, http.StatusOK, result)
}
