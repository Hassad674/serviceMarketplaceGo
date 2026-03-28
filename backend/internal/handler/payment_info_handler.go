package handler

import (
	"errors"
	"net/http"
	"time"

	paymentapp "marketplace-backend/internal/app/payment"
	"marketplace-backend/internal/domain/payment"
	"marketplace-backend/internal/handler/dto/request"
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

func (h *PaymentInfoHandler) SavePaymentInfo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.SavePaymentInfoRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	dob, err := time.Parse("2006-01-02", req.DateOfBirth)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_date", "date_of_birth must be in YYYY-MM-DD format")
		return
	}

	input := paymentapp.SavePaymentInfoInput{
		FirstName:          req.FirstName,
		LastName:           req.LastName,
		DateOfBirth:        dob,
		Nationality:        req.Nationality,
		Address:            req.Address,
		City:               req.City,
		PostalCode:         req.PostalCode,
		IsBusiness:         req.IsBusiness,
		BusinessName:       req.BusinessName,
		BusinessAddress:    req.BusinessAddress,
		BusinessCity:       req.BusinessCity,
		BusinessPostalCode: req.BusinessPostalCode,
		BusinessCountry:    req.BusinessCountry,
		TaxID:              req.TaxID,
		VATNumber:          req.VATNumber,
		RoleInCompany:      req.RoleInCompany,
		IBAN:               req.IBAN,
		BIC:                req.BIC,
		AccountNumber:      req.AccountNumber,
		RoutingNumber:      req.RoutingNumber,
		AccountHolder:      req.AccountHolder,
		BankCountry:        req.BankCountry,
	}

	info, err := h.paymentService.SavePaymentInfo(r.Context(), userID, input)
	if err != nil {
		handlePaymentInfoError(w, err)
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

func handlePaymentInfoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, payment.ErrFirstNameRequired),
		errors.Is(err, payment.ErrLastNameRequired),
		errors.Is(err, payment.ErrDateOfBirthRequired),
		errors.Is(err, payment.ErrNationalityRequired),
		errors.Is(err, payment.ErrAddressRequired),
		errors.Is(err, payment.ErrCityRequired),
		errors.Is(err, payment.ErrPostalCodeRequired),
		errors.Is(err, payment.ErrAccountHolderRequired),
		errors.Is(err, payment.ErrBankDetailsRequired),
		errors.Is(err, payment.ErrBusinessNameRequired),
		errors.Is(err, payment.ErrTaxIDRequired):
		res.Error(w, http.StatusUnprocessableEntity, "validation_error", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
