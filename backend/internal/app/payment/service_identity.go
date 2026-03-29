package payment

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/google/uuid"

	domain "marketplace-backend/internal/domain/payment"
)

type UploadIdentityDocumentInput struct {
	Category     string
	DocumentType string
	Side         string
	Filename     string
	ContentType  string
	FileData     []byte // buffered file content
}

// UploadIdentityDocument handles the full upload flow: R2 + Stripe + DB.
func (s *Service) UploadIdentityDocument(ctx context.Context, userID uuid.UUID, input UploadIdentityDocumentInput) (*domain.IdentityDocument, error) {
	// Get payment info for Stripe account ID
	info, err := s.payments.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get payment info: %w", err)
	}
	if info.StripeAccountID == "" {
		return nil, domain.ErrStripeAccountNotFound
	}

	// Generate storage key
	ext := filepath.Ext(input.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	storageKey := fmt.Sprintf("documents/%s/%s_%s_%s_%s%s",
		userID, input.Category, input.DocumentType, input.Side, uuid.New().String()[:8], ext)

	// Upload to R2/S3
	_, err = s.storage.Upload(ctx, storageKey, bytes.NewReader(input.FileData), input.ContentType, int64(len(input.FileData)))
	if err != nil {
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	// Upload to Stripe
	var stripeFileID string
	if s.stripe != nil {
		stripeFileID, err = s.stripe.UploadIdentityFile(ctx, input.Filename, bytes.NewReader(input.FileData), "identity_document")
		if err != nil {
			slog.Error("failed to upload to stripe", "user_id", userID, "error", err)
			// Continue — R2 upload succeeded, Stripe can be retried
		}
	}

	// Delete existing document for same type+side (replace semantics)
	existing, _ := s.documents.GetByUserAndTypeSide(ctx, userID, input.Category, input.DocumentType, input.Side)
	if existing != nil {
		_ = s.storage.Delete(ctx, existing.FileKey)
		_ = s.documents.Delete(ctx, existing.ID)
	}

	// Create domain entity
	doc, err := domain.NewIdentityDocument(domain.NewIdentityDocumentInput{
		UserID:       userID,
		Category:     input.Category,
		DocumentType: input.DocumentType,
		Side:         input.Side,
		FileKey:      storageKey,
	})
	if err != nil {
		return nil, err
	}

	if stripeFileID != "" {
		doc.SetStripeFileID(stripeFileID)
	}

	// Persist to DB
	if err := s.documents.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("persist identity document: %w", err)
	}

	// Try to attach to Stripe account
	if s.stripe != nil && stripeFileID != "" {
		s.attachDocumentToAccount(ctx, userID, info.StripeAccountID)
	}

	return doc, nil
}

// attachDocumentToAccount gathers all uploaded docs and attaches to Stripe.
func (s *Service) attachDocumentToAccount(ctx context.Context, userID uuid.UUID, accountID string) {
	docs, err := s.documents.ListByUserID(ctx, userID)
	if err != nil {
		return
	}

	var frontID, backID string
	for _, d := range docs {
		if d.Category != domain.CategoryIdentity || d.StripeFileID == "" {
			continue
		}
		switch d.Side {
		case domain.SideFront, domain.SideSingle:
			frontID = d.StripeFileID
		case domain.SideBack:
			backID = d.StripeFileID
		}
	}

	if frontID == "" {
		return
	}

	if err := s.stripe.UpdateAccountVerification(ctx, accountID, frontID, backID); err != nil {
		slog.Error("failed to attach verification docs to stripe", "account_id", accountID, "error", err)
	}
}

// ListIdentityDocuments returns all documents for a user.
// Status updates come exclusively from the Stripe webhook (account.updated).
func (s *Service) ListIdentityDocuments(ctx context.Context, userID uuid.UUID) ([]*domain.IdentityDocument, error) {
	docs, err := s.documents.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list identity documents: %w", err)
	}
	return docs, nil
}

// DeleteIdentityDocument deletes a document from storage and DB.
func (s *Service) DeleteIdentityDocument(ctx context.Context, userID uuid.UUID, docID uuid.UUID) error {
	doc, err := s.documents.GetByID(ctx, docID)
	if err != nil {
		return err
	}
	if doc.UserID != userID {
		return fmt.Errorf("not authorized to delete this document")
	}

	_ = s.storage.Delete(ctx, doc.FileKey)
	return s.documents.Delete(ctx, docID)
}

// GetDocumentFileURL returns the public URL for a document.
func (s *Service) GetDocumentFileURL(key string) string {
	return s.storage.GetPublicURL(key)
}
