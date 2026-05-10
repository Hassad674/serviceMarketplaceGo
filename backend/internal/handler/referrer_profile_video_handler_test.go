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
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domainreferrer "marketplace-backend/internal/domain/referrerprofile"
	"marketplace-backend/internal/handler/middleware"
)

func newTestReferrerVideoHandler(
	storage *mockStorageService,
	repo *mockReferrerProfileRepo,
) *ReferrerProfileVideoHandler {
	return NewReferrerProfileVideoHandler(storage, repo, nil)
}

func TestReferrerProfileVideoHandler_Upload_Success(t *testing.T) {
	uid := uuid.New()
	smallVideo := bytes.Repeat([]byte{0x00}, 2048)

	var savedVideoURL string
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "https://storage.example.com/profiles/referrer.mp4", nil
		},
	}
	repo := &mockReferrerProfileRepo{
		updateVideoFn: func(_ context.Context, _ uuid.UUID, videoURL string) error {
			savedVideoURL = videoURL
			return nil
		},
	}
	h := newTestReferrerVideoHandler(storage, repo)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/referrer-profile/video",
		"file", "intro.mp4", "video/mp4", smallVideo,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "https://storage.example.com/profiles/referrer.mp4", resp["video_url"])
	assert.Equal(t, "https://storage.example.com/profiles/referrer.mp4", savedVideoURL)
}

func TestReferrerProfileVideoHandler_Upload_Unauthorized(t *testing.T) {
	smallVideo := bytes.Repeat([]byte{0x00}, 1024)
	h := newTestReferrerVideoHandler(&mockStorageService{}, &mockReferrerProfileRepo{})

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/referrer-profile/video",
		"file", "intro.mp4", "video/mp4", smallVideo,
	)
	rec := httptest.NewRecorder()
	h.Upload(rec, req)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestReferrerProfileVideoHandler_Upload_RejectsNonVideoMime(t *testing.T) {
	uid := uuid.New()
	smallFile := bytes.Repeat([]byte{0x00}, 1024)
	h := newTestReferrerVideoHandler(&mockStorageService{}, &mockReferrerProfileRepo{})

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/referrer-profile/video",
		"file", "doc.pdf", "application/pdf", smallFile,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestReferrerProfileVideoHandler_Upload_RejectsOversizedFile(t *testing.T) {
	uid := uuid.New()
	oversized := bytes.Repeat([]byte{0x00}, 51<<20)
	h := newTestReferrerVideoHandler(&mockStorageService{}, &mockReferrerProfileRepo{})

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/referrer-profile/video",
		"file", "big.mp4", "video/mp4", oversized,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	// Post-G120 streaming surfaces oversized payloads as 413
	// Payload Too Large (RFC 7231 §6.5.11).
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestReferrerProfileVideoHandler_Upload_StorageFailure(t *testing.T) {
	uid := uuid.New()
	smallVideo := bytes.Repeat([]byte{0x00}, 1024)
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "", errors.New("s3 boom")
		},
	}
	h := newTestReferrerVideoHandler(storage, &mockReferrerProfileRepo{})

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/referrer-profile/video",
		"file", "intro.mp4", "video/mp4", smallVideo,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestReferrerProfileVideoHandler_Upload_DBFailure(t *testing.T) {
	uid := uuid.New()
	smallVideo := bytes.Repeat([]byte{0x00}, 1024)
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "https://storage.example.com/referrer.mp4", nil
		},
	}
	repo := &mockReferrerProfileRepo{
		updateVideoFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			return errors.New("db boom")
		},
	}
	h := newTestReferrerVideoHandler(storage, repo)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/referrer-profile/video",
		"file", "intro.mp4", "video/mp4", smallVideo,
	)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Upload(rec, req)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestReferrerProfileVideoHandler_Delete_ClearsRow(t *testing.T) {
	uid := uuid.New()
	var savedVideo string
	repo := &mockReferrerProfileRepo{
		updateVideoFn: func(_ context.Context, _ uuid.UUID, videoURL string) error {
			savedVideo = videoURL
			return nil
		},
	}
	h := newTestReferrerVideoHandler(&mockStorageService{}, repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/referrer-profile/video", nil)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code)
	assert.Equal(t, "", savedVideo)
}

func TestReferrerProfileVideoHandler_Delete_NotFound(t *testing.T) {
	uid := uuid.New()
	repo := &mockReferrerProfileRepo{
		updateVideoFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			return domainreferrer.ErrProfileNotFound
		},
	}
	h := newTestReferrerVideoHandler(&mockStorageService{}, repo)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/referrer-profile/video", nil)
	req = withVideoCtx(req, uid, uid)
	rec := httptest.NewRecorder()

	h.Delete(rec, req)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

// ---------------------------------------------------------------------------
// Goroutine context propagation (CodeQL #64)
// ---------------------------------------------------------------------------

// referrerVideoCtxKey is a private sentinel used only inside the
// ctx-propagation tests so collisions with production keys are
// impossible.
type referrerVideoCtxKey struct{}

// TestReferrerProfileVideoHandler_Upload_GoroutineInheritsRequestValues
// asserts that the moderation goroutine receives a context derived
// from r.Context() (carries request-scoped values) and that the
// goroutine's context is NOT cancelled by the request finishing.
// Closes CodeQL #64 (go/goroutine-with-background-context).
func TestReferrerProfileVideoHandler_Upload_GoroutineInheritsRequestValues(t *testing.T) {
	uid := uuid.New()
	orgID := uuid.New()
	smallVideo := bytes.Repeat([]byte{0x00}, 1024)

	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "https://storage.example.com/profiles/referrer.mp4", nil
		},
	}
	repo := &mockReferrerProfileRepo{
		updateVideoFn: func(_ context.Context, _ uuid.UUID, _ string) error { return nil },
	}
	h := NewReferrerProfileVideoHandler(storage, repo, nil)

	rec := newFakeRecorder()
	h.withRecorder(rec)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/referrer-profile/video",
		"file", "intro.mp4", "video/mp4", smallVideo,
	)
	reqCtx := context.WithValue(req.Context(), referrerVideoCtxKey{}, "carry-me")
	reqCtx = context.WithValue(reqCtx, middleware.ContextKeyUserID, uid)
	reqCtx = context.WithValue(reqCtx, middleware.ContextKeyOrganizationID, orgID)
	reqCtx, cancelReq := context.WithCancel(reqCtx)
	defer cancelReq()
	req = req.WithContext(reqCtx)

	w := httptest.NewRecorder()
	h.Upload(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Cancel the request context AFTER the response is written.
	// context.WithoutCancel must shield the goroutine from this
	// cancellation — otherwise the moderation pipeline aborts before
	// it can finish.
	cancelReq()

	select {
	case <-rec.done:
	case <-time.After(2 * time.Second):
		t.Fatal("RecordUpload goroutine never started")
	}

	rec.mu.Lock()
	require.Len(t, rec.calls, 1, "RecordUpload must run exactly once")
	rec.mu.Unlock()

	gotCtx := rec.lastCtx()
	require.NotNil(t, gotCtx, "fakeRecorder must have captured a context")
	assert.Equal(t, "carry-me", gotCtx.Value(referrerVideoCtxKey{}),
		"goroutine ctx must inherit request values (WithoutCancel preserves baggage)")
	assert.NoError(t, gotCtx.Err(),
		"goroutine ctx must NOT be cancelled when request ctx cancels (fire-and-forget)")
}
