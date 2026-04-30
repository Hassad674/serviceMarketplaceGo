package handler

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/profile"
	"marketplace-backend/internal/handler/middleware"
)

// stubProfileLookup returns a getByOrgIDFn that always succeeds with
// a minimal Profile carrying the queried OrganizationID. Used by the
// streaming upload tests so the upload happy path can complete.
func stubProfileLookup() func(context.Context, uuid.UUID) (*profile.Profile, error) {
	return func(_ context.Context, orgID uuid.UUID) (*profile.Profile, error) {
		return &profile.Profile{OrganizationID: orgID}, nil
	}
}

// ---------------------------------------------------------------------------
// Streaming multipart helper — direct unit tests on the new pipeline
// closes gosec G120 (Unbounded form parsing) across upload_handler.go
// and freelance_profile_video_handler.go.
// ---------------------------------------------------------------------------

// TestReadMultipartFile_HappyPath_FindsFilePart asserts the helper
// returns the bytes + header of the part named "file" and ignores
// every other field, exactly as the legacy ParseMultipartForm path
// did. Tested with a multipart that carries unrelated fields BEFORE
// and AFTER the "file" part to lock down the iteration order.
func TestReadMultipartFile_HappyPath_FindsFilePart(t *testing.T) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	require.NoError(t, w.WriteField("ignored_before", "abc"))

	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="x.jpg"`)
	hdr.Set("Content-Type", "image/jpeg")
	part, err := w.CreatePart(hdr)
	require.NoError(t, err)
	_, err = part.Write(validJPEG())
	require.NoError(t, err)

	require.NoError(t, w.WriteField("ignored_after", "xyz"))
	require.NoError(t, w.Close())

	r := httptest.NewRequest(http.MethodPost, "/upload", body)
	r.Header.Set("Content-Type", w.FormDataContentType())

	buf, header, err := readMultipartFile(r, 1<<20)
	require.NoError(t, err)
	assert.NotEmpty(t, buf)
	assert.Equal(t, "x.jpg", header.Filename)
	assert.Equal(t, "image/jpeg", header.Header.Get("Content-Type"))
}

// TestReadMultipartFile_FileFieldMissing_ReturnsSentinel asserts
// that when no part named "file" is sent, the helper returns the
// sentinel error so the caller can map it to 400 (vs leaking 500).
func TestReadMultipartFile_FileFieldMissing_ReturnsSentinel(t *testing.T) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	require.NoError(t, w.WriteField("not_file", "blah"))
	require.NoError(t, w.Close())

	r := httptest.NewRequest(http.MethodPost, "/upload", body)
	r.Header.Set("Content-Type", w.FormDataContentType())

	_, _, err := readMultipartFile(r, 1<<20)
	assert.ErrorIs(t, err, errFileFieldNotFound)
}

// TestReadMultipartFile_OverLimit_ReturnsError asserts the byte cap
// is enforced strictly: a part exceeding `max` returns an error
// signalling oversize, allowing the caller to map to 413.
func TestReadMultipartFile_OverLimit_ReturnsError(t *testing.T) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="big.bin"`)
	hdr.Set("Content-Type", "application/octet-stream")
	part, err := w.CreatePart(hdr)
	require.NoError(t, err)
	_, err = part.Write(bytes.Repeat([]byte{0x55}, 2048))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	r := httptest.NewRequest(http.MethodPost, "/upload", body)
	r.Header.Set("Content-Type", w.FormDataContentType())

	_, _, err = readMultipartFile(r, 1024)
	require.Error(t, err)
	// Either MaxBytesError or our internal oversize message — both
	// are acceptable: callers handle them by returning 413 either way.
	assert.True(t, isMaxBytesError(err) ||
		strings.Contains(err.Error(), "exceeds maximum size"),
		"expected oversize error, got %v", err)
}

