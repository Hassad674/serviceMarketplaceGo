package response

import (
	"time"

	orgapp "marketplace-backend/internal/app/organization"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
)

type UserResponse struct {
	ID              string  `json:"id"`
	Email           string  `json:"email"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	DisplayName     string  `json:"display_name"`
	Role            string  `json:"role"`
	AccountType     string  `json:"account_type"`
	ReferrerEnabled bool    `json:"referrer_enabled"`
	IsAdmin         bool    `json:"is_admin"`
	EmailVerified   bool    `json:"email_verified"`
	KYCStatus       string  `json:"kyc_status"`
	KYCDeadline     *string `json:"kyc_deadline,omitempty"`
	// DeletedAt is set when the user requested deletion via the
	// GDPR right-to-erasure flow (P5). Until the cron purges them
	// at T+30 they can still log in as a normal user (the auth
	// guard refuses login at the handler layer when this field is
	// set, BUT /auth/me lets the frontend learn the schedule so it
	// can show the "your account will be deleted" banner).
	DeletedAt    *string `json:"deleted_at,omitempty"`
	HardDeleteAt *string `json:"hard_delete_at,omitempty"`
	CreatedAt    string  `json:"created_at"`
}

// OrganizationResponse carries the user's organization context in /me and
// /auth responses. It is only populated when the user belongs to an org
// (Agency/Enterprise owner or invited operator). Providers receive nil.
//
// The pending_transfer_* fields are exposed so the web team page can
// render the "transfer in progress" banner without a second round-trip.
// They are nil whenever no transfer is in flight.
type OrganizationResponse struct {
	ID                         string   `json:"id"`
	Type                       string   `json:"type"`
	OwnerUserID                string   `json:"owner_user_id"`
	MemberRole                 string   `json:"member_role"`
	MemberTitle                string   `json:"member_title"`
	Permissions                []string `json:"permissions"`
	PendingTransferToUserID    *string  `json:"pending_transfer_to_user_id,omitempty"`
	PendingTransferInitiatedAt *string  `json:"pending_transfer_initiated_at,omitempty"`
	PendingTransferExpiresAt   *string  `json:"pending_transfer_expires_at,omitempty"`
}

type AuthResponse struct {
	User         UserResponse          `json:"user"`
	Organization *OrganizationResponse `json:"organization,omitempty"`
	AccessToken  string                `json:"access_token"`
	RefreshToken string                `json:"refresh_token"`
}

// MeResponse is what GET /api/v1/auth/me returns. Same user + org shape
// as AuthResponse minus the tokens (cookies carry them on web, the
// original login response carried them on mobile).
type MeResponse struct {
	User         UserResponse          `json:"user"`
	Organization *OrganizationResponse `json:"organization,omitempty"`
}

func NewUserResponse(u *user.User) UserResponse {
	accountType := u.AccountType.String()
	if accountType == "" {
		accountType = string(user.AccountTypeMarketplaceOwner)
	}
	resp := UserResponse{
		ID:              u.ID.String(),
		Email:           u.Email,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		DisplayName:     u.DisplayName,
		Role:            u.Role.String(),
		AccountType:     accountType,
		ReferrerEnabled: u.ReferrerEnabled,
		IsAdmin:         u.IsAdmin,
		EmailVerified:   u.EmailVerified,
		CreatedAt:       u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if u.DeletedAt != nil {
		// Render BOTH the soft-delete timestamp AND the scheduled
		// hard-delete time so the frontend banner can compute the
		// 30-day countdown without an extra fetch.
		s := u.DeletedAt.Format(time.RFC3339)
		hard := u.DeletedAt.Add(30 * 24 * time.Hour).Format(time.RFC3339)
		resp.DeletedAt = &s
		resp.HardDeleteAt = &hard
	}
	return resp
}

// applyOrgKYCToUser injects the org's KYC state onto the user response.
// Called when the /me handler has loaded the caller's org — the
// frontend still reads kyc_status/kyc_deadline off the user object,
// but since phase R5 these values come from the merchant org.
func applyOrgKYCToUser(resp *UserResponse, org *organization.Organization) {
	if org == nil {
		resp.KYCStatus = "none"
		return
	}
	resp.KYCStatus = orgKYCStatus(org)
	if org.KYCFirstEarningAt != nil && !org.HasKYCCompleted() {
		deadline := org.KYCFirstEarningAt.Add(14 * 24 * time.Hour).Format(time.RFC3339)
		resp.KYCDeadline = &deadline
	}
}

// NewOrganizationResponse converts an app-layer org context into the HTTP
// response shape. Returns nil when the context is nil or incomplete.
func NewOrganizationResponse(ctx *orgapp.Context) *OrganizationResponse {
	if ctx == nil || ctx.Organization == nil || ctx.Member == nil {
		return nil
	}
	perms := make([]string, 0, len(ctx.Permissions))
	for _, p := range ctx.Permissions {
		perms = append(perms, string(p))
	}
	resp := &OrganizationResponse{
		ID:          ctx.Organization.ID.String(),
		Type:        ctx.Organization.Type.String(),
		OwnerUserID: ctx.Organization.OwnerUserID.String(),
		MemberRole:  ctx.Member.Role.String(),
		MemberTitle: ctx.Member.Title,
		Permissions: perms,
	}
	if ctx.Organization.PendingTransferToUserID != nil {
		s := ctx.Organization.PendingTransferToUserID.String()
		resp.PendingTransferToUserID = &s
	}
	if ctx.Organization.PendingTransferInitiatedAt != nil {
		s := ctx.Organization.PendingTransferInitiatedAt.Format(time.RFC3339)
		resp.PendingTransferInitiatedAt = &s
	}
	if ctx.Organization.PendingTransferExpiresAt != nil {
		s := ctx.Organization.PendingTransferExpiresAt.Format(time.RFC3339)
		resp.PendingTransferExpiresAt = &s
	}
	return resp
}

// NewMeResponse assembles the /me payload from a user and an optional
// org context. The org's KYC state is projected onto the user response
// so existing frontends keep reading kyc_status / kyc_deadline without
// changes.
func NewMeResponse(u *user.User, orgCtx *orgapp.Context) MeResponse {
	userResp := NewUserResponse(u)
	if orgCtx != nil && orgCtx.Organization != nil {
		applyOrgKYCToUser(&userResp, orgCtx.Organization)
	} else {
		userResp.KYCStatus = "none"
	}
	return MeResponse{
		User:         userResp,
		Organization: NewOrganizationResponse(orgCtx),
	}
}

// orgKYCStatus computes the KYC status string for an organization.
//   - "completed" — Stripe account exists
//   - "restricted" — 14 days elapsed, no KYC
//   - "pending" — first earning recorded, KYC deadline running
//   - "none" — no earnings yet (no KYC required)
func orgKYCStatus(o *organization.Organization) string {
	if o.HasKYCCompleted() {
		return "completed"
	}
	if o.IsKYCBlocked() {
		return "restricted"
	}
	if o.KYCFirstEarningAt != nil {
		return "pending"
	}
	return "none"
}

func NewAuthResponse(u *user.User, accessToken, refreshToken string) AuthResponse {
	return AuthResponse{
		User:         NewUserResponse(u),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}

// NewAuthResponseWithOrg is like NewAuthResponse but also includes the
// user's organization context. Used by Register/Login responses on mobile
// (X-Auth-Mode: token) so the mobile client knows the org on first contact.
func NewAuthResponseWithOrg(u *user.User, orgCtx *orgapp.Context, accessToken, refreshToken string) AuthResponse {
	return AuthResponse{
		User:         NewUserResponse(u),
		Organization: NewOrganizationResponse(orgCtx),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
}
