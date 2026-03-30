package payment

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "marketplace-backend/internal/domain/payment"
)

// --- helpers ---

func validUploadInput() UploadIdentityDocumentInput {
	return UploadIdentityDocumentInput{
		Category:     string(domain.CategoryIdentity),
		DocumentType: string(domain.TypePassport),
		Side:         string(domain.SideSingle),
		Filename:     "passport.jpg",
		ContentType:  "image/jpeg",
		FileData:     []byte("fake-file-content"),
	}
}

func newIdentityTestService(
	payments *mockPaymentInfoRepo,
	docs *mockIdentityDocRepo,
	stripe *mockStripeService,
	storage *mockStorageService,
) *Service {
	return NewService(
		payments,
		&mockPaymentRecordRepo{},
		docs,
		&mockBusinessPersonRepo{},
		stripe,
		storage,
		nil,
		"",
	)
}

// --- UploadIdentityDocument tests ---

func TestUploadIdentityDocument(t *testing.T) {
	tests := []struct {
		name      string
		input     UploadIdentityDocumentInput
		payments  *mockPaymentInfoRepo
		docs      *mockIdentityDocRepo
		stripe    *mockStripeService
		storage   *mockStorageService
		assertDoc func(t *testing.T, doc *domain.IdentityDocument)
		wantErr   error
	}{
		{
			name:  "success with stripe upload",
			input: validUploadInput(),
			payments: &mockPaymentInfoRepo{
				getByUserIDFn: func(_ context.Context, _ uuid.UUID) (*domain.PaymentInfo, error) {
					return &domain.PaymentInfo{StripeAccountID: "acct_123"}, nil
				},
			},
			docs:    &mockIdentityDocRepo{},
			storage: &mockStorageService{},
			stripe: &mockStripeService{
				uploadIdentityFileFn: func(_ context.Context, _ string, _ io.Reader, _ string) (string, error) {
					return "file_stripe_abc", nil
				},
			},
			assertDoc: func(t *testing.T, doc *domain.IdentityDocument) {
				assert.Equal(t, domain.CategoryIdentity, doc.Category)
				assert.Equal(t, domain.TypePassport, doc.DocumentType)
				assert.Equal(t, domain.SideSingle, doc.Side)
				assert.Equal(t, "file_stripe_abc", doc.StripeFileID)
				assert.Contains(t, doc.FileKey, "documents/")
			},
		},
		{
			name:  "success without stripe account",
			input: validUploadInput(),
			payments: &mockPaymentInfoRepo{},
			docs:     &mockIdentityDocRepo{},
			storage:  &mockStorageService{},
			stripe:   &mockStripeService{},
			assertDoc: func(t *testing.T, doc *domain.IdentityDocument) {
				assert.Empty(t, doc.StripeFileID)
				assert.Equal(t, domain.DocStatusPending, doc.Status)
			},
		},
		{
			name:  "replaces existing document",
			input: validUploadInput(),
			payments: &mockPaymentInfoRepo{},
			docs: &mockIdentityDocRepo{
				getByUserAndTypeSideFn: func(_ context.Context, _ uuid.UUID, _, _, _ string) (*domain.IdentityDocument, error) {
					return &domain.IdentityDocument{
						ID:      uuid.New(),
						FileKey: "old-key",
					}, nil
				},
				deleteFn: func(_ context.Context, _ uuid.UUID) error {
					return nil
				},
			},
			storage: &mockStorageService{
				deleteFn: func(_ context.Context, key string) error {
					assert.Equal(t, "old-key", key)
					return nil
				},
			},
			stripe: &mockStripeService{},
			assertDoc: func(t *testing.T, doc *domain.IdentityDocument) {
				assert.NotEmpty(t, doc.FileKey)
			},
		},
		{
			name: "invalid category returns error",
			input: UploadIdentityDocumentInput{
				Category:     "invalid_cat",
				DocumentType: string(domain.TypePassport),
				Side:         string(domain.SideSingle),
				Filename:     "file.jpg",
				ContentType:  "image/jpeg",
				FileData:     []byte("data"),
			},
			payments: &mockPaymentInfoRepo{},
			docs:     &mockIdentityDocRepo{},
			storage:  &mockStorageService{},
			stripe:   &mockStripeService{},
			wantErr:  domain.ErrInvalidDocumentCategory,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newIdentityTestService(
				tt.payments, tt.docs, tt.stripe, tt.storage,
			)

			doc, err := svc.UploadIdentityDocument(
				context.Background(), uuid.New(), tt.input,
			)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				assert.Nil(t, doc)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, doc)
			if tt.assertDoc != nil {
				tt.assertDoc(t, doc)
			}
		})
	}
}