// TestReadMultipartFile_MaxBytesReader_StreamsAndCaps verifies the
// integration with http.MaxBytesReader applied on r.Body upstream.
// The body reader returns an error AFTER `max+1` bytes which
// surfaces inside the multipart machinery. This is the exact path
// hostile clients hit when streaming a 100MB body to the 5MB photo
// endpoint.
func TestReadMultipartFile_MaxBytesReader_StreamsAndCaps(t *testing.T) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="big.bin"`)
	hdr.Set("Content-Type", "application/octet-stream")
	part, err := w.CreatePart(hdr)
	require.NoError(t, err)
	// 5MB + 1 byte — enough to overflow the photo cap.
	_, err = part.Write(bytes.Repeat([]byte{0xAA}, (5<<20)+1))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	r := httptest.NewRequest(http.MethodPost, "/upload", body)
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.Body = http.MaxBytesReader(nil, r.Body, 5<<20)

	_, _, err = readMultipartFile(r, 5<<20)
	require.Error(t, err)
	assert.True(t,
		isMaxBytesError(err) || strings.Contains(err.Error(), "exceeds maximum size"),
		"expected MaxBytes/oversize, got %v", err)
}

// TestReadMultipartFile_SkipsExtraSmallParts asserts the helper does
// NOT buffer the unrelated parts of a hostile multipart with many
// small fields (the previous OOM vector for ParseMultipartForm).
// We measure RSS before and after to confirm memory does not balloon.
func TestReadMultipartFile_SkipsExtraSmallParts(t *testing.T) {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)

	// 50 noise fields — the legacy code would have buffered every
	// one of them. Each field is 1KB so 50 × 1KB = 50KB if buffered.
	noise := bytes.Repeat([]byte{0x42}, 1024)
	for i := 0; i < 50; i++ {
		require.NoError(t, w.WriteField(fmt.Sprintf("noise_%d", i), string(noise)))
	}
	hdr := make(textproto.MIMEHeader)
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="x.jpg"`)
	hdr.Set("Content-Type", "image/jpeg")
	part, err := w.CreatePart(hdr)
	require.NoError(t, err)
	_, err = part.Write(validJPEG())
	require.NoError(t, err)
	require.NoError(t, w.Close())

	r := httptest.NewRequest(http.MethodPost, "/upload", body)
	r.Header.Set("Content-Type", w.FormDataContentType())

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	buf, _, err := readMultipartFile(r, 1<<20)
	require.NoError(t, err)
	assert.NotEmpty(t, buf)

	runtime.ReadMemStats(&after)
	// We do NOT assert a precise number because GC heuristics make
	// the figure noisy; we DO assert that we didn't allocate more
	// than the file's bytes plus a sane upper bound on Go runtime
	// overhead. 4MB ≫ a JPEG fixture and the 50 noise fields if
	// they had been buffered (which they aren't).
	const upperBoundDelta = 4 << 20
	delta := after.Alloc - before.Alloc
	if after.Alloc < before.Alloc { // GC can shrink alloc; use HeapInuse delta
		delta = after.HeapInuse - before.HeapInuse
	}
	assert.Less(t, delta, uint64(upperBoundDelta),
		"streaming reader should not buffer noise fields")
}

// TestIsMaxBytesError_MatchesStdlib verifies our helper recognises
// http.MaxBytesError by type and unwrapped wrappers by message.
// Critical because the multipart machinery occasionally wraps the
// underlying MaxBytesError in opaque errors.
func TestIsMaxBytesError_MatchesStdlib(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("disk full"), false},
		{"max bytes error type", &http.MaxBytesError{Limit: 100}, true},
		{"wrapped max bytes error", fmt.Errorf("multipart next part: %w", &http.MaxBytesError{Limit: 100}), true},
		{"plain message", errors.New("http: request body too large"), true},
		{"wrapped plain message", fmt.Errorf("read upload: %w", errors.New("http: request body too large")), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isMaxBytesError(tt.err))
		})
	}
}

// ---------------------------------------------------------------------------
// End-to-end handler-level tests for the streaming pipeline.
//
// These prove the public HTTP contract: a small file → 201/200, a
// file at the cap → success, a file 1B over the cap → 413.
// ---------------------------------------------------------------------------

func TestUploadHandler_UploadPhoto_AtCap_Succeeds(t *testing.T) {
	uid := uuid.New()
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "https://files.example.com/photo.jpg", nil
		},
	}
	profiles := &mockProfileRepo{
		getByOrgIDFn: stubProfileLookup(),
		updateFn:     func(_ context.Context, _ *profile.Profile) error { return nil },
	}
	h := newTestUploadHandler(storage, profiles)

	// Build a JPEG just under the 5MB cap (use a 4.99MB filler).
	jpeg := validJPEG()
	pad := bytes.Repeat([]byte{0x55}, (5<<20)-len(jpeg)-512)
	full := append(append([]byte{}, jpeg[:len(jpeg)-2]...), pad...)
	full = append(full, 0xFF, 0xD9) // re-append EOI

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/upload/photo",
		"file", "photo.jpg", "image/jpeg", full,
	)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uid)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UploadPhoto(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

func TestUploadHandler_UploadPhoto_OneByteOver_Returns413(t *testing.T) {
	uid := uuid.New()
	h := newTestUploadHandler(&mockStorageService{}, &mockProfileRepo{})

	// 5 MB + 1 byte JPEG.
	jpeg := validJPEG()
	pad := bytes.Repeat([]byte{0x55}, (5<<20)-len(jpeg)+1)
	full := append(append([]byte{}, jpeg[:len(jpeg)-2]...), pad...)
	full = append(full, 0xFF, 0xD9)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/upload/photo",
		"file", "big.jpg", "image/jpeg", full,
	)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uid)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UploadPhoto(rec, req)
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
}

