package handler

import (
	"net/http"

	paymentapp "marketplace-backend/internal/app/payment"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

type WalletHandler struct {
	paymentSvc *paymentapp.Service
}

func NewWalletHandler(paymentSvc *paymentapp.Service) *WalletHandler {
	return &WalletHandler{paymentSvc: paymentSvc}
}

// GetWallet returns wallet overview: escrow balance, available, transfers history.
func (h *WalletHandler) GetWallet(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	wallet, err := h.paymentSvc.GetWalletOverview(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	res.JSON(w, http.StatusOK, wallet)
}

// RequestPayout triggers a payout from the connected account to the provider's bank.
func (h *WalletHandler) RequestPayout(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	result, err := h.paymentSvc.RequestPayout(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "payout_error", err.Error())
		return
	}

	res.JSON(w, http.StatusOK, result)
}
