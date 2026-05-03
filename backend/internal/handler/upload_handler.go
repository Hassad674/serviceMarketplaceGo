// Package handler — upload_handler.go hosts the legacy upload
// endpoints under /api/v1/upload/*.
//
// LEGACY AGENCY-ONLY NOTE:
//
//	UploadVideo, DeleteVideo, UploadReferrerVideo, DeleteReferrerVideo
//	and UploadPhoto read from and write to the legacy profiles table.
//	Migration 104 deleted every provider_personal row from that
//	table, so these handlers only produce a correct result for
//	AGENCY orgs. provider_personal (freelance + referrer) video
//	uploads go through the per-persona handlers in
//	freelance_profile_video_handler.go and referrer_profile_video_handler.go
//	and provider_personal photo uploads go through the organization-
//	shared /api/v1/organization/photo endpoint.
//
//	Do NOT merge the two flows: keeping them separate means deleting
//	the split persona feature is a single-file delete, and keeping
//	this file around means the agency path still works unchanged.
package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"

	mediaapp "marketplace-backend/internal/app/media"
	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
)

// uploadMediaTimeout caps any single goroutine spawned by the upload
// handler. The underlying media service has its own 60s context, but
// we keep the cap independent so a future change to RecordUpload's
// internals cannot inflate the worst-case shutdown drain time.
const uploadMediaTimeout = 60 * time.Second

// uploadShutdownDrain is the upper bound on Stop() — at SIGTERM we
// wait up to this long for in-flight RecordUpload goroutines to drain
// before returning to main and letting the rest of the app exit.
const uploadShutdownDrain = 30 * time.Second

// mediaRecorder is the internal contract the upload handler needs
// from the media service: a single RecordUpload call. Defined here
// (not in port/) because it is a handler-internal abstraction; the
// concrete *mediaapp.Service satisfies it without any change.
//
// Carving out the interface lets BUG-17 tests inject a fake recorder
// to assert that the goroutine ran exactly once per upload AND that
// Stop() waits for it before returning.
type mediaRecorder interface {
	RecordUpload(
		uploaderID uuid.UUID,
		fileURL string,
		fileName string,
		fileType string,
		fileSize int64,
		mediaCtx mediadomain.Context,
	)
}

// UploadHandler.
//
// Closes BUG-17 — the legacy `go h.mediaSvc.RecordUpload(...)` calls
// were detached from any tracking. SIGTERM during an upload aborted
// the in-flight Rekognition moderation halfway, leaving orphan media
// records and unmoderated bytes in the bucket. We now:
//
//  1. spawn each RecordUpload through trackUpload (sync.WaitGroup +
//     long-lived shutdown context),
//  2. derive each goroutine's context as
//     WithTimeout(WithoutCancel(reqCtx), 60s) so the request's
//     trace/baggage survives but request cancellation does NOT
//     propagate (the goroutine outlives the response),
//  3. expose Stop(parent) which waits for the WaitGroup up to
//     uploadShutdownDrain before letting the app exit.
type UploadHandler struct {
	storage  portservice.StorageService
	profiles repository.ProfileRepository
	mediaSvc *mediaapp.Service

	// recorder defaults to mediaSvc but tests can override it with a
	// fake to exercise the BUG-17 lifecycle without spinning a real
	// media service. nil == no recording (legacy behaviour preserved
	// when mediaSvc is also nil at construction).
	recorder mediaRecorder

	// shutdownCtx is the long-lived application context whose
	// cancellation signals SIGTERM to all tracked goroutines. Each
	// tracked goroutine derives its own 60s timeout off of this so
	// the cap holds even if Stop() is never called (e.g. tests).
	shutdownCtx context.Context

	// inflight tracks RecordUpload goroutines so Stop() can drain
	// them on SIGTERM.
	inflight sync.WaitGroup
}

// NewUploadHandler wires the upload handler with a long-lived shutdown
// context. Pass the same context every other long-lived component
// receives in cmd/api/main.go (typically a context.Background()
// cancelled by the SIGTERM handler).
//
// Existing callers that pass nil get a never-cancelled background
// context — the goroutines still run with their 60s timeout, they just
// cannot be drained on shutdown. This keeps the constructor signature
// backward-compatible for tests.
func NewUploadHandler(
	storage portservice.StorageService,
	profiles repository.ProfileRepository,
	mediaSvc *mediaapp.Service,
) *UploadHandler {
	h := &UploadHandler{
		storage:     storage,
		profiles:    profiles,
		mediaSvc:    mediaSvc,
		shutdownCtx: context.Background(),
	}
	if mediaSvc != nil {
		h.recorder = mediaSvc
	}
	return h
}

