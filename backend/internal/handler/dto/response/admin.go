package response

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/report"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// DashboardStatsResponse is the JSON response for GET /api/v1/admin/dashboard/stats.
type DashboardStatsResponse struct {
	TotalUsers         int                    `json:"total_users"`
	UsersByRole        map[string]int         `json:"users_by_role"`
	ActiveUsers        int                    `json:"active_users"`
	SuspendedUsers     int                    `json:"suspended_users"`
	BannedUsers        int                    `json:"banned_users"`
	TotalProposals     int                    `json:"total_proposals"`
	ActiveProposals    int                    `json:"active_proposals"`
	TotalJobs          int                    `json:"total_jobs"`
	OpenJobs           int                    `json:"open_jobs"`
	TotalOrganizations int                    `json:"total_organizations"`
	PendingInvitations int                    `json:"pending_invitations"`
	RecentSignups      []RecentSignupResponse `json:"recent_signups"`
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
	// AccountType tells the admin UI whether this user is a marketplace
	// owner (self-registered agency/enterprise/provider) or an operator
	// invited into an organization. The team management UI is only
	// rendered when the user has the right account_type + org_id.
	AccountType string `json:"account_type"`
	// OrganizationID is the denormalized pointer maintained by the
	// organization app service. Null for solo providers and for
	// marketplace owners who have not yet created an org.
	OrganizationID      *string `json:"organization_id,omitempty"`
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
	accountType := string(u.AccountType)
	if accountType == "" {
		// Legacy rows that predate migration 055 have an empty string;
		// treat them as marketplace owners (the historical default).
		accountType = string(user.AccountTypeMarketplaceOwner)
	}
	r := AdminUserResponse{
		ID:               u.ID.String(),
		Email:            u.Email,
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		DisplayName:      u.DisplayName,
		Role:             string(u.Role),
		AccountType:      accountType,
		ReferrerEnabled:  u.ReferrerEnabled,
		IsAdmin:          u.IsAdmin,
		Status:           string(u.Status),
		SuspensionReason: u.SuspensionReason,
		BanReason:        u.BanReason,
		EmailVerified:    u.EmailVerified,
		CreatedAt:        u.CreatedAt.Format(time.RFC3339),
		UpdatedAt:        u.UpdatedAt.Format(time.RFC3339),
	}
	if u.OrganizationID != nil {
		s := u.OrganizationID.String()
		r.OrganizationID = &s
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

// AdminConversationResponse is the JSON response for admin conversation listing.
type AdminConversationResponse struct {
	ID                 string                             `json:"id"`
	Participants       []AdminConversationParticipantResp  `json:"participants"`
	MessageCount       int                                `json:"message_count"`
	LastMessage        *string                            `json:"last_message"`
	LastMessageAt      *string                            `json:"last_message_at"`
	PendingReportCount int                                `json:"pending_report_count"`
	ReportedMessage    *string                            `json:"reported_message,omitempty"`
	CreatedAt          string                             `json:"created_at"`
}

// AdminConversationParticipantResp is a lightweight participant in a conversation.
type AdminConversationParticipantResp struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Role        string `json:"role"`
}

// NewAdminConversationResponse converts an admin conversation to its JSON response.
func NewAdminConversationResponse(c repository.AdminConversation) AdminConversationResponse {
	participants := make([]AdminConversationParticipantResp, 0, len(c.Participants))
	for _, p := range c.Participants {
		participants = append(participants, AdminConversationParticipantResp{
			ID:          p.ID.String(),
			DisplayName: p.DisplayName,
			Email:       p.Email,
			Role:        p.Role,
		})
	}

	resp := AdminConversationResponse{
		ID:                 c.ID.String(),
		Participants:       participants,
		MessageCount:       c.MessageCount,
		LastMessage:        c.LastMessage,
		PendingReportCount: c.PendingReportCount,
		ReportedMessage:    c.ReportedMessage,
		CreatedAt:          c.CreatedAt.Format(time.RFC3339),
	}
	if c.LastMessageAt != nil {
		s := c.LastMessageAt.Format(time.RFC3339)
		resp.LastMessageAt = &s
	}
	return resp
}

// AdminMessageResponse is the JSON response for admin message viewing.
type AdminMessageResponse struct {
	ID               string           `json:"id"`
	ConversationID   string           `json:"conversation_id"`
	SenderID         string           `json:"sender_id"`
	SenderName       string           `json:"sender_name"`
	SenderRole       string           `json:"sender_role"`
	Content          string           `json:"content"`
	Type             string           `json:"type"`
	Metadata         *json.RawMessage `json:"metadata,omitempty"`
	ReplyToID        *string          `json:"reply_to_id,omitempty"`
	ModerationStatus string           `json:"moderation_status"`
	ModerationScore  float64          `json:"moderation_score"`
	ModerationLabels *json.RawMessage `json:"moderation_labels,omitempty"`
	CreatedAt        string           `json:"created_at"`
}

// NewAdminMessageResponse converts an admin message to its JSON response.
func NewAdminMessageResponse(m repository.AdminMessage) AdminMessageResponse {
	resp := AdminMessageResponse{
		ID:               m.ID.String(),
		ConversationID:   m.ConversationID.String(),
		SenderID:         m.SenderID.String(),
		SenderName:       m.SenderName,
		SenderRole:       m.SenderRole,
		Content:          m.Content,
		Type:             m.Type,
		ModerationStatus: m.ModerationStatus,
		ModerationScore:  m.ModerationScore,
		CreatedAt:        m.CreatedAt.Format(time.RFC3339),
	}
	if m.ReplyToID != nil {
		s := m.ReplyToID.String()
		resp.ReplyToID = &s
	}
	if len(m.Metadata) > 0 {
		raw := json.RawMessage(m.Metadata)
		resp.Metadata = &raw
	}
	if len(m.ModerationLabels) > 0 {
		raw := json.RawMessage(m.ModerationLabels)
		resp.ModerationLabels = &raw
	}
	return resp
}

// AdminReportResponse is the JSON response for admin report viewing.
type AdminReportResponse struct {
	ID             string  `json:"id"`
	ReporterID     string  `json:"reporter_id"`
	TargetType     string  `json:"target_type"`
	TargetID       string  `json:"target_id"`
	ConversationID *string `json:"conversation_id,omitempty"`
	Reason         string  `json:"reason"`
	Description    string  `json:"description"`
	Status         string  `json:"status"`
	AdminNote      string  `json:"admin_note"`
	ResolvedAt     *string `json:"resolved_at,omitempty"`
	ResolvedBy     *string `json:"resolved_by,omitempty"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

// NewAdminReportResponse converts a domain report to its JSON response.
func NewAdminReportResponse(r *report.Report) AdminReportResponse {
	resp := AdminReportResponse{
		ID:          r.ID.String(),
		ReporterID:  r.ReporterID.String(),
		TargetType:  string(r.TargetType),
		TargetID:    r.TargetID.String(),
		Reason:      string(r.Reason),
		Description: r.Description,
		Status:      string(r.Status),
		AdminNote:   r.AdminNote,
		CreatedAt:   r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   r.UpdatedAt.Format(time.RFC3339),
	}
	if r.ConversationID != uuid.Nil {
		s := r.ConversationID.String()
		resp.ConversationID = &s
	}
	if r.ResolvedAt != nil {
		s := r.ResolvedAt.Format(time.RFC3339)
		resp.ResolvedAt = &s
	}
	if r.ResolvedBy != nil {
		s := r.ResolvedBy.String()
		resp.ResolvedBy = &s
	}
	return resp
}
