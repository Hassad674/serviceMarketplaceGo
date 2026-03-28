package response

import (
	"encoding/json"
	"time"

	"marketplace-backend/internal/domain/notification"
)

type NotificationResponse struct {
	ID        string          `json:"id"`
	UserID    string          `json:"user_id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      string          `json:"body"`
	Data      json.RawMessage `json:"data"`
	ReadAt    *time.Time      `json:"read_at"`
	CreatedAt time.Time       `json:"created_at"`
}

func NotificationFromDomain(n *notification.Notification) NotificationResponse {
	return NotificationResponse{
		ID:        n.ID.String(),
		UserID:    n.UserID.String(),
		Type:      string(n.Type),
		Title:     n.Title,
		Body:      n.Body,
		Data:      n.Data,
		ReadAt:    n.ReadAt,
		CreatedAt: n.CreatedAt,
	}
}

type NotificationPreferenceResponse struct {
	Type  string `json:"type"`
	InApp bool   `json:"in_app"`
	Push  bool   `json:"push"`
	Email bool   `json:"email"`
}

func PreferenceFromDomain(p *notification.Preferences) NotificationPreferenceResponse {
	return NotificationPreferenceResponse{
		Type:  string(p.NotificationType),
		InApp: p.InApp,
		Push:  p.Push,
		Email: p.Email,
	}
}