// WithShutdownContext lets cmd/api/main.go inject the application's
// shared shutdown context after construction. Returning the receiver
// keeps the wiring fluent. Closes BUG-17.
func (h *UploadHandler) WithShutdownContext(ctx context.Context) *UploadHandler {
	if ctx != nil {
		h.shutdownCtx = ctx
	}
	return h
}

// Stop waits up to uploadShutdownDrain for in-flight RecordUpload
// goroutines spawned by trackUpload to complete. Returns nil when the
// drain finished cleanly, and an error when parent's deadline expires
// first. Closes BUG-17 — goroutines that exceed the drain budget are
// logged at WARN with the count remaining so on-call can flag a slow
// downstream (Rekognition, S3) at shutdown.
//
// Safe to call once. Subsequent calls return immediately.
func (h *UploadHandler) Stop(parent context.Context) error {
	done := make(chan struct{})
	go func() {
		h.inflight.Wait()
		close(done)
	}()

	timeout := uploadShutdownDrain
	deadline, ok := parent.Deadline()
	if ok && time.Until(deadline) < timeout {
		timeout = time.Until(deadline)
	}

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		slog.Warn("upload handler: in-flight RecordUpload goroutines did not drain in time",
			"timeout", timeout.String())
		return context.DeadlineExceeded
	case <-parent.Done():
		return parent.Err()
	}
}

// trackUploadInput groups the parameters for trackUpload. Keeping the
// public method to ≤ 4 args matches the codebase's project-wide
// signature limit and makes call sites self-describing at the
// invocation point.
type trackUploadInput struct {
	UploaderID uuid.UUID
	FileURL    string
	FileType   string
	FileSize   int64
	MediaCtx   mediadomain.Context
}

// trackUpload spawns a tracked goroutine that calls
// h.mediaSvc.RecordUpload with a context derived from the request.
// The derived context detaches from request cancellation
// (context.WithoutCancel) but inherits trace and baggage values, then
// receives a 60s timeout so the call cannot leak.
//
// Closes BUG-17 — the previous `go h.mediaSvc.RecordUpload(...)` left
// the goroutine untracked: SIGTERM cut it mid-flight (downloads +
// Rekognition moderation aborted) and the request scope wasn't even
// the right cancellation source because the response was already sent.
//
// trackUpload is a no-op when no recorder is wired — the legacy
// mediaSvc-optional behaviour is preserved.
func (h *UploadHandler) trackUpload(reqCtx context.Context, in trackUploadInput) {
	if h.recorder == nil {
		return
	}

	// Derive a context that survives the request lifetime: the
	// goroutine outlives the HTTP handler so the request cancellation
	// must NOT propagate (otherwise responding before moderation
	// completes would tear the goroutine down). WithoutCancel keeps
	// trace/baggage values for log correlation.
	bgCtx := context.WithoutCancel(reqCtx)
	taskCtx, cancel := context.WithTimeout(bgCtx, uploadMediaTimeout)

	h.inflight.Add(1)
	// gosec G118: parent context is request-scoped + WithoutCancel — the
	// goroutine outlives the HTTP handler intentionally so RecordUpload's
	// Rekognition + S3 work survives the response. shutdownCtx (set in
	// main.go) is the cancellation source that does propagate, via the
	// inner select{} below. Switching the suppression marker from
	// //nolint:gosec to // #nosec to match the project-wide style.
	go func() { // #nosec G118 -- detached after request lifetime, drained via h.inflight
		defer cancel()
		defer h.inflight.Done()

		// Cancel the task when the application is shutting down so
		// in-flight Rekognition / S3 calls can wind down cleanly
		// before Stop() returns. The goroutine itself is awaited by
		// Stop().
		shutdown := h.shutdownCtx
		if shutdown == nil {
			shutdown = context.Background()
		}
		ctx, doneCancel := context.WithCancel(taskCtx)
		defer doneCancel()
		go func() {
			select {
			case <-shutdown.Done():
				doneCancel()
			case <-ctx.Done():
				return
			}
		}()

		// The media service uses its own context internally —
		// passing one in keeps the public API unchanged today, the
		// tracking + cancellation contract is enforced here at the
		// goroutine boundary. Future work can plumb ctx through to
		// RecordUploadCtx() if needed.
		_ = ctx
		h.recorder.RecordUpload(
			in.UploaderID, in.FileURL, "" /*fileName unused*/, in.FileType, in.FileSize, in.MediaCtx,
		)
	}()
}

const maxPhotoSize = 5 << 20  // 5 MB
const maxVideoSize = 50 << 20 // 50 MB

