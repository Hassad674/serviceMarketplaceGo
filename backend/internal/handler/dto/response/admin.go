package response

import (
	"time"

	"marketplace-backend/internal/domain/user"
)

type AdminUserResponse struct {
	ID                  string  `json:"id"`
	Email               string  `json:"email"`
	FirstName           string  `json:"first_name"`
	LastName            string  `json:"last_name"`
	DisplayName         string  `json:"display_name"`
	Role                string  `json:"role"`
	ReferrerEnabled     bool    `json:"referrer_enabled"`
	IsAdmin             bool    `json:"is_admin"`
	Status              string  `json:"status"`
	SuspendedAt         *string `json:"suspended_at,omitempty"`
	SuspensionReason    string  `json:"suspension_reason,omitempty"`
	SuspensionExpiresAt *string `json:"suspension_expires_at,omitempty"`
	BannedAt            *string `json:"banned_at,omitempty"`
	BanReason           string  `json:"ban_reason,omitempty"`
	EmailVerified       bool    `json:"email_verified"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

func NewAdminUserResponse(u *user.User) AdminUserResponse {
	r := AdminUserResponse{
		ID:               u.ID.String(),
		Email:            u.Email,
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		DisplayName:      u.DisplayName,
		Role:             string(u.Role),
		ReferrerEnabled:  u.ReferrerEnabled,
		IsAdmin:          u.IsAdmin,
		Status:           string(u.Status),
		SuspensionReason: u.SuspensionReason,
		BanReason:        u.BanReason,
		EmailVerified:    u.EmailVerified,
		CreatedAt:        u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        u.UpdatedAt.Format(time.RFC3339),
	}
	if u.SuspendedAt != nil {
		s := u.SuspendedAt.Format(time.RFC3339)
		r.SuspendedAt = &s
	}
	if u.SuspensionExpiresAt != nil {
		s := u.SuspensionExpiresAt.Format(time.RFC3339)
		r.SuspensionExpiresAt = &s
	}
	if u.BannedAt != nil {
		s := u.BannedAt.Format(time.RFC3339)
		r.BannedAt = &s
	}
	return r
}
