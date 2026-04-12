package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"marketplace-backend/internal/app/auth"
	orgapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/pkg/validator"
	res "marketplace-backend/pkg/response"
)

// InvitationHandler owns the HTTP transport for team invitations.
// Exposes 6 endpoints: send, list pending, resend, cancel (protected),
// validate by token, and accept (public).
type InvitationHandler struct {
	invitationService *orgapp.InvitationService
	orgService        *orgapp.Service
	tokenSvc          service.TokenService
	sessionSvc        service.SessionService
	cookie            *CookieConfig
}

// InvitationHandlerDeps groups the constructor params to keep it under
// the project's 4-parameter rule.
type InvitationHandlerDeps struct {
	InvitationService *orgapp.InvitationService
	OrgService        *orgapp.Service
	TokenService      service.TokenService
	SessionService    service.SessionService
	Cookie            *CookieConfig
}

func NewInvitationHandler(deps InvitationHandlerDeps) *InvitationHandler {
	return &InvitationHandler{
		invitationService: deps.InvitationService,
		orgService:        deps.OrgService,
		tokenSvc:          deps.TokenService,
		sessionSvc:        deps.SessionService,
		cookie:            deps.Cookie,
	}
}

// ---------------------------------------------------------------------------
// Protected endpoints (Owner/Admin only)
// ---------------------------------------------------------------------------

// Send handles POST /api/v1/organizations/{orgID}/invitations.
func (h *InvitationHandler) Send(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "invalid organization id")
		return
	}

	var req struct {
		Email     string `json:"email"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Title     string `json:"title"`
		Role      string `json:"role"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if errs := validator.ValidateRequired(map[string]string{
		"email":      req.Email,
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"role":       req.Role,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}

	inv, err := h.invitationService.SendInvitation(r.Context(), orgapp.SendInvitationInput{
		InviterUserID:  actorID,
		OrganizationID: orgID,
		Email:          req.Email,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Title:          req.Title,
		Role:           organization.Role(req.Role),
	})
	if err != nil && inv == nil {
		h.handleInvitationError(w, err)
		return
	}
	// Non-nil inv + non-nil err means the invitation was persisted but
	// the email delivery failed. We still return 201 so the UI shows
	// the pending row, and log the error server-side for follow-up.
	if err != nil {
		slog.Warn("team invitation email delivery failed", "invitation_id", inv.ID, "error", err)
	}

	res.JSON(w, http.StatusCreated, response.NewInvitationResponse(inv))
}

// List handles GET /api/v1/organizations/{orgID}/invitations.
func (h *InvitationHandler) List(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "invalid organization id")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	items, next, err := h.invitationService.ListPending(r.Context(), actorID, orgID, cursor, limit)
	if err != nil {
		h.handleInvitationError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewInvitationListResponse(items, next))
}

// Resend handles POST /api/v1/organizations/{orgID}/invitations/{invID}/resend.
func (h *InvitationHandler) Resend(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "invalid organization id")
		return
	}
	invID, err := uuid.Parse(chi.URLParam(r, "invID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_invitation_id", "invalid invitation id")
		return
	}

	inv, err := h.invitationService.ResendInvitation(r.Context(), actorID, orgID, invID)
	if err != nil && inv == nil {
		h.handleInvitationError(w, err)
		return
	}
	if err != nil {
		slog.Warn("team invitation resend email delivery failed", "invitation_id", inv.ID, "error", err)
	}

	res.JSON(w, http.StatusOK, response.NewInvitationResponse(inv))
}

// Cancel handles DELETE /api/v1/organizations/{orgID}/invitations/{invID}.
func (h *InvitationHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	actorID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}

	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "invalid organization id")
		return
	}
	invID, err := uuid.Parse(chi.URLParam(r, "invID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_invitation_id", "invalid invitation id")
		return
	}

	if err := h.invitationService.CancelInvitation(r.Context(), actorID, orgID, invID); err != nil {
		h.handleInvitationError(w, err)
		return
	}

	res.JSON(w, http.StatusNoContent, nil)
}

// ---------------------------------------------------------------------------
// Public endpoints (no auth)
// ---------------------------------------------------------------------------

// Validate handles GET /api/v1/invitations/validate?token=X.
// Returns a lightweight preview of the invitation so the acceptance
// page can pre-fill the user's first name and show what org they're
// joining.
func (h *InvitationHandler) Validate(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		res.Error(w, http.StatusBadRequest, "missing_token", "token query parameter is required")
		return
	}

	result, err := h.invitationService.ValidateToken(r.Context(), token)
	if err != nil {
		h.handleInvitationError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewInvitationPreviewResponse(result.Invitation, result.Organization))
}

