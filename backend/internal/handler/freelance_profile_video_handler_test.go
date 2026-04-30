package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainfreelance "marketplace-backend/internal/domain/freelanceprofile"
	"marketplace-backend/internal/handler/middleware"
)

// withVideoCtx attaches the given user + organization IDs to the
// request context, mirroring what the auth middleware would do.
func withVideoCtx(req *http.Request, userID, orgID uuid.UUID) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	return req.WithContext(ctx)
}

func newTestFreelanceVideoHandler(
	storage *mockStorageService,
	repo *mockFreelanceProfileRepo,
) *FreelanceProfileVideoHandler {
	return NewFreelanceProfileVideoHandler(storage, repo, nil)
}

// ---------------------------------------------------------------------------
// Upload happy path + repository write
// ---------------------------------------------------------------------------

func TestFreelanceProfileVideoHandler_Upload_Success(t *testing.T) {
	uid := uuid.New()
	orgID := uuid.New()
	smallVideo := bytes.Repeat([]byte{0x00}, 2048)

	var savedVideoURL string
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "https://storage.example.com/profiles/intro.mp4", nil
		},
	}
	repo := &mockFreelanceProfileRepo{
		updateVideoFn: func(_ context.Context, _ uuid.UUID, videoURL string) error {
			savedVideoURL = videoURL
			return nil
		},
	}
	h := newTestFreelanceVideoHandler(storage, repo)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/freelance-profile/video",
		"file", "intro.mp4", "video/mp4", smallVideo,
	)
	req = withVideoCtx(req, uid, orgID)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "https://storage.example.com/profiles/intro.mp4", resp["video_url"])
	assert.Equal(t, "https://storage.example.com/profiles/intro.mp4", savedVideoURL)
}

// ---------------------------------------------------------------------------
// Validation failures
// ---------------------------------------------------------------------------

func TestFreelanceProfileVideoHandler_Upload_Unauthorized(t *testing.T) {
	smallVideo := bytes.Repeat([]byte{0x00}, 1024)
	h := newTestFreelanceVideoHandler(&mockStorageService{}, &mockFreelanceProfileRepo{})

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/freelance-profile/video",
		"file", "intro.mp4", "video/mp4", smallVideo,
	)
	rec := httptest.NewRecorder()
	h.Upload(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestFreelanceProfileVideoHandler_Upload_RejectsNonVideoMime(t *testing.T) {
	uid := uuid.New()
	smallFile := bytes.Repeat([]byte{0x00}, 1024)
	h := newTestFreelanceVideoHandler(&mockStorageService{}, &mockFreelanceProfileRepo{})

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/freelance-profile/video",
		"file", "doc.pdf", "application/pdf", smallFile,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "invalid_type", resp["error"])
}

func TestFreelanceProfileVideoHandler_Upload_RejectsOversizedFile(t *testing.T) {
	uid := uuid.New()
	// 51 MB exceeds the 50 MB limit.
	oversized := bytes.Repeat([]byte{0x00}, 51<<20)
	h := newTestFreelanceVideoHandler(&mockStorageService{}, &mockFreelanceProfileRepo{})

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/freelance-profile/video",
		"file", "big.mp4", "video/mp4", oversized,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	// Streaming upload returns 413 Payload Too Large on size cap.
	// (Previous ParseMultipartForm path returned 400 for the same case;
	// 413 is RFC 7231 §6.5.11 correct.)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

// ---------------------------------------------------------------------------
// Storage / DB failure paths
// ---------------------------------------------------------------------------

func TestFreelanceProfileVideoHandler_Upload_StorageFailure(t *testing.T) {
	uid := uuid.New()
	smallVideo := bytes.Repeat([]byte{0x00}, 1024)

	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "", errors.New("s3 boom")
		},
	}
	h := newTestFreelanceVideoHandler(storage, &mockFreelanceProfileRepo{})

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/freelance-profile/video",
		"file", "intro.mp4", "video/mp4", smallVideo,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestFreelanceProfileVideoHandler_Upload_DBNotFound(t *testing.T) {
	uid := uuid.New()
	smallVideo := bytes.Repeat([]byte{0x00}, 1024)
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "https://storage.example.com/profiles/intro.mp4", nil
		},
	}
	repo := &mockFreelanceProfileRepo{
		updateVideoFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			return domainfreelance.ErrProfileNotFound
		},
	}
	h := newTestFreelanceVideoHandler(storage, repo)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/freelance-profile/video",
		"file", "intro.mp4", "video/mp4", smallVideo,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestFreelanceProfileVideoHandler_Upload_DBFailure(t *testing.T) {
	uid := uuid.New()
	smallVideo := bytes.Repeat([]byte{0x00}, 1024)
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "https://storage.example.com/profiles/intro.mp4", nil
		},
	}
	repo := &mockFreelanceProfileRepo{
		updateVideoFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			return errors.New("db boom")
		},
	}
	h := newTestFreelanceVideoHandler(storage, repo)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/freelance-profile/video",
		"file", "intro.mp4", "video/mp4", smallVideo,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

// ---------------------------------------------------------------------------
// Previous-object cleanup
// ---------------------------------------------------------------------------

func TestFreelanceProfileVideoHandler_Upload_DeletesPreviousObject(t *testing.T) {
	uid := uuid.New()
	smallVideo := bytes.Repeat([]byte{0x00}, 1024)

	var deletedKey string
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "https://storage.example.com/profiles/new.mp4", nil
		},
		deleteFn: func(_ context.Context, key string) error {
			deletedKey = key
			return nil
		},
		// Use a base URL that matches the previous upload below.
		getPublicURLFn: func(key string) string {
			return "https://storage.example.com/" + key
		},
	}
	repo := &mockFreelanceProfileRepo{
		getVideoFn: func(_ context.Context, _ uuid.UUID) (string, error) {
			return "https://storage.example.com/profiles/old/video_abc.mp4", nil
		},
	}
	h := newTestFreelanceVideoHandler(storage, repo)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/freelance-profile/video",
		"file", "new.mp4", "video/mp4", smallVideo,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "profiles/old/video_abc.mp4", deletedKey)
}

// ---------------------------------------------------------------------------
// Delete endpoint
// ---------------------------------------------------------------------------

func TestFreelanceProfileVideoHandler_Delete_ClearsRow(t *testing.T) {
	uid := uuid.New()
	var savedVideo string
	repo := &mockFreelanceProfileRepo{
		updateVideoFn: func(_ context.Context, _ uuid.UUID, videoURL string) error {
			savedVideo = videoURL
			return nil
		},
	}
	h := newTestFreelanceVideoHandler(&mockStorageService{}, repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/freelance-profile/video", nil)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "", savedVideo)
}

func TestFreelanceProfileVideoHandler_Delete_Unauthorized(t *testing.T) {
	h := newTestFreelanceVideoHandler(&mockStorageService{}, &mockFreelanceProfileRepo{})
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/freelance-profile/video", nil)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestFreelanceProfileVideoHandler_Delete_NotFound(t *testing.T) {
	uid := uuid.New()
	repo := &mockFreelanceProfileRepo{
		updateVideoFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			return domainfreelance.ErrProfileNotFound
		},
	}
	h := newTestFreelanceVideoHandler(&mockStorageService{}, repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/freelance-profile/video", nil)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}