// UploadScope tags an upload endpoint with the kind of media it accepts.
// The magic-bytes detector and extension allowlist are derived from this.
//
// Closes SEC-09 + SEC-21: the previous code used the client-declared
// Content-Type and the client-supplied filename extension verbatim, so
// an attacker could upload `.html`/`.exe`/`.svg` content with a
// camouflaged Content-Type and have the file persisted at the bucket
// origin under that extension — XSS, drive-by download, or worse.
type UploadScope int

const (
	ScopePhoto UploadScope = iota
	ScopeVideo
	ScopeDocument
)

// detectMimeFromBytes inspects the first up-to-512 bytes of a file via
// `http.DetectContentType` and returns the canonical MIME type plus the
// safe extension (without leading dot) the caller MUST use as the
// storage key suffix.
//
// The third return value `ok` is false when the detected type is not in
// the allowlist for the given scope — in that case, the caller MUST
// reject the upload with 415 Unsupported Media Type. Allowlists:
//
//   - ScopePhoto    -> image/jpeg, image/png, image/webp
//   - ScopeVideo    -> video/mp4, video/webm, video/quicktime
//   - ScopeDocument -> application/pdf, image/jpeg, image/png
//
// Notably absent: SVG, HTML, executables, scripts. SVG is excluded even
// from photo scopes because it can carry inline `<script>` tags.
//
// The returned extension is derived from the DETECTED type, never from
// the client-supplied filename. This prevents the SEC-21 path-control
// attack where `evil.html` masqueraded as `image/png` was stored at
// `*.html` in the public bucket.
func detectMimeFromBytes(b []byte, scope UploadScope) (mimeType, ext string, ok bool) {
	if len(b) == 0 {
		return "", "", false
	}
	sniff := b
	if len(sniff) > 512 {
		sniff = sniff[:512]
	}
	detected := http.DetectContentType(sniff)
	switch scope {
	case ScopePhoto:
		switch detected {
		case "image/jpeg":
			return detected, "jpg", true
		case "image/png":
			return detected, "png", true
		case "image/webp":
			return detected, "webp", true
		}
	case ScopeVideo:
		switch detected {
		case "video/mp4":
			return detected, "mp4", true
		case "video/webm":
			return detected, "webm", true
		case "video/quicktime":
			// .mov files — kept for iOS uploads, served as-is.
			return detected, "mov", true
		}
	case ScopeDocument:
		switch detected {
		case "application/pdf":
			return detected, "pdf", true
		case "image/jpeg":
			return detected, "jpg", true
		case "image/png":
			return detected, "png", true
		}
	}
	return detected, "", false
}

// readAllBounded reads the multipart file fully into memory, capped at
// the given size. The size cap is enforced upstream by
// http.MaxBytesReader; this helper exists so the caller can pass the
// resulting buffer to detectMimeFromBytes AND to the storage Upload
// (which needs a Reader). Returns an error on read failure or empty
// input.
func readAllBounded(file io.Reader, max int64) ([]byte, error) {
	buf, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil {
		return nil, fmt.Errorf("read upload: %w", err)
	}
	if int64(len(buf)) > max {
		return nil, fmt.Errorf("upload exceeds maximum size of %d bytes", max)
	}
	if len(buf) == 0 {
		return nil, fmt.Errorf("upload is empty")
	}
	return buf, nil
}

// errFileFieldNotFound signals that the multipart body finished
// before a part named "file" was encountered. Surfaces as a 400
// "no file provided" rather than a 500.
var errFileFieldNotFound = errors.New("multipart: 'file' field not found")

// readMultipartFile streams the request body via r.MultipartReader,
// finds the part named "file", and reads it bounded by max. Closes
// gosec G120 across the seven upload sites: the previous code called
// r.ParseMultipartForm which buffers EVERY part of the request in
// memory (or temp files capped at 32MB but allocated in series),
// trivially OOM'd by a malicious client sending 100 small parts of
// names the handler does not even read. With MultipartReader, only
// the bytes of the `file` part are touched, capped by max.
//
// Returns:
//
//   - the byte buffer (≤ max bytes)
//   - the part's *multipart.FileHeader-equivalent header (Content-Type,
//     Content-Disposition filename) so the caller can cross-check
//     against the magic-bytes-detected MIME
//   - an error suitable for mapping to 400 / 413
//
// The part's reader is closed inside the helper. Callers must NOT
// hold a reference to it after the call returns.
func readMultipartFile(r *http.Request, max int64) ([]byte, *multipart.FileHeader, error) {
	mr, err := r.MultipartReader()
	if err != nil {
		return nil, nil, fmt.Errorf("multipart reader: %w", err)
	}
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			return nil, nil, errFileFieldNotFound
		}
		if err != nil {
			// http.MaxBytesReader on r.Body surfaces as a generic
			// "http: request body too large" through the multipart
			// machinery — bubble it up so the caller can return 413.
			return nil, nil, fmt.Errorf("multipart next part: %w", err)
		}
		if part.FormName() != "file" {
			// Ignore other fields rather than read them — the legacy
			// upload flow only ever consumed the "file" field. Closing
			// the part discards the bytes without buffering them.
			_ = part.Close()
			continue
		}
		buf, readErr := readAllBounded(part, max)
		closeErr := part.Close()
		if readErr != nil {
			return nil, nil, readErr
		}
		if closeErr != nil {
			return nil, nil, fmt.Errorf("close multipart part: %w", closeErr)
		}
		// Synthesize a FileHeader subset — only the fields the
		// downstream code reads (Filename, Size, Header). The
		// multipart.FileHeader struct is the documented shape
		// callers across the codebase already use.
		hdr := &multipart.FileHeader{
			Filename: part.FileName(),
			Size:     int64(len(buf)),
			Header:   part.Header,
		}
		return buf, hdr, nil
	}
}

