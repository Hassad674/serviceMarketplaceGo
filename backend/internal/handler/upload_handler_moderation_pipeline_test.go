package handler

// Regression for the moderation-pipeline bypass: after the BUG-17
// trackUpload refactor, the helper hard-coded `""` for `fileName`
// when invoking RecordUpload. The media domain entity validates
// `FileName` as required and `mediadomain.NewMedia` rejects empty
// values with `ErrMissingFileName`. The result was that every
// /upload/photo (and friends) silently created NO media row, ran
// NO Rekognition analysis, and produced NO admin queue entry —
// while the HTTP response was still 200.
//
// These tests pin three guarantees:
//
//  1. TrackUpload_ForwardsFileName: the FileName supplied via
//     trackUploadInput is the value the recorder receives. If a
//     future refactor re-introduces an empty-string short-circuit,
//     this test fails.
//  2. UploadPhoto_RecordsMediaWithNonEmptyFileName: the
//     end-to-end /upload/photo request path forwards a non-empty
//     FileName derived from the storage key. This is the test
//     yesterday's "fix" was missing — the slog.Info added in
//     1f64b78f only fired AFTER NewMedia succeeded, so an upstream
//     entity-creation failure left no log trace.
//  3. NewMedia_RejectsEmptyFileName_DocsTheRegression:
//     a documentation test exercising domain.NewMedia with
//     FileName="" so the failure mode is permanently captured in
//     the test corpus. If domain validation ever loosens, the
//     filename-required contract here also needs to be revisited.

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/middleware"
)

// TestTrackUpload_ForwardsFileName asserts the FileName field in
// trackUploadInput reaches the recorder unchanged. The previous
// bug hard-coded "" at the call site so the recorder always
// received an empty filename and the media entity creation
// failed with ErrMissingFileName.
func TestTrackUpload_ForwardsFileName(t *testing.T) {
	h, rec, _ := withFakeRecorder(t)

	in := trackUploadInput{
		UploaderID: uuid.New(),
		FileURL:    "http://localhost:9000/bucket/profiles/abc/photo/xyz.jpg",
		FileName:   "xyz.jpg",
		FileType:   "image/jpeg",
		FileSize:   2048,
		MediaCtx:   mediadomain.ContextProfilePhoto,
	}
	h.trackUpload(context.Background(), in)
	require.NoError(t, h.Stop(context.Background()))

	rec.mu.Lock()
	defer rec.mu.Unlock()
	require.Len(t, rec.calls, 1)
	assert.Equal(t, "xyz.jpg", rec.calls[0].FileName,
		"recorder must receive the FileName supplied via trackUploadInput")
}

// TestNewMedia_RejectsEmptyFileName_DocsTheRegression freezes the
// domain contract: NewMedia rejects empty FileName. This test
// fails on purpose if the validation is ever removed — at that
// point the trackUpload empty-string short-circuit could be
// re-introduced safely AND this test (plus the comment above)
// must be updated.
func TestNewMedia_RejectsEmptyFileName_DocsTheRegression(t *testing.T) {
	_, err := mediadomain.NewMedia(mediadomain.NewMediaInput{
		UploaderID: uuid.New(),
		FileURL:    "http://localhost:9000/bucket/profiles/abc/photo/xyz.jpg",
		FileName:   "",
		FileType:   "image/jpeg",
		FileSize:   1024,
		Context:    mediadomain.ContextProfilePhoto,
	})
	require.ErrorIs(t, err, mediadomain.ErrMissingFileName,
		"domain.NewMedia MUST reject empty FileName — empty values short-circuit "+
			"the entire Rekognition pipeline silently when forwarded by trackUpload")
}

// TestUploadPhoto_RecordsMediaWithNonEmptyFileName drives the
// HTTP handler end-to-end with a small JPEG buffer and asserts
// the recorder is invoked with a FileName derived from the
// storage key (not the empty string).
//
// Uses the existing mockStorageService + mockProfileRepo from
// mocks_test.go to stay aligned with the rest of the upload
// handler suite — adding a brand-new stub set would diverge the
// test fixtures and make future refactors riskier.
func TestUploadPhoto_RecordsMediaWithNonEmptyFileName(t *testing.T) {
	uploadedKey := ""
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, key string, _ io.Reader, _ string, _ int64) (string, error) {
			uploadedKey = key
			return "http://localhost:9000/bucket/" + key, nil
		},
	}
	profiles := &mockProfileRepo{
		getByOrgIDFn: func(_ context.Context, orgID uuid.UUID) (*profile.Profile, error) {
			// Return a minimal profile so the handler's GetByOrganizationID
			// + Update flow succeeds. Update writes the URL back; we don't
			// assert on it here — the moderation contract is what matters.
			return &profile.Profile{OrganizationID: orgID}, nil
		},
	}

	rec := newFakeRecorder()
	h := NewUploadHandler(storage, profiles, nil)
	h.recorder = rec

	req := buildMultipartRequest(
		http.MethodPost,
		"/api/v1/upload/photo",
		"file",
		"upload-from-client.jpg",
		"image/jpeg",
		validJPEG(),
	)

	userID := uuid.New()
	orgID := uuid.New()
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, orgID)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	h.UploadPhoto(w, req)
	require.Equal(t, http.StatusOK, w.Code, "body=%s", w.Body.String())
	require.NotEmpty(t, uploadedKey, "storage.Upload must have been invoked")

	// Wait for the trackUpload goroutine to fire.
	select {
	case <-rec.done:
	case <-time.After(2 * time.Second):
		t.Fatal("recorder never fired — trackUpload did not invoke RecordUpload")
	}

	require.NoError(t, h.Stop(context.Background()))

	rec.mu.Lock()
	defer rec.mu.Unlock()
	require.Len(t, rec.calls, 1, "RecordUpload must be called exactly once")
	got := rec.calls[0].FileName
	require.NotEmpty(t, got,
		"FileName MUST be non-empty — empty value silently bypasses moderation")

	// The handler builds the storage key as
	// `profiles/<orgID>/photo/<uuid>.jpg` and trackUpload uses
	// path.Base on that key. Assert the recorder sees the basename
	// of the bucket path (i.e. the random UUID + extension).
	assert.Equal(t, path.Base(uploadedKey), got,
		"FileName must be the basename of the storage key, not the client-supplied filename")
	assert.Equal(t, ".jpg", path.Ext(got),
		"FileName must carry the magic-bytes-derived extension")
}
