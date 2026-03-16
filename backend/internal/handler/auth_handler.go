package handler

import (
	"errors"
	"net/http"

	"marketplace-backend/internal/app/auth"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"
	res "marketplace-backend/pkg/response"
)

type AuthHandler struct {
	authService *auth.Service
}

func NewAuthHandler(authService *auth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		DisplayName string `json:"display_name"`
		Role        string `json:"role"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if errs := validator.ValidateRequired(map[string]string{
		"email":        req.Email,
		"password":     req.Password,
		"first_name":   req.FirstName,
		"last_name":    req.LastName,
		"display_name": req.DisplayName,
		"role":         req.Role,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}

	if err := validator.ValidateRole(req.Role); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_role", err.Error())
		return
	}

	output, err := h.authService.Register(r.Context(), auth.RegisterInput{
		Email:       req.Email,
		Password:    req.Password,
		FirstName:   req.FirstName,
		LastName:    req.LastName,
		DisplayName: req.DisplayName,
		Role:        user.Role(req.Role),
	})
	if err != nil {
		handleAuthError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, response.NewAuthResponse(output.User, output.AccessToken, output.RefreshToken))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if errs := validator.ValidateRequired(map[string]string{
		"email":    req.Email,
		"password": req.Password,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}

	output, err := h.authService.Login(r.Context(), auth.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		handleAuthError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewAuthResponse(output.User, output.AccessToken, output.RefreshToken))
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if req.RefreshToken == "" {
		res.Error(w, http.StatusBadRequest, "invalid_request", "refresh_token is required")
		return
	}

	output, err := h.authService.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewAuthResponse(output.User, output.AccessToken, output.RefreshToken))
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	u, err := h.authService.GetMe(r.Context(), userID)
	if err != nil {
		handleAuthError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewUserResponse(u))
}

func handleAuthError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, user.ErrInvalidEmail):
		res.Error(w, http.StatusBadRequest, "invalid_email", err.Error())
	case errors.Is(err, user.ErrWeakPassword):
		res.Error(w, http.StatusBadRequest, "weak_password", err.Error())
	case errors.Is(err, user.ErrEmailAlreadyExists):
		res.Error(w, http.StatusConflict, "email_exists", err.Error())
	case errors.Is(err, user.ErrInvalidCredentials):
		res.Error(w, http.StatusUnauthorized, "invalid_credentials", err.Error())
	case errors.Is(err, user.ErrUserNotFound):
		res.Error(w, http.StatusNotFound, "user_not_found", err.Error())
	case errors.Is(err, user.ErrUnauthorized):
		res.Error(w, http.StatusUnauthorized, "unauthorized", err.Error())
	case errors.Is(err, user.ErrInvalidRole):
		res.Error(w, http.StatusBadRequest, "invalid_role", err.Error())
	default:
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