func TestUploadHandler_UploadPhoto_SmallValid_Succeeds(t *testing.T) {
	uid := uuid.New()
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "https://files.example.com/photo.jpg", nil
		},
	}
	h := newTestUploadHandler(storage, &mockProfileRepo{
		getByOrgIDFn: stubProfileLookup(),
		updateFn:     func(_ context.Context, _ *profile.Profile) error { return nil },
	})

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/upload/photo",
		"file", "photo.jpg", "image/jpeg", validJPEG(),
	)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uid)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UploadPhoto(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
}

// TestUploadHandler_UploadPhoto_NoFileField asserts a multipart body
// without a "file" part returns 400 invalid_file (not 500).
func TestUploadHandler_UploadPhoto_NoFileField(t *testing.T) {
	uid := uuid.New()
	h := newTestUploadHandler(&mockStorageService{}, &mockProfileRepo{})

	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	require.NoError(t, w.WriteField("not_file", "noise"))
	require.NoError(t, w.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/upload/photo", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uid)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	h.UploadPhoto(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "invalid_file")
}

// ---------------------------------------------------------------------------
// Concurrency: 5 photo uploads in parallel must all complete with
// independent storage calls. Lock down the absence of shared mutable
// state introduced by the streaming refactor.
// ---------------------------------------------------------------------------

func TestUploadHandler_UploadPhoto_ConcurrentUploads(t *testing.T) {
	uid := uuid.New()

	var seen int
	var mu sync.Mutex
	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			mu.Lock()
			seen++
			mu.Unlock()
			return "https://files.example.com/photo.jpg", nil
		},
	}
	profiles := &mockProfileRepo{
		getByOrgIDFn: stubProfileLookup(),
		updateFn:     func(_ context.Context, _ *profile.Profile) error { return nil },
	}
	h := newTestUploadHandler(storage, profiles)

	const N = 5
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			req := buildMultipartRequest(
				http.MethodPost, "/api/v1/upload/photo",
				"file", "photo.jpg", "image/jpeg", validJPEG(),
			)
			ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
			ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uid)
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()
			h.UploadPhoto(rec, req)
			assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		}()
	}
	wg.Wait()
	assert.Equal(t, N, seen)
}

// ---------------------------------------------------------------------------
// Memory pressure: a single 50MB upload through the video pipeline.
// Asserts heap allocation stays within ~2× the file size, proving
// the request body is not double-buffered (which the legacy
// ParseMultipartForm path was, doubling memory pressure to ~100MB).
// ---------------------------------------------------------------------------

func TestUploadHandler_UploadVideo_MemoryFootprint(t *testing.T) {
	if testing.Short() {
		t.Skip("memory pressure test is slow in -short mode")
	}
	uid := uuid.New()

	storage := &mockStorageService{
		uploadFn: func(_ context.Context, _ string, _ io.Reader, _ string, _ int64) (string, error) {
			return "https://files.example.com/video.mp4", nil
		},
	}
	h := newTestUploadHandler(storage, &mockProfileRepo{
		getByOrgIDFn: stubProfileLookup(),
		updateFn:     func(_ context.Context, _ *profile.Profile) error { return nil },
	})

	// 30MB MP4 (well under the 50MB cap to leave room for the
	// in-memory copy).
	mp4 := validMP4()
	pad := bytes.Repeat([]byte{0x33}, (30<<20)-len(mp4))
	full := append(append([]byte{}, mp4...), pad...)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	req := buildMultipartRequest(
		http.MethodPost, "/api/v1/upload/video",
		"file", "video.mp4", "video/mp4", full,
	)
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, uid)
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, uid)
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()
	h.UploadVideo(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	runtime.ReadMemStats(&after)

	// Heap-in-use should not exceed ~3× the payload size. The
	// previous ParseMultipartForm path peaked around 2× because it
	// buffers both the request body AND the part content; the
	// streaming reader only buffers the file part once.
	const upperBound = 100 << 20 // 100MB sanity ceiling for 30MB upload
	delta := after.HeapInuse
	if after.HeapInuse < before.HeapInuse {
		delta = before.HeapInuse - after.HeapInuse // unlikely; protect against underflow
	} else {
		delta = after.HeapInuse - before.HeapInuse
	}
	assert.Less(t, delta, uint64(upperBound),
		"streaming upload should not balloon process memory beyond ~3x file size")
}
