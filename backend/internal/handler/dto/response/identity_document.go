package response

import (
	"time"

	"marketplace-backend/internal/domain/payment"
)

type IdentityDocumentResponse struct {
	ID              string `json:"id"`
	UserID          string `json:"user_id"`
	Category        string `json:"category"`
	DocumentType    string `json:"document_type"`
	Side            string `json:"side"`
	FileURL         string `json:"file_url"`
	Status          string `json:"status"`
	RejectionReason string `json:"rejection_reason,omitempty"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

func NewIdentityDocumentResponse(doc *payment.IdentityDocument, fileURL string) IdentityDocumentResponse {
	return IdentityDocumentResponse{
		ID:              doc.ID.String(),
		UserID:          doc.UserID.String(),
		Category:        string(doc.Category),
		DocumentType:    string(doc.DocumentType),
		Side:            string(doc.Side),
		FileURL:         fileURL,
		Status:          string(doc.Status),
		RejectionReason: doc.RejectionReason,
		CreatedAt:       doc.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       doc.UpdatedAt.Format(time.RFC3339),
	}
}