// Accept handles POST /api/v1/invitations/accept.
// Creates the operator user, adds them to the org, and returns a full
// AuthResponse (tokens + user + organization) so the frontend can log
// the new operator in immediately without a separate login call.
func (h *InvitationHandler) Accept(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if errs := validator.ValidateRequired(map[string]string{
		"token":    req.Token,
		"password": req.Password,
	}); errs != nil {
		res.ValidationError(w, errs)
		return
	}

	result, err := h.invitationService.AcceptInvitation(r.Context(), orgapp.AcceptInvitationInput{
		Token:    req.Token,
		Password: req.Password,
	})
	if err != nil {
		h.handleInvitationError(w, err)
		return
	}

	// Issue tokens for the brand new operator and return the full
	// auth envelope — matches the register/login response shape so
	// mobile and web clients can reuse their existing auth flow.
	accessInput := service.AccessTokenInput{
		UserID:  result.User.ID,
		Role:    result.User.Role.String(),
		IsAdmin: result.User.IsAdmin,
	}
	if result.Organization != nil && result.Member != nil {
		orgID := result.Organization.ID
		accessInput.OrganizationID = &orgID
		accessInput.OrgRole = result.Member.Role.String()
	}

	accessToken, err := h.tokenSvc.GenerateAccessToken(accessInput)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to generate access token")
		return
	}
	refreshToken, err := h.tokenSvc.GenerateRefreshToken(result.User.ID)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to generate refresh token")
		return
	}

	output := &auth.AuthOutput{
		User:         result.User,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	if result.Organization != nil && result.Member != nil {
		orgID := result.Organization.ID
		output.OrganizationID = &orgID
		output.OrgRole = result.Member.Role.String()
	}

	// Mobile mode: return tokens in body
	if r.Header.Get("X-Auth-Mode") == "token" {
		res.JSON(w, http.StatusCreated, response.NewAuthResponseWithOrg(output.User, result.OrgContext, output.AccessToken, output.RefreshToken))
		return
	}

	// Web mode: create session, set cookie, return user + org
	session, err := h.sessionSvc.Create(r.Context(), service.CreateSessionInput{
		UserID:         output.User.ID,
		Role:           output.User.Role.String(),
		IsAdmin:        output.User.IsAdmin,
		OrganizationID: output.OrganizationID,
		OrgRole:        output.OrgRole,
		Permissions:    permissionKeysFromOrgContext(result.OrgContext),
		SessionVersion: output.User.SessionVersion,
	})
	if err != nil {
		slog.Error("failed to create session after invitation accept", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to create session")
		return
	}
	h.cookie.SetSession(w, session.ID, output.User.Role.String())
	res.JSON(w, http.StatusCreated, response.NewMeResponse(output.User, result.OrgContext))
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

func (h *InvitationHandler) handleInvitationError(w http.ResponseWriter, err error) {
	switch {
	// Authorization
	case errors.Is(err, organization.ErrNotAMember):
		res.Error(w, http.StatusForbidden, "not_a_member", "you are not a member of this organization")
	case errors.Is(err, organization.ErrPermissionDenied):
		res.Error(w, http.StatusForbidden, "permission_denied", "you do not have permission to perform this action")
	case errors.Is(err, organization.ErrForbidden):
		res.Error(w, http.StatusForbidden, "forbidden", err.Error())

	// Lookup
	case errors.Is(err, organization.ErrOrgNotFound):
		res.Error(w, http.StatusNotFound, "organization_not_found", "organization not found")
	case errors.Is(err, organization.ErrInvitationNotFound):
		res.Error(w, http.StatusNotFound, "invitation_not_found", "invitation not found")
	case errors.Is(err, organization.ErrMemberNotFound):
		res.Error(w, http.StatusNotFound, "member_not_found", "member not found")

	// Invitation lifecycle
	case errors.Is(err, organization.ErrInvitationExpired):
		res.Error(w, http.StatusGone, "invitation_expired", "invitation expired")
	case errors.Is(err, organization.ErrInvitationAlreadyUsed):
		res.Error(w, http.StatusConflict, "invitation_already_used", "invitation already accepted")
	case errors.Is(err, organization.ErrInvitationCancelled):
		res.Error(w, http.StatusConflict, "invitation_cancelled", "invitation was cancelled")
	case errors.Is(err, organization.ErrInvalidInvitationStatus):
		res.Error(w, http.StatusConflict, "invalid_invitation_status", "invitation is no longer in a pending state")
	case errors.Is(err, organization.ErrCannotInviteAsOwner):
		res.Error(w, http.StatusBadRequest, "cannot_invite_as_owner", "cannot invite a user as Owner — use transfer ownership instead")
	case errors.Is(err, organization.ErrAlreadyMember):
		res.Error(w, http.StatusConflict, "already_member", "this email is already registered on the platform")
	case errors.Is(err, organization.ErrAlreadyInvited):
		res.Error(w, http.StatusConflict, "already_invited", "an invitation is already pending for this email")

	// Validation
	case errors.Is(err, organization.ErrInvalidEmail):
		res.Error(w, http.StatusBadRequest, "invalid_email", "invalid email format")
	case errors.Is(err, organization.ErrInvalidRole):
		res.Error(w, http.StatusBadRequest, "invalid_role", "invalid role")
	case errors.Is(err, organization.ErrNameRequired):
		res.Error(w, http.StatusBadRequest, "name_required", "first name and last name are required")
	case errors.Is(err, organization.ErrNameTooLong):
		res.Error(w, http.StatusBadRequest, "name_too_long", "name exceeds the maximum length")
	case errors.Is(err, organization.ErrTitleTooLong):
		res.Error(w, http.StatusBadRequest, "title_too_long", "title exceeds the maximum length")

	// Password (bubbled up from domain/user)
	case errors.Is(err, user.ErrWeakPassword):
		res.Error(w, http.StatusBadRequest, "weak_password", err.Error())
	case errors.Is(err, user.ErrEmailAlreadyExists):
		res.Error(w, http.StatusConflict, "email_exists", "email already exists")

	// Rate limit
	case errors.Is(err, orgapp.ErrInvitationRateLimited):
		w.Header().Set("Retry-After", "3600")
		res.Error(w, http.StatusTooManyRequests, "rate_limit_exceeded", "too many invitations sent recently, try again later")

	default:
		slog.Error("unhandled invitation error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}