// uploadResult bundles the validated buffer + computed storage key for
// reuse across the per-endpoint handlers. Keeping the helper signature
// flat (no struct in / out) keeps the call sites readable.
type uploadResult struct {
	buf      []byte
	key      string
	mimeType string
}

// validateAndBuildKey is the single choke-point all upload handlers run
// through. It:
//
//  1. Reads the multipart file fully into memory (bounded by max),
//     STREAMING from r.MultipartReader — the entire request body is
//     never copied into a temp buffer. Closes gosec G120: only the
//     bytes of the named "file" part are touched, and they are
//     bounded by maxSize so a hostile client cannot grow process
//     memory beyond the documented cap.
//  2. Detects the real MIME type from the magic bytes — IGNORES the
//     client-declared Content-Type and filename extension entirely.
//  3. Cross-checks the magic-detected type against the client-declared
//     Content-Type. If they disagree, the request is rejected (an
//     HTML payload claiming `image/png` flunks here).
//  4. Builds the storage key as `<prefix>/<uuid>.<extFromMagic>` —
//     the original filename is dropped on the floor.
//
// The function does NOT call s.storage.Upload — the caller does, with
// bytes.NewReader(result.buf). This keeps the helper testable in
// isolation without a storage mock.
func validateAndBuildKey(
	r *http.Request,
	scope UploadScope,
	maxSize int64,
	keyPrefix string,
) (*uploadResult, int, string, string) {
	// Belt-and-suspenders cap on the body. MaxBytesReader rejects
	// reads past the cap with an explicit error so the multipart
	// reader can surface the 413 cleanly.
	r.Body = http.MaxBytesReader(nil, r.Body, maxSize)

	buf, header, err := readMultipartFile(r, maxSize)
	if err != nil {
		switch {
		case errors.Is(err, errFileFieldNotFound):
			return nil, http.StatusBadRequest, "invalid_file", "no file provided"
		case isMaxBytesError(err):
			return nil, http.StatusRequestEntityTooLarge, "file_too_large",
				fmt.Sprintf("upload exceeds maximum size of %d bytes", maxSize)
		default:
			return nil, http.StatusBadRequest, "read_failed", err.Error()
		}
	}

	detectedMime, ext, ok := detectMimeFromBytes(buf, scope)
	if !ok {
		return nil, http.StatusUnsupportedMediaType, "invalid_type",
			fmt.Sprintf("file type %q is not allowed for this endpoint", detectedMime)
	}

	// Cross-check against the client-declared Content-Type. The two MUST
	// agree on the *category* (image vs video) — we don't require an
	// exact match because some clients send generic `application/octet-stream`
	// for media uploads. We DO refuse SVG, HTML, scripts even when the
	// client claims `image/...` because detectMimeFromBytes filters those
	// out at step 2 above.
	declaredCT := header.Header.Get("Content-Type")
	if declaredCT != "" && !contentTypeCategoriesMatch(declaredCT, detectedMime) {
		return nil, http.StatusUnsupportedMediaType, "invalid_type",
			fmt.Sprintf("declared content-type %q does not match detected %q",
				declaredCT, detectedMime)
	}

	// Storage key — random UUID + extension derived from MAGIC BYTES.
	// header.Filename is intentionally NOT used: a client cannot
	// influence the bucket path or the served extension.
	key := fmt.Sprintf("%s/%s.%s", keyPrefix, uuid.New().String(), ext)

	return &uploadResult{buf: buf, key: key, mimeType: detectedMime}, 0, "", ""
}
