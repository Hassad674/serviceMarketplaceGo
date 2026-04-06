package response

import (
	"time"

	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/port/repository"
)

// AdminMediaResponse is the JSON response for admin media listing.
type AdminMediaResponse struct {
	ID               string                `json:"id"`
	UploaderID       string                `json:"uploader_id"`
	FileURL          string                `json:"file_url"`
	FileName         string                `json:"file_name"`
	FileType         string                `json:"file_type"`
	FileSize         int64                 `json:"file_size"`
	Context          string                `json:"context"`
	ContextID        *string               `json:"context_id,omitempty"`
	ModerationStatus string                `json:"moderation_status"`
	ModerationLabels []ModerationLabelResp `json:"moderation_labels"`
	ModerationScore  float64               `json:"moderation_score"`
	ReviewedAt       *string               `json:"reviewed_at,omitempty"`
	ReviewedBy       *string               `json:"reviewed_by,omitempty"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
	// Uploader info (from JOIN)
	UploaderDisplayName string `json:"uploader_display_name"`
	UploaderEmail       string `json:"uploader_email"`
	UploaderRole        string `json:"uploader_role"`
}

// ModerationLabelResp is a single moderation label in the API response.
type ModerationLabelResp struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	ParentName string  `json:"parent_name,omitempty"`
}

// NewAdminMediaResponse converts an AdminMediaItem to its JSON response.
func NewAdminMediaResponse(item repository.AdminMediaItem) AdminMediaResponse {
	resp := AdminMediaResponse{
		ID:                  item.ID.String(),
		UploaderID:          item.UploaderID.String(),
		FileURL:             item.FileURL,
		FileName:            item.FileName,
		FileType:            item.FileType,
		FileSize:            item.FileSize,
		Context:             string(item.Context),
		ModerationStatus:    string(item.ModerationStatus),
		ModerationLabels:    convertLabels(item.ModerationLabels),
		ModerationScore:     item.ModerationScore,
		CreatedAt:           item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:           item.UpdatedAt.Format(time.RFC3339),
		UploaderDisplayName: item.UploaderDisplayName,
		UploaderEmail:       item.UploaderEmail,
		UploaderRole:        item.UploaderRole,
	}
	if item.ContextID != nil {
		s := item.ContextID.String()
		resp.ContextID = &s
	}
	if item.ReviewedAt != nil {
		s := item.ReviewedAt.Format(time.RFC3339)
		resp.ReviewedAt = &s
	}
	if item.ReviewedBy != nil {
		s := item.ReviewedBy.String()
		resp.ReviewedBy = &s
	}
	return resp
}

func convertLabels(labels []mediadomain.ModerationLabel) []ModerationLabelResp {
	if labels == nil {
		return []ModerationLabelResp{}
	}
	result := make([]ModerationLabelResp, len(labels))
	for i, l := range labels {
		result[i] = ModerationLabelResp{
			Name:       l.Name,
			Confidence: l.Confidence,
			ParentName: l.ParentName,
		}
	}
	return result
}
