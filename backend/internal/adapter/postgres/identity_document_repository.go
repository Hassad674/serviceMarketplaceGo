package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/payment"
)

type IdentityDocumentRepository struct {
	db *sql.DB
}

func NewIdentityDocumentRepository(db *sql.DB) *IdentityDocumentRepository {
	return &IdentityDocumentRepository{db: db}
}

func (r *IdentityDocumentRepository) Create(ctx context.Context, doc *payment.IdentityDocument) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO identity_documents (id, user_id, category, document_type, side, file_key, stripe_file_id, status, rejection_reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		doc.ID, doc.UserID, string(doc.Category), string(doc.DocumentType), string(doc.Side),
		doc.FileKey, nullableStr(doc.StripeFileID), string(doc.Status), nullableStr(doc.RejectionReason),
		doc.CreatedAt, doc.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert identity document: %w", err)
	}
	return nil
}

func (r *IdentityDocumentRepository) GetByID(ctx context.Context, id uuid.UUID) (*payment.IdentityDocument, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.scanDoc(r.db.QueryRowContext(ctx, `
		SELECT id, user_id, category, document_type, side, file_key,
			COALESCE(stripe_file_id, ''), status, COALESCE(rejection_reason, ''),
			created_at, updated_at
		FROM identity_documents WHERE id = $1`, id))
}

func (r *IdentityDocumentRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*payment.IdentityDocument, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, category, document_type, side, file_key,
			COALESCE(stripe_file_id, ''), status, COALESCE(rejection_reason, ''),
			created_at, updated_at
		FROM identity_documents WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("list identity documents: %w", err)
	}
	defer rows.Close()

	var docs []*payment.IdentityDocument
	for rows.Next() {
		doc, err := r.scanDocRow(rows)
		if err != nil {
			return nil, err
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func (r *IdentityDocumentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx, `DELETE FROM identity_documents WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete identity document: %w", err)
	}
	return nil
}

func (r *IdentityDocumentRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, rejectionReason string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE identity_documents SET status = $1, rejection_reason = $2 WHERE id = $3`,
		status, nullableStr(rejectionReason), id)
	if err != nil {
		return fmt.Errorf("update document status: %w", err)
	}
	return nil
}

func (r *IdentityDocumentRepository) UpdateStripeFileID(ctx context.Context, id uuid.UUID, stripeFileID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := r.db.ExecContext(ctx,
		`UPDATE identity_documents SET stripe_file_id = $1 WHERE id = $2`,
		stripeFileID, id)
	if err != nil {
		return fmt.Errorf("update stripe file id: %w", err)
	}
	return nil
}

func (r *IdentityDocumentRepository) GetByUserAndTypeSide(ctx context.Context, userID uuid.UUID, category, docType, side string) (*payment.IdentityDocument, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return r.scanDoc(r.db.QueryRowContext(ctx, `
		SELECT id, user_id, category, document_type, side, file_key,
			COALESCE(stripe_file_id, ''), status, COALESCE(rejection_reason, ''),
			created_at, updated_at
		FROM identity_documents
		WHERE user_id = $1 AND category = $2 AND document_type = $3 AND side = $4`,
		userID, category, docType, side))
}

func (r *IdentityDocumentRepository) scanDoc(row *sql.Row) (*payment.IdentityDocument, error) {
	var doc payment.IdentityDocument
	var cat, docType, side, status string

	err := row.Scan(
		&doc.ID, &doc.UserID, &cat, &docType, &side, &doc.FileKey,
		&doc.StripeFileID, &status, &doc.RejectionReason,
		&doc.CreatedAt, &doc.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, payment.ErrDocumentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("scan identity document: %w", err)
	}

	doc.Category = payment.DocumentCategory(cat)
	doc.DocumentType = payment.DocumentType(docType)
	doc.Side = payment.DocumentSide(side)
	doc.Status = payment.DocumentStatus(status)
	return &doc, nil
}

func (r *IdentityDocumentRepository) scanDocRow(rows *sql.Rows) (*payment.IdentityDocument, error) {
	var doc payment.IdentityDocument
	var cat, docType, side, status string

	err := rows.Scan(
		&doc.ID, &doc.UserID, &cat, &docType, &side, &doc.FileKey,
		&doc.StripeFileID, &status, &doc.RejectionReason,
		&doc.CreatedAt, &doc.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan identity document row: %w", err)
	}

	doc.Category = payment.DocumentCategory(cat)
	doc.DocumentType = payment.DocumentType(docType)
	doc.Side = payment.DocumentSide(side)
	doc.Status = payment.DocumentStatus(status)
	return &doc, nil
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
