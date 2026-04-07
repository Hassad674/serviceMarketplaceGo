package response

import (
	"time"

	"marketplace-backend/internal/port/repository"
)

// ModerationItemResponse is the JSON response for a unified moderation item.
type ModerationItemResponse struct {
	ID              string  `json:"id"`
	Source          string  `json:"source"`
	ContentType     string  `json:"content_type"`
	ContentID       string  `json:"content_id"`
	ContentPreview  string  `json:"content_preview"`
	ContentURL      string  `json:"content_url"`
	Status          string  `json:"status"`
	ModerationScore float64 `json:"moderation_score"`
	Reason          string  `json:"reason"`
	UserInvolved    ModerationUserBrief `json:"user_involved"`
	ConversationID  *string `json:"conversation_id,omitempty"`
	CreatedAt       string  `json:"created_at"`
}

// ModerationUserBrief is a lightweight user summary embedded in a moderation item response.
type ModerationUserBrief struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role"`
}

// NewModerationItemResponse converts a ModerationItem to its JSON response.
func NewModerationItemResponse(item repository.ModerationItem) ModerationItemResponse {
	resp := ModerationItemResponse{
		ID:              item.ID.String(),
		Source:          string(item.Source),
		ContentType:     item.ContentType,
		ContentID:       item.ContentID.String(),
		ContentPreview:  item.ContentPreview,
		ContentURL:      computeContentURL(item),
		Status:          item.Status,
		ModerationScore: item.ModerationScore,
		Reason:          item.Reason,
		UserInvolved: ModerationUserBrief{
			ID:          item.UserInvolvedID.String(),
			DisplayName: item.UserInvolvedName,
			Role:        item.UserInvolvedRole,
		},
		CreatedAt: item.CreatedAt.Format(time.RFC3339),
	}

	if item.ConversationID != nil {
		s := item.ConversationID.String()
		resp.ConversationID = &s
	}

	return resp
}

// computeContentURL generates a frontend-friendly URL based on the source and content type.
func computeContentURL(item repository.ModerationItem) string {
	switch item.ContentType {
	case "report":
		return "/reports/" + item.ContentID.String()
	case "message":
		if item.ConversationID != nil {
			return "/conversations/" + item.ConversationID.String()
		}
		return ""
	case "review":
		return "/reviews/" + item.ContentID.String()
	case "media":
		return "/media/" + item.ContentID.String()
	default:
		return ""
	}
}
