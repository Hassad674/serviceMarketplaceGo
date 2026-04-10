package response

import (
	"time"

	orgapp "marketplace-backend/internal/app/organization"
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
	CreatedAt       string  `json:"created_at"`
}

// OrganizationResponse carries the user's organization context in /me and
// /auth responses. It is only populated when the user belongs to an org
// (Agency/Enterprise owner or invited operator). Providers receive nil.
type OrganizationResponse struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	OwnerUserID string   `json:"owner_user_id"`
	MemberRole  string   `json:"member_role"`
	MemberTitle string   `json:"member_title"`
	Permissions []string `json:"permissions"`
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
		KYCStatus:       kycStatus(u),
		CreatedAt:       u.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if u.KYCFirstEarningAt != nil && !u.HasKYCCompleted() {
		deadline := u.KYCFirstEarningAt.Add(14 * 24 * time.Hour).Format(time.RFC3339)
		resp.KYCDeadline = &deadline
	}
	return resp
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
	return &OrganizationResponse{
		ID:          ctx.Organization.ID.String(),
		Type:        ctx.Organization.Type.String(),
		OwnerUserID: ctx.Organization.OwnerUserID.String(),
		MemberRole:  ctx.Member.Role.String(),
		MemberTitle: ctx.Member.Title,
		Permissions: perms,
	}
}

// NewMeResponse assembles the /me payload from a user and an optional
// org context.
func NewMeResponse(u *user.User, orgCtx *orgapp.Context) MeResponse {
	return MeResponse{
		User:         NewUserResponse(u),
		Organization: NewOrganizationResponse(orgCtx),
	}
}

// kycStatus computes the KYC status string for the auth response.
//   - "completed" — Stripe account exists
//   - "restricted" — 14 days elapsed, no KYC
//   - "pending" — first earning recorded, KYC deadline running
//   - "none" — no earnings yet (no KYC required)
func kycStatus(u *user.User) string {
	if u.HasKYCCompleted() {
		return "completed"
	}
	if u.IsKYCBlocked() {
		return "restricted"
	}
	if u.KYCFirstEarningAt != nil {
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
