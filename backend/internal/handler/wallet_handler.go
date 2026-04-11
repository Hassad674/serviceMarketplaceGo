package handler

import (
	"errors"
	"net/http"

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
