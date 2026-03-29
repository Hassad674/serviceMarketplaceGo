package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/middleware"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestUploadHandler(
	storage *mockStorageService,
	profiles *mockProfileRepo,
) *UploadHandler {
	return NewUploadHandler(storage, profiles)
}

// buildMultipartRequest creates a multipart form request with a file field.
func buildMultipartRequest(
	method, url, fieldName, fileName, contentType string,
	fileContent []byte,
) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
	header.Set("Content-Type", contentType)

	part, _ := writer.CreatePart(header)
	part.Write(fileContent)
	writer.Close()

	req := httptest.NewRequest(method, url, body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// ---------------------------------------------------------------------------
// UploadPhoto tests
// ---------------------------------------------------------------------------

func TestUploadHandler_UploadPhoto(t *testing.T) {
	uid := uuid.New()
	smallImage := bytes.Repeat([]byte{0xFF}, 1024) // 1 KB fake image

	tests := []struct {
		name       string
		userID     *uuid.UUID
		fileName   string
		mime       string
		content    []byte
		setupMocks func(*mockStorageService, *mockProfileRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:     "success",
			userID:   &uid,
			fileName: "photo.jpg",
			mime:     "image/jpeg",
			content:  smallImage,
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/photo.jpg", nil
				}
				p.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			fileName:   "photo.jpg",
			mime:       "image/jpeg",
			content:    smallImage,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid mime type",
			userID:     &uid,
			fileName:   "file.pdf",
			mime:       "application/pdf",
			content:    smallImage,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_type",
		},
		{
			name:     "upload failure",
			userID:   &uid,
			fileName: "photo.jpg",
			mime:     "image/jpeg",
			content:  smallImage,
			setupMocks: func(s *mockStorageService, _ *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
					return "", errors.New("storage error")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "upload_failed",
		},
		{
			name:     "profile not found after upload",
			userID:   &uid,
			fileName: "photo.jpg",
			mime:     "image/jpeg",
			content:  smallImage,
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/photo.jpg", nil
				}
				p.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return nil, profile.ErrProfileNotFound
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "profile_error",
		},
		{
			name:     "profile update failure",
			userID:   &uid,
			fileName: "photo.jpg",
			mime:     "image/jpeg",
			content:  smallImage,
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/photo.jpg", nil
				}
				p.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
				p.updateFn = func(_ context.Context, _ *profile.Profile) error {
					return errors.New("db failure")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "update_failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			storage := &mockStorageService{}
			profiles := &mockProfileRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(storage, profiles)
			}
			h := newTestUploadHandler(storage, profiles)

			req := buildMultipartRequest(
				http.MethodPost, "/api/v1/upload/photo",
				"file", tc.fileName, tc.mime, tc.content,
			)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.UploadPhoto(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}

			if tc.wantStatus == http.StatusOK {
				var resp map[string]string
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.NotEmpty(t, resp["url"])
			}
		})
	}
}

// ---------------------------------------------------------------------------
// UploadPhoto file too large (separate test: needs oversized body)
// ---------------------------------------------------------------------------

func TestUploadHandler_UploadPhoto_FileTooLarge(t *testing.T) {
	uid := uuid.New()
	// 6 MB exceeds the 5 MB limit
	oversized := bytes.Repeat([]byte{0xFF}, 6<<20)

	storage := &mockStorageService{}
	profiles := &mockProfileRepo{}
	h := newTestUploadHandler(storage, profiles)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/upload/photo",
		"file", "big.jpg", "image/jpeg", oversized,
	)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UploadPhoto(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

// ---------------------------------------------------------------------------
// UploadVideo tests
// ---------------------------------------------------------------------------

func TestUploadHandler_UploadVideo(t *testing.T) {
	uid := uuid.New()
	smallVideo := bytes.Repeat([]byte{0x00}, 2048) // 2 KB fake video

	tests := []struct {
		name       string
		userID     *uuid.UUID
		fileName   string
		mime       string
		content    []byte
		setupMocks func(*mockStorageService, *mockProfileRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:     "success",
			userID:   &uid,
			fileName: "intro.mp4",
			mime:     "video/mp4",
			content:  smallVideo,
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/intro.mp4", nil
				}
				p.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			fileName:   "intro.mp4",
			mime:       "video/mp4",
			content:    smallVideo,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "invalid mime type",
			userID:     &uid,
			fileName:   "doc.pdf",
			mime:       "application/pdf",
			content:    smallVideo,
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_type",
		},
		{
			name:     "upload failure",
			userID:   &uid,
			fileName: "intro.mp4",
			mime:     "video/mp4",
			content:  smallVideo,
			setupMocks: func(s *mockStorageService, _ *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
					return "", errors.New("storage error")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "upload_failed",
		},
		{
			name:     "profile not found after upload",
			userID:   &uid,
			fileName: "intro.mp4",
			mime:     "video/mp4",
			content:  smallVideo,
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/intro.mp4", nil
				}
				p.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return nil, profile.ErrProfileNotFound
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "profile_error",
		},
		{
			name:     "profile update failure",
			userID:   &uid,
			fileName: "intro.mp4",
			mime:     "video/mp4",
			content:  smallVideo,
			setupMocks: func(s *mockStorageService, p *mockProfileRepo) {
				s.uploadFn = func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
					return "https://storage.example.com/intro.mp4", nil
				}
				p.getByUserIDFn = func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
					return testProfile(uid), nil
				}
				p.updateFn = func(_ context.Context, _ *profile.Profile) error {
					return errors.New("db failure")
				}
			},
			wantStatus: http.StatusInternalServerError,
			wantCode:   "update_failed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			storage := &mockStorageService{}
			profiles := &mockProfileRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(storage, profiles)
			}
			h := newTestUploadHandler(storage, profiles)

			req := buildMultipartRequest(
				http.MethodPost, "/api/v1/upload/video",
				"file", tc.fileName, tc.mime, tc.content,
			)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.UploadVideo(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}

			if tc.wantStatus == http.StatusOK {
				var resp map[string]string
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.NotEmpty(t, resp["url"])
			}
		})
	}
}
