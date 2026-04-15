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

	domainreferrer "marketplace-backend/internal/domain/referrerprofile"
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
	assert.Equal(t, http.StatusBadRequest, rec.Code)
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
