package payment

import (
	"time"

	"github.com/google/uuid"
)

type DocumentCategory string

const (
	CategoryIdentity DocumentCategory = "identity"
	CategoryBusiness DocumentCategory = "business"
)

type DocumentType string

const (
	TypePassport       DocumentType = "passport"
	TypeIDCard         DocumentType = "id_card"
	TypeDrivingLicense DocumentType = "driving_license"
	TypeKBIS           DocumentType = "kbis"
	TypeRegistration   DocumentType = "registration"
)

type DocumentSide string

const (
	SideFront  DocumentSide = "front"
	SideBack   DocumentSide = "back"
	SideSingle DocumentSide = "single"
)

type DocumentStatus string

const (
	DocStatusPending  DocumentStatus = "pending"
	DocStatusVerified DocumentStatus = "verified"
	DocStatusRejected DocumentStatus = "rejected"
)

type IdentityDocument struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Category        DocumentCategory
	DocumentType    DocumentType
	Side            DocumentSide
	FileKey         string
	StripeFileID    string
	Status          DocumentStatus
	RejectionReason string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type NewIdentityDocumentInput struct {
	UserID       uuid.UUID
	Category     string
	DocumentType string
	Side         string
	FileKey      string
}

func NewIdentityDocument(input NewIdentityDocumentInput) (*IdentityDocument, error) {
	cat := DocumentCategory(input.Category)
	if !isValidCategory(cat) {
		return nil, ErrInvalidDocumentCategory
	}

	docType := DocumentType(input.DocumentType)
	if !isValidDocumentType(docType) {
		return nil, ErrInvalidDocumentType
	}

	side := DocumentSide(input.Side)
	if !isValidSideForType(docType, side) {
		return nil, ErrInvalidDocumentSide
	}

	if input.FileKey == "" {
		return nil, ErrDocumentFileKeyRequired
	}

	now := time.Now()
	return &IdentityDocument{
		ID:           uuid.New(),
		UserID:       input.UserID,
		Category:     cat,
		DocumentType: docType,
		Side:         side,
		FileKey:      input.FileKey,
		Status:       DocStatusPending,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (d *IdentityDocument) MarkVerified() error {
	if d.Status != DocStatusPending {
		return ErrDocumentNotPending
	}
	d.Status = DocStatusVerified
	d.UpdatedAt = time.Now()
	return nil
}

func (d *IdentityDocument) MarkRejected(reason string) error {
	if d.Status != DocStatusPending {
		return ErrDocumentNotPending
	}
	d.Status = DocStatusRejected
	d.RejectionReason = reason
	d.UpdatedAt = time.Now()
	return nil
}

func (d *IdentityDocument) SetStripeFileID(id string) {
	d.StripeFileID = id
	d.UpdatedAt = time.Now()
}

func RequiresBothSides(docType DocumentType) bool {
	return docType == TypeIDCard || docType == TypeDrivingLicense
}

func isValidCategory(c DocumentCategory) bool {
	return c == CategoryIdentity || c == CategoryBusiness
}

func isValidDocumentType(t DocumentType) bool {
	switch t {
	case TypePassport, TypeIDCard, TypeDrivingLicense, TypeKBIS, TypeRegistration:
		return true
	}
	return false
}

func isValidSideForType(t DocumentType, s DocumentSide) bool {
	switch t {
	case TypePassport, TypeKBIS, TypeRegistration:
		return s == SideSingle
	case TypeIDCard, TypeDrivingLicense:
		return s == SideFront || s == SideBack
	}
	return false
}