func TestUploadIdentityDocument_StorageFailure(t *testing.T) {
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "", errors.New("storage unavailable")
		},
	}
	svc := newIdentityTestService(
		&mockPaymentInfoRepo{}, &mockIdentityDocRepo{},
		&mockStripeService{}, storage,
	)

	doc, err := svc.UploadIdentityDocument(
		context.Background(), uuid.New(), validUploadInput(),
	)

	assert.Nil(t, doc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "upload to storage")
}

// --- ListIdentityDocuments tests ---

func TestListIdentityDocuments(t *testing.T) {
	tests := []struct {
		name    string
		docs    *mockIdentityDocRepo
		wantLen int
		wantErr bool
	}{
		{
			name: "returns documents",
			docs: &mockIdentityDocRepo{
				listByUserIDFn: func(_ context.Context, _ uuid.UUID) ([]*domain.IdentityDocument, error) {
					return []*domain.IdentityDocument{
						{ID: uuid.New(), Category: domain.CategoryIdentity},
						{ID: uuid.New(), Category: domain.CategoryBusiness},
					}, nil
				},
			},
			wantLen: 2,
		},
		{
			name:    "empty list",
			docs:    &mockIdentityDocRepo{},
			wantLen: 0,
		},
		{
			name: "repository error",
			docs: &mockIdentityDocRepo{
				listByUserIDFn: func(_ context.Context, _ uuid.UUID) ([]*domain.IdentityDocument, error) {
					return nil, errors.New("db down")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newIdentityTestService(
				&mockPaymentInfoRepo{}, tt.docs,
				&mockStripeService{}, &mockStorageService{},
			)

			docs, err := svc.ListIdentityDocuments(
				context.Background(), uuid.New(),
			)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Len(t, docs, tt.wantLen)
		})
	}
}

// --- DeleteIdentityDocument tests ---

func TestDeleteIdentityDocument(t *testing.T) {
	ownerID := uuid.New()
	docID := uuid.New()
	existingDoc := &domain.IdentityDocument{
		ID:      docID,
		UserID:  ownerID,
		FileKey: "documents/owner/file.jpg",
	}

	tests := []struct {
		name    string
		userID  uuid.UUID
		docID   uuid.UUID
		docs    *mockIdentityDocRepo
		wantErr string
	}{
		{
			name:   "success",
			userID: ownerID,
			docID:  docID,
			docs: &mockIdentityDocRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.IdentityDocument, error) {
					if id == docID {
						return existingDoc, nil
					}
					return nil, domain.ErrDocumentNotFound
				},
				deleteFn: func(_ context.Context, id uuid.UUID) error {
					assert.Equal(t, docID, id)
					return nil
				},
			},
		},
		{
			name:   "not found",
			userID: ownerID,
			docID:  uuid.New(),
			docs:   &mockIdentityDocRepo{},
			wantErr: domain.ErrDocumentNotFound.Error(),
		},
		{
			name:   "not owner",
			userID: uuid.New(),
			docID:  docID,
			docs: &mockIdentityDocRepo{
				getByIDFn: func(_ context.Context, id uuid.UUID) (*domain.IdentityDocument, error) {
					if id == docID {
						return existingDoc, nil
					}
					return nil, domain.ErrDocumentNotFound
				},
			},
			wantErr: "not authorized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newIdentityTestService(
				&mockPaymentInfoRepo{}, tt.docs,
				&mockStripeService{}, &mockStorageService{},
			)

			err := svc.DeleteIdentityDocument(
				context.Background(), tt.userID, tt.docID,
			)

			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// --- GetDocumentFileURL tests ---

func TestGetDocumentFileURL(t *testing.T) {
	svc := newIdentityTestService(
		&mockPaymentInfoRepo{}, &mockIdentityDocRepo{},
		&mockStripeService{}, &mockStorageService{},
	)

	url := svc.GetDocumentFileURL("documents/user/file.jpg")

	assert.Equal(t, "https://storage.example.com/documents/user/file.jpg", url)
}

func TestGetDocumentFileURL_CustomResolver(t *testing.T) {
	storage := &mockStorageService{
		getPublicURLFn: func(key string) string {
			return "https://cdn.example.com/" + key
		},
	}
	svc := newIdentityTestService(
		&mockPaymentInfoRepo{}, &mockIdentityDocRepo{},
		&mockStripeService{}, storage,
	)

	url := svc.GetDocumentFileURL("doc/key.pdf")

	assert.Equal(t, "https://cdn.example.com/doc/key.pdf", url)
}
