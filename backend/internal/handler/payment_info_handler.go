package handler

import (
	"net/http"

	paymentapp "marketplace-backend/internal/app/payment"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

type PaymentInfoHandler struct {
	paymentService *paymentapp.Service
}

func NewPaymentInfoHandler(paymentService *paymentapp.Service) *PaymentInfoHandler {
	return &PaymentInfoHandler{paymentService: paymentService}
}

func (h *PaymentInfoHandler) GetPaymentInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	info, err := h.paymentService.GetPaymentInfo(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}

	if info == nil {
		res.JSON(w, http.StatusOK, nil)
		return
	}

	res.JSON(w, http.StatusOK, response.NewPaymentInfoResponse(info))
}

func (h *PaymentInfoHandler) GetPaymentInfoStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	complete, err := h.paymentService.IsComplete(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}

	res.JSON(w, http.StatusOK, response.PaymentInfoStatusResponse{Complete: complete})
}

func (h *PaymentInfoHandler) CreateAccountSession(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var body struct {
		Email string `json:"email"`
	}
	if err := validator.DecodeJSON(r, &body); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	result, err := h.paymentService.CreateAccountSession(r.Context(), userID, body.Email)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "account_session_error", err.Error())
		return
	}

	res.JSON(w, http.StatusOK, result)
}
