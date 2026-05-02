package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	orgapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/domain/organization"
	domainuser "marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
	"marketplace-backend/pkg/validator"
	res "marketplace-backend/pkg/response"
)

// TeamHandler owns the HTTP surface for team management: listing
// members, updating roles/titles, removing members, self-leave, and
// the 4-step transfer ownership flow.
//
// userBatch is an additive, read-only batch reader injected so the
// member list endpoint can hydrate user identity fields (display name,
// first/last, email) without an N+1 loop. The dependency is optional —
// if it is nil the handler degrades to "no identity fields" rather
// than failing the request, so older wirings keep working.
//
// sessionSvc, cookie, and users are required for the AcceptTransfer
// endpoint, which refreshes the accepter's session inline so the user
// stays logged in after the ownership transfer bumps session_version.
type TeamHandler struct {
	membership *orgapp.MembershipService
	orgService *orgapp.Service
	userBatch  repository.UserBatchReader
	sessionSvc service.SessionService
	cookie     *CookieConfig
	users      repository.UserReader
}

// TeamHandlerDeps groups constructor params for TeamHandler. The
// handler needs 6 dependencies (membership, org service, user batch
// reader, session service, cookie config, user repo) which exceeds
// the project's 4-parameter limit for plain function signatures.
type TeamHandlerDeps struct {
	Membership     *orgapp.MembershipService
	OrgService     *orgapp.Service
	UserBatch      repository.UserBatchReader
	SessionService service.SessionService
	Cookie         *CookieConfig
	Users          repository.UserReader
}

func NewTeamHandler(deps TeamHandlerDeps) *TeamHandler {
	return &TeamHandler{
		membership: deps.Membership,
		orgService: deps.OrgService,
		userBatch:  deps.UserBatch,
		sessionSvc: deps.SessionService,
		cookie:     deps.Cookie,
		users:      deps.Users,
	}
}

// ---------------------------------------------------------------------------
// Role definitions — read-only catalogue
// ---------------------------------------------------------------------------

// RoleDefinitions handles GET /api/v1/organizations/role-definitions.
//
// Returns the static catalogue of roles and permissions used by the
// team page's "About roles" panel and the Edit Member modal's inline
// permissions preview. The response is read directly from the domain
// rolePermissions map (the single source of truth) — no duplication.
//
// English labels and descriptions are returned inline as fallbacks
// the frontend can use until its own i18n catalogue is updated. The
// frontend translates by key (`team.roles.admin.label`,
// `team.permissions.team_invite.label`) and falls back to the inline
// English string when no translation exists.
//
// Authentication required: this is called from authenticated team
// pages, so the same auth middleware as the rest of the team routes
// applies. No special role check — every authenticated user may
// read the catalogue.
func (h *TeamHandler) RoleDefinitions(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.GetUserID(r.Context()); !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	payload := response.NewRoleDefinitionsPayload(
		organization.AllRoles(),
		organization.AllPermissionMetadata(),
	)
	res.JSON(w, http.StatusOK, payload)
}

// ---------------------------------------------------------------------------
// Members — list, update, remove
// ---------------------------------------------------------------------------

// ListMembers handles GET /api/v1/organizations/{orgID}/members.
//
// Each member row is hydrated with the matching user identity block
// (display name, email, first/last) via a single batch query against
// the users table — no N+1 loop. If the userBatch dependency is
// missing or the batch call fails, the handler still returns the
// member rows but without the identity block (the frontend already
// handles the missing-user case gracefully by falling back to a
// generic label).
func (h *TeamHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	actorID, orgID, ok := h.authContext(w, r)
	if !ok {
		return
	}
	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	items, next, err := h.membership.ListMembers(r.Context(), actorID, orgID, cursor, limit)
	if err != nil {
		h.handleTeamError(w, err)
		return
	}
	usersByID := h.batchUsersForMembers(r, items)
	res.JSON(w, http.StatusOK, response.NewMemberListResponseWithUsers(items, usersByID, next))
}

