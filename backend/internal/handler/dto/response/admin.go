package response

import (
	"time"

	"marketplace-backend/internal/domain/user"
)

// DashboardStatsResponse is the JSON response for GET /api/v1/admin/dashboard/stats.
type DashboardStatsResponse struct {
	TotalUsers      int                    `json:"total_users"`
	UsersByRole     map[string]int         `json:"users_by_role"`
	ActiveUsers     int                    `json:"active_users"`
	SuspendedUsers  int                    `json:"suspended_users"`
	BannedUsers     int                    `json:"banned_users"`
	TotalProposals  int                    `json:"total_proposals"`
	ActiveProposals int                    `json:"active_proposals"`
	TotalJobs       int                    `json:"total_jobs"`
	OpenJobs        int                    `json:"open_jobs"`
	RecentSignups   []RecentSignupResponse `json:"recent_signups"`
}

// RecentSignupResponse is a lightweight user representation for recent signups.
type RecentSignupResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
	CreatedAt   string `json:"created_at"`
}

func NewRecentSignupResponse(u *user.User) RecentSignupResponse {
	displayName := u.DisplayName
	if displayName == "" {
		displayName = u.FirstName + " " + u.LastName
	}
	return RecentSignupResponse{
		ID:          u.ID.String(),
		DisplayName: displayName,
		Email:       u.Email,
		Role:        string(u.Role),
		CreatedAt:   u.CreatedAt.Format(time.RFC3339),
	}
}

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
