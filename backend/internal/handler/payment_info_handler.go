package handler

import (
	"errors"
	"net/http"
	"strings"
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

	info, persons, err := h.paymentService.GetPaymentInfo(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
		return
	}

	if info == nil {
		res.JSON(w, http.StatusOK, nil)
		return
	}

	res.JSON(w, http.StatusOK, response.NewPaymentInfoResponse(info, persons))
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
		Phone:                req.Phone,
		ActivitySector:       req.ActivitySector,
		IsSelfRepresentative: req.IsSelfRepresentative,
		IsSelfDirector:       req.IsSelfDirector,
		NoMajorOwners:        req.NoMajorOwners,
		IsSelfExecutive:      req.IsSelfExecutive,
		BusinessPersons:      mapBusinessPersons(req.BusinessPersons),
		IBAN:                 req.IBAN,
		BIC:                  req.BIC,
		AccountNumber:        req.AccountNumber,
		RoutingNumber:        req.RoutingNumber,
		AccountHolder:        req.AccountHolder,
		BankCountry:          req.BankCountry,
		Country:              req.Country,
		ExtraFields:          req.ExtraFields,
	}

	tosIP := extractIP(r.RemoteAddr)
	email := req.Email
	info, stripeErr, err := h.paymentService.SavePaymentInfo(r.Context(), userID, input, tosIP, email)
	if err != nil {
		handlePaymentInfoError(w, err)
		return
	}

	resp := response.NewPaymentInfoResponse(info, nil)
	resp.StripeError = stripeErr
	res.JSON(w, http.StatusOK, resp)
}

func mapBusinessPersons(reqs []request.BusinessPersonRequest) []paymentapp.BusinessPersonInput {
	if len(reqs) == 0 {
		return nil
	}
	result := make([]paymentapp.BusinessPersonInput, len(reqs))
	for i, r := range reqs {
		var dob time.Time
		if r.DateOfBirth != "" {
			dob, _ = time.Parse("2006-01-02", r.DateOfBirth)
		}
		result[i] = paymentapp.BusinessPersonInput{
			Role:        r.Role,
			FirstName:   r.FirstName,
			LastName:    r.LastName,
			DateOfBirth: dob,
			Email:       r.Email,
			Phone:       r.Phone,
			Address:     r.Address,
			City:        r.City,
			PostalCode:  r.PostalCode,
			Title:       r.Title,
		}
	}
	return result
}

func (h *PaymentInfoHandler) GetCountryFields(w http.ResponseWriter, r *http.Request) {
	country := r.URL.Query().Get("country")
	if country == "" || len(country) != 2 {
		res.Error(w, http.StatusBadRequest, "invalid_country", "country must be a 2-letter ISO code")
		return
	}

	businessType := r.URL.Query().Get("business_type")
	if businessType == "" {
		businessType = "individual"
	}
	if businessType != "individual" && businessType != "company" {
		res.Error(w, http.StatusBadRequest, "invalid_business_type", "business_type must be individual or company")
		return
	}

	fields, err := h.paymentService.GetCountryFields(r.Context(), country, businessType)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get country fields")
		return
	}

	res.JSON(w, http.StatusOK, fields)
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

func (h *PaymentInfoHandler) GetRequirements(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	reqs, err := h.paymentService.GetRequirements(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}

	res.JSON(w, http.StatusOK, reqs)
}

func (h *PaymentInfoHandler) CreateAccountLink(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	url, err := h.paymentService.CreateAccountLink(r.Context(), userID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "account_link_error", err.Error())
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}

// extractIP strips port and brackets from RemoteAddr (e.g. "[::1]:8080" → "127.0.0.1").
func extractIP(addr string) string {
	// Handle IPv6 bracket notation "[::1]:port"
	if len(addr) > 0 && addr[0] == '[' {
		if idx := strings.Index(addr, "]"); idx != -1 {
			ip := addr[1:idx]
			if ip == "::1" {
				return "127.0.0.1"
			}
			return ip
		}
	}
	// Handle IPv4 "host:port"
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		return addr[:idx]
	}
	return addr
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
