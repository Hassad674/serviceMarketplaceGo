package response

import (
	"time"

	"marketplace-backend/internal/domain/user"
)

type UserResponse struct {
	ID              string  `json:"id"`
	Email           string  `json:"email"`
	FirstName       string  `json:"first_name"`
	LastName        string  `json:"last_name"`
	DisplayName     string  `json:"display_name"`
	Role            string  `json:"role"`
	ReferrerEnabled bool    `json:"referrer_enabled"`
	IsAdmin         bool    `json:"is_admin"`
	EmailVerified   bool    `json:"email_verified"`
	KYCStatus       string  `json:"kyc_status"`
	KYCDeadline     *string `json:"kyc_deadline,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

type AuthResponse struct {
	User         UserResponse `json:"user"`
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
}

func NewUserResponse(u *user.User) UserResponse {
	resp := UserResponse{
		ID:              u.ID.String(),
		Email:           u.Email,
		FirstName:       u.FirstName,
		LastName:        u.LastName,
		DisplayName:     u.DisplayName,
		Role:            u.Role.String(),
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
