package payment

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewIdentityDocument(t *testing.T) {
	userID := uuid.New()

	validCases := []struct {
		name     string
		category string
		docType  string
		side     string
	}{
		{"passport single", "identity", "passport", "single"},
		{"id card front", "identity", "id_card", "front"},
		{"id card back", "identity", "id_card", "back"},
		{"driving license front", "identity", "driving_license", "front"},
		{"driving license back", "identity", "driving_license", "back"},
		{"kbis single", "business", "kbis", "single"},
		{"registration single", "business", "registration", "single"},
	}

	for _, tt := range validCases {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := NewIdentityDocument(NewIdentityDocumentInput{
				UserID:       userID,
				Category:     tt.category,
				DocumentType: tt.docType,
				Side:         tt.side,
				FileKey:      "uploads/doc.pdf",
			})

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, doc.ID)
			assert.Equal(t, userID, doc.UserID)
			assert.Equal(t, DocumentCategory(tt.category), doc.Category)
			assert.Equal(t, DocumentType(tt.docType), doc.DocumentType)
			assert.Equal(t, DocumentSide(tt.side), doc.Side)
			assert.Equal(t, "uploads/doc.pdf", doc.FileKey)
			assert.Equal(t, DocStatusPending, doc.Status)
			assert.Empty(t, doc.StripeFileID)
			assert.Empty(t, doc.RejectionReason)
		})
	}

	invalidCases := []struct {
		name     string
		category string
		docType  string
		side     string
		fileKey  string
		wantErr  error
	}{
		{"invalid category", "other", "passport", "single", "key", ErrInvalidDocumentCategory},
		{"invalid document type", "identity", "visa", "single", "key", ErrInvalidDocumentType},
		{"passport with front", "identity", "passport", "front", "key", ErrInvalidDocumentSide},
		{"passport with back", "identity", "passport", "back", "key", ErrInvalidDocumentSide},
		{"id card with single", "identity", "id_card", "single", "key", ErrInvalidDocumentSide},
		{"driving license with single", "identity", "driving_license", "single", "key", ErrInvalidDocumentSide},
		{"kbis with front", "business", "kbis", "front", "key", ErrInvalidDocumentSide},
		{"empty file key", "identity", "passport", "single", "", ErrDocumentFileKeyRequired},
	}

	for _, tt := range invalidCases {
		t.Run(tt.name, func(t *testing.T) {
			doc, err := NewIdentityDocument(NewIdentityDocumentInput{
				UserID:       userID,
				Category:     tt.category,
				DocumentType: tt.docType,
				Side:         tt.side,
				FileKey:      tt.fileKey,
			})

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Nil(t, doc)
		})
	}
}

func TestIdentityDocument_MarkVerified(t *testing.T) {
	t.Run("success from pending", func(t *testing.T) {
		doc, _ := NewIdentityDocument(NewIdentityDocumentInput{
			UserID:       uuid.New(),
			Category:     "identity",
			DocumentType: "passport",
			Side:         "single",
			FileKey:      "key",
		})

		err := doc.MarkVerified()

		require.NoError(t, err)
		assert.Equal(t, DocStatusVerified, doc.Status)
	})

	t.Run("fails from verified", func(t *testing.T) {
		doc, _ := NewIdentityDocument(NewIdentityDocumentInput{
			UserID:       uuid.New(),
			Category:     "identity",
			DocumentType: "passport",
			Side:         "single",
			FileKey:      "key",
		})
		_ = doc.MarkVerified()

		err := doc.MarkVerified()

		assert.ErrorIs(t, err, ErrDocumentNotPending)
	})

	t.Run("fails from rejected", func(t *testing.T) {
		doc, _ := NewIdentityDocument(NewIdentityDocumentInput{
			UserID:       uuid.New(),
			Category:     "identity",
			DocumentType: "passport",
			Side:         "single",
			FileKey:      "key",
		})
		_ = doc.MarkRejected("bad quality")

		err := doc.MarkVerified()

		assert.ErrorIs(t, err, ErrDocumentNotPending)
	})
}

func TestIdentityDocument_MarkRejected(t *testing.T) {
	t.Run("success from pending", func(t *testing.T) {
		doc, _ := NewIdentityDocument(NewIdentityDocumentInput{
			UserID:       uuid.New(),
			Category:     "identity",
			DocumentType: "passport",
			Side:         "single",
			FileKey:      "key",
		})

		err := doc.MarkRejected("blurry image")

		require.NoError(t, err)
		assert.Equal(t, DocStatusRejected, doc.Status)
		assert.Equal(t, "blurry image", doc.RejectionReason)
	})

	t.Run("fails from verified", func(t *testing.T) {
		doc, _ := NewIdentityDocument(NewIdentityDocumentInput{
			UserID:       uuid.New(),
			Category:     "identity",
			DocumentType: "passport",
			Side:         "single",
			FileKey:      "key",
		})
		_ = doc.MarkVerified()

		err := doc.MarkRejected("reason")

		assert.ErrorIs(t, err, ErrDocumentNotPending)
	})
}

func TestIdentityDocument_SetStripeFileID(t *testing.T) {
	doc, _ := NewIdentityDocument(NewIdentityDocumentInput{
		UserID:       uuid.New(),
		Category:     "identity",
		DocumentType: "passport",
		Side:         "single",
		FileKey:      "key",
	})

	doc.SetStripeFileID("file_abc123")

	assert.Equal(t, "file_abc123", doc.StripeFileID)
}

func TestRequiresBothSides(t *testing.T) {
	tests := []struct {
		docType DocumentType
		want    bool
	}{
		{TypePassport, false},
		{TypeIDCard, true},
		{TypeDrivingLicense, true},
		{TypeKBIS, false},
		{TypeRegistration, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.docType), func(t *testing.T) {
			assert.Equal(t, tt.want, RequiresBothSides(tt.docType))
		})
	}
}