// batchUsersForMembers fetches the joined user records for a slice of
// members in a single round-trip and returns them keyed by user id
// string. Logs (but does not propagate) errors so the list endpoint
// degrades gracefully when the user lookup fails.
func (h *TeamHandler) batchUsersForMembers(
	r *http.Request,
	members []*organization.Member,
) map[string]*domainuser.User {
	out := make(map[string]*domainuser.User, len(members))
	if h.userBatch == nil || len(members) == 0 {
		return out
	}
	ids := make([]uuid.UUID, 0, len(members))
	for _, m := range members {
		ids = append(ids, m.UserID)
	}
	users, err := h.userBatch.GetByIDs(r.Context(), ids)
	if err != nil {
		slog.Warn("team list: batch user fetch failed",
			"error", err, "member_count", len(members))
		return out
	}
	for _, u := range users {
		if u == nil {
			continue
		}
		out[u.ID.String()] = u
	}
	return out
}

// UpdateMember handles PATCH /api/v1/organizations/{orgID}/members/{userID}.
// Body accepts optional `role` and/or `title` — the handler applies
// whichever is present, each in a dedicated service call.
func (h *TeamHandler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	actorID, orgID, ok := h.authContext(w, r)
	if !ok {
		return
	}
	targetUserID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return
	}

	var req struct {
		Role  *string `json:"role,omitempty"`
		Title *string `json:"title,omitempty"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	if req.Role == nil && req.Title == nil {
		res.Error(w, http.StatusBadRequest, "no_changes", "provide at least one of role or title")
		return
	}

	var updated *organization.Member
	if req.Role != nil {
		updated, err = h.membership.UpdateMemberRole(r.Context(), actorID, orgID, targetUserID, organization.Role(*req.Role))
		if err != nil {
			h.handleTeamError(w, err)
			return
		}
	}
	if req.Title != nil {
		updated, err = h.membership.UpdateMemberTitle(r.Context(), actorID, orgID, targetUserID, *req.Title)
		if err != nil {
			h.handleTeamError(w, err)
			return
		}
	}
	res.JSON(w, http.StatusOK, response.NewMemberResponse(updated))
}

// RemoveMember handles DELETE /api/v1/organizations/{orgID}/members/{userID}.
func (h *TeamHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	actorID, orgID, ok := h.authContext(w, r)
	if !ok {
		return
	}
	targetUserID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_user_id", "invalid user id")
		return
	}
	if err := h.membership.RemoveMember(r.Context(), actorID, orgID, targetUserID); err != nil {
		h.handleTeamError(w, err)
		return
	}
	res.JSON(w, http.StatusNoContent, nil)
}

// Leave handles POST /api/v1/organizations/{orgID}/leave.
// The caller removes themselves from the org.
func (h *TeamHandler) Leave(w http.ResponseWriter, r *http.Request) {
	actorID, orgID, ok := h.authContext(w, r)
	if !ok {
		return
	}
	if err := h.membership.LeaveOrganization(r.Context(), actorID, orgID); err != nil {
		h.handleTeamError(w, err)
		return
	}
	res.JSON(w, http.StatusNoContent, nil)
}

// ---------------------------------------------------------------------------
// Transfer ownership
// ---------------------------------------------------------------------------

// InitiateTransfer handles POST /api/v1/organizations/{orgID}/transfer.
// Body: {"target_user_id": "uuid"}.
func (h *TeamHandler) InitiateTransfer(w http.ResponseWriter, r *http.Request) {
	actorID, orgID, ok := h.authContext(w, r)
	if !ok {
		return
	}
	var req struct {
		TargetUserID string `json:"target_user_id"`
	}
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	targetUserID, err := uuid.Parse(req.TargetUserID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_target_user_id", "invalid target user id")
		return
	}
	org, err := h.membership.InitiateTransferOwnership(r.Context(), actorID, orgID, targetUserID)
	if err != nil {
		h.handleTeamError(w, err)
		return
	}
	res.JSON(w, http.StatusAccepted, response.NewTransferResponse(org))
}

// CancelTransfer handles DELETE /api/v1/organizations/{orgID}/transfer.
// The current Owner cancels a pending transfer.
func (h *TeamHandler) CancelTransfer(w http.ResponseWriter, r *http.Request) {
	actorID, orgID, ok := h.authContext(w, r)
	if !ok {
		return
	}
	if err := h.membership.CancelTransferOwnership(r.Context(), actorID, orgID); err != nil {
		h.handleTeamError(w, err)
		return
	}
	res.JSON(w, http.StatusNoContent, nil)
}

// AcceptTransfer handles POST /api/v1/organizations/{orgID}/transfer/accept.
// The proposed new owner confirms.
//
// After the service call succeeds, the handler refreshes the accepter's
// session inline in the HTTP response so the user stays logged in
// seamlessly. AcceptTransferOwnership bumps session_version for both
// the old and new owner; without this refresh, the accepter would get a
// 401 on the very next request (their old session's version is stale).
//
// Mobile clients (X-Auth-Mode: token) handle token refresh through
// their own refresh-token flow, so they still receive the plain
// transfer response.
func (h *TeamHandler) AcceptTransfer(w http.ResponseWriter, r *http.Request) {
	actorID, orgID, ok := h.authContext(w, r)
	if !ok {
		return
	}
	if _, err := h.membership.AcceptTransferOwnership(r.Context(), actorID, orgID); err != nil {
		h.handleTeamError(w, err)
		return
	}

	// Mobile path: return the transfer response and let the client
	// handle token refresh via the standard refresh-token flow.
	if r.Header.Get("X-Auth-Mode") == "token" {
		h.sendTransferResponseFallback(w, r, actorID, orgID)
		return
	}

	// Web path: refresh the session inline so the user stays logged in.
	h.refreshSessionAfterTransfer(w, r, actorID)
}

// refreshSessionAfterTransfer deletes the accepter's stale session,
// creates a fresh one with the updated SessionVersion and org context,
// sets the new cookie, and returns a /me-style response so the
// frontend cache stays in sync.
func (h *TeamHandler) refreshSessionAfterTransfer(
	w http.ResponseWriter,
	r *http.Request,
	actorID uuid.UUID,
) {
	// 1. Delete the old (now-stale) session.
	if cookie, err := r.Cookie("session_id"); err == nil {
		_ = h.sessionSvc.Delete(r.Context(), cookie.Value)
	}

	// 2. Fetch the fresh user row (carries updated SessionVersion).
	freshUser, err := h.users.GetByID(r.Context(), actorID)
	if err != nil {
		slog.Error("accept transfer: failed to fetch fresh user", "user_id", actorID, "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to refresh session")
		return
	}

	// 3. Resolve the fresh org context (role is now Owner).
	var orgCtx *orgapp.Context
	if h.orgService != nil {
		resolved, resolveErr := h.orgService.ResolveContext(r.Context(), actorID)
		if resolveErr != nil {
			slog.Warn("accept transfer: failed to resolve org context", "user_id", actorID, "error", resolveErr)
		} else {
			orgCtx = resolved
		}
	}

	// 4. Build session input — include the resolved effective
	// permission set so the new session honors any per-org role
	// overrides the accepter inherits as the new Owner.
	input := service.CreateSessionInput{
		UserID:         freshUser.ID,
		Role:           freshUser.Role.String(),
		IsAdmin:        freshUser.IsAdmin,
		Permissions:    permissionKeysFromOrgContext(orgCtx),
		SessionVersion: freshUser.SessionVersion,
	}
	if orgCtx != nil && orgCtx.Organization != nil && orgCtx.Member != nil {
		orgID := orgCtx.Organization.ID
		input.OrganizationID = &orgID
		input.OrgRole = orgCtx.Member.Role.String()
	}

	// 5. Create the new session.
	session, err := h.sessionSvc.Create(r.Context(), input)
	if err != nil {
		slog.Error("accept transfer: failed to create session", "user_id", actorID, "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to refresh session")
		return
	}

	// 6. Set the new cookie and return /me-style response.
	h.cookie.SetSession(w, session.ID, freshUser.Role.String())
	res.JSON(w, http.StatusOK, response.NewMeResponse(freshUser, orgCtx))
}

// sendTransferResponseFallback resolves the org and returns the plain
// transfer response for mobile clients that handle token refresh
// independently.
func (h *TeamHandler) sendTransferResponseFallback(
	w http.ResponseWriter,
	r *http.Request,
	actorID uuid.UUID,
	orgID uuid.UUID,
) {
	var orgCtx *orgapp.Context
	if h.orgService != nil {
		resolved, _ := h.orgService.ResolveContext(r.Context(), actorID)
		orgCtx = resolved
	}
	var org *organization.Organization
	if orgCtx != nil {
		org = orgCtx.Organization
	}
	if org == nil {
		org = &organization.Organization{ID: orgID, OwnerUserID: actorID}
	}
	res.JSON(w, http.StatusOK, response.NewTransferResponse(org))
}

// DeclineTransfer handles POST /api/v1/organizations/{orgID}/transfer/decline.
// The proposed new owner refuses — the org reverts to its previous state.
func (h *TeamHandler) DeclineTransfer(w http.ResponseWriter, r *http.Request) {
	actorID, orgID, ok := h.authContext(w, r)
	if !ok {
		return
	}
	if err := h.membership.DeclineTransferOwnership(r.Context(), actorID, orgID); err != nil {
		h.handleTeamError(w, err)
		return
	}
	res.JSON(w, http.StatusNoContent, nil)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// authContext pulls the authenticated user id and the URL's orgID
// parameter out of the request. Returns (uuid.Nil, uuid.Nil, false)
// and writes the appropriate error response when either is missing.
func (h *TeamHandler) authContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	actorID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "authentication required")
		return uuid.Nil, uuid.Nil, false
	}
	orgID, err := uuid.Parse(chi.URLParam(r, "orgID"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_org_id", "invalid organization id")
		return uuid.Nil, uuid.Nil, false
	}
	return actorID, orgID, true
}

// handleTeamError maps domain sentinels to HTTP codes for team routes.
func (h *TeamHandler) handleTeamError(w http.ResponseWriter, err error) {
	switch {
	// Authorization
	case errors.Is(err, organization.ErrNotAMember):
		res.Error(w, http.StatusForbidden, "not_a_member", "you are not a member of this organization")
	case errors.Is(err, organization.ErrPermissionDenied):
		res.Error(w, http.StatusForbidden, "permission_denied", "you do not have permission to perform this action")
	case errors.Is(err, organization.ErrForbidden):
		res.Error(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, organization.ErrCannotChangeOwnRole):
		res.Error(w, http.StatusForbidden, "cannot_change_own_role", "cannot change your own role — use leave or transfer ownership")
	case errors.Is(err, organization.ErrCannotRemoveSelf):
		res.Error(w, http.StatusForbidden, "cannot_remove_self", "cannot remove yourself — use leave organization")

	// Lookup
	case errors.Is(err, organization.ErrOrgNotFound):
		res.Error(w, http.StatusNotFound, "organization_not_found", "organization not found")
	case errors.Is(err, organization.ErrMemberNotFound):
		res.Error(w, http.StatusNotFound, "member_not_found", "member not found")

	// Membership invariants
	case errors.Is(err, organization.ErrOwnerCannotBeRemoved):
		res.Error(w, http.StatusConflict, "owner_cannot_be_removed", "the organization owner cannot be removed — transfer ownership first")
	case errors.Is(err, organization.ErrOwnerCannotBeDemoted):
		res.Error(w, http.StatusConflict, "owner_cannot_be_demoted", "the organization owner cannot be demoted — transfer ownership first")
	case errors.Is(err, organization.ErrLastOwnerCannotLeave):
		res.Error(w, http.StatusConflict, "last_owner_cannot_leave", "the owner cannot leave — transfer ownership first")
	case errors.Is(err, organization.ErrLastOwnerCannotDemote):
		res.Error(w, http.StatusConflict, "last_owner_cannot_demote", "the owner cannot self-demote — transfer ownership first")
	case errors.Is(err, organization.ErrCannotInviteAsOwner):
		res.Error(w, http.StatusBadRequest, "cannot_promote_to_owner", "cannot promote to Owner — use transfer ownership instead")

	// Transfer
	case errors.Is(err, organization.ErrTransferAlreadyPending):
		res.Error(w, http.StatusConflict, "transfer_already_pending", "a transfer is already pending")
	case errors.Is(err, organization.ErrNoPendingTransfer):
		res.Error(w, http.StatusNotFound, "no_pending_transfer", "no transfer pending")
	case errors.Is(err, organization.ErrTransferExpired):
		res.Error(w, http.StatusGone, "transfer_expired", "transfer expired")
	case errors.Is(err, organization.ErrTransferTargetInvalid):
		res.Error(w, http.StatusBadRequest, "transfer_target_invalid", "transfer target must be an existing Admin of the organization")
	case errors.Is(err, organization.ErrCannotTransferToSelf):
		res.Error(w, http.StatusBadRequest, "cannot_transfer_to_self", "cannot transfer ownership to yourself")

	// Validation
	case errors.Is(err, organization.ErrInvalidRole):
		res.Error(w, http.StatusBadRequest, "invalid_role", "invalid role")
	case errors.Is(err, organization.ErrTitleTooLong):
		res.Error(w, http.StatusBadRequest, "title_too_long", "title exceeds the maximum length")

	default:
		slog.Error("unhandled team error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
