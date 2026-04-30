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
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	mediaapp "marketplace-backend/internal/app/media"
	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
)

type UploadHandler struct {
	storage  portservice.StorageService
	profiles repository.ProfileRepository
	mediaSvc *mediaapp.Service
}

func NewUploadHandler(
	storage portservice.StorageService,
	profiles repository.ProfileRepository,
	mediaSvc *mediaapp.Service,
) *UploadHandler {
	return &UploadHandler{storage: storage, profiles: profiles, mediaSvc: mediaSvc}
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

// uploadResult bundles the validated buffer + computed storage key for
// reuse across the per-endpoint handlers. Keeping the helper signature
// flat (no struct in / out) keeps the call sites readable.
type uploadResult struct {
	buf []byte
	key string
	mimeType string
}

// validateAndBuildKey is the single choke-point all upload handlers run
// through. It:
//
//  1. Reads the multipart file fully into memory (bounded by max).
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
	file, header, err := r.FormFile("file")
	if err != nil {
		return nil, http.StatusBadRequest, "invalid_file", "no file provided"
	}
	defer file.Close()

	buf, err := readAllBounded(file, maxSize)
	if err != nil {
		return nil, http.StatusBadRequest, "read_failed", err.Error()
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

// contentTypeCategoriesMatch returns true when the client-declared
// Content-Type is in the same category as the magic-bytes-detected type
// ("image/" vs "image/", "video/" vs "video/", "application/pdf" vs
// "application/pdf"). The category check is permissive enough to allow
// `application/octet-stream` clients (which have no useful Content-Type)
// while still catching the SEC-09 attack where `image/png` is claimed
// for a `text/html` payload.
func contentTypeCategoriesMatch(declared, detected string) bool {
	if declared == "application/octet-stream" {
		return true
	}
	declaredPrefix := categoryPrefix(declared)
	detectedPrefix := categoryPrefix(detected)
	return declaredPrefix == detectedPrefix
}

func categoryPrefix(ct string) string {
	for i, c := range ct {
		if c == '/' {
			return ct[:i]
		}
	}
	return ct
}

func (h *UploadHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPhotoSize)
	if err := r.ParseMultipartForm(maxPhotoSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "photo must be under 5MB")
		return
	}

	prefix := fmt.Sprintf("profiles/%s/photo", orgID.String())
	result, status, code, msg := validateAndBuildKey(r, ScopePhoto, maxPhotoSize, prefix)
	if result == nil {
		res.Error(w, status, code, msg)
		return
	}

	url, err := h.storage.Upload(r.Context(), result.key, bytes.NewReader(result.buf), result.mimeType, int64(len(result.buf)))
	if err != nil {
		slog.Error("photo upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload photo")
		return
	}

	profile, err := h.profiles.GetByOrganizationID(r.Context(), orgID)
	if err != nil {
		slog.Error("get profile failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "profile_error", "failed to get profile")
		return
	}

	profile.PhotoURL = url
	if err := h.profiles.Update(r.Context(), profile); err != nil {
		slog.Error("update profile photo failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "update_failed", "failed to update profile")
		return
	}

	if h.mediaSvc != nil {
		go h.mediaSvc.RecordUpload(userID, url, "", result.mimeType, int64(len(result.buf)), mediadomain.ContextProfilePhoto)
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *UploadHandler) UploadVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxVideoSize)
	if err := r.ParseMultipartForm(maxVideoSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "video must be under 50MB")
		return
	}

	prefix := fmt.Sprintf("profiles/%s/video", orgID.String())
	result, status, code, msg := validateAndBuildKey(r, ScopeVideo, maxVideoSize, prefix)
	if result == nil {
		res.Error(w, status, code, msg)
		return
	}

	url, err := h.storage.Upload(r.Context(), result.key, bytes.NewReader(result.buf), result.mimeType, int64(len(result.buf)))
	if err != nil {
		slog.Error("video upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload video")
		return
	}

	profile, err := h.profiles.GetByOrganizationID(r.Context(), orgID)
	if err != nil {
		slog.Error("get profile failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "profile_error", "failed to get profile")
		return
	}

	profile.PresentationVideoURL = url
	if err := h.profiles.Update(r.Context(), profile); err != nil {
		slog.Error("update profile video failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "update_failed", "failed to update profile")
		return
	}

	if h.mediaSvc != nil {
		go h.mediaSvc.RecordUpload(userID, url, "", result.mimeType, int64(len(result.buf)), mediadomain.ContextProfileVideo)
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *UploadHandler) UploadReferrerVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxVideoSize)
	if err := r.ParseMultipartForm(maxVideoSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "video must be under 50MB")
		return
	}

	prefix := fmt.Sprintf("profiles/%s/referrer_video", orgID.String())
	result, status, code, msg := validateAndBuildKey(r, ScopeVideo, maxVideoSize, prefix)
	if result == nil {
		res.Error(w, status, code, msg)
		return
	}

	url, err := h.storage.Upload(r.Context(), result.key, bytes.NewReader(result.buf), result.mimeType, int64(len(result.buf)))
	if err != nil {
		slog.Error("referrer video upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload video")
		return
	}

	profile, err := h.profiles.GetByOrganizationID(r.Context(), orgID)
	if err != nil {
		slog.Error("get profile failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "profile_error", "failed to get profile")
		return
	}

	profile.ReferrerVideoURL = url
	if err := h.profiles.Update(r.Context(), profile); err != nil {
		slog.Error("update profile referrer video failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "update_failed", "failed to update profile")
		return
	}

	if h.mediaSvc != nil {
		go h.mediaSvc.RecordUpload(userID, url, "", result.mimeType, int64(len(result.buf)), mediadomain.ContextReferrerVideo)
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}

const maxReviewVideoSize = 100 << 20 // 100 MB

func (h *UploadHandler) UploadReviewVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxReviewVideoSize)
	if err := r.ParseMultipartForm(maxReviewVideoSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "video must be under 100MB")
		return
	}

	prefix := fmt.Sprintf("reviews/%s/video", userID.String())
	result, status, code, msg := validateAndBuildKey(r, ScopeVideo, maxReviewVideoSize, prefix)
	if result == nil {
		res.Error(w, status, code, msg)
		return
	}

	url, err := h.storage.Upload(r.Context(), result.key, bytes.NewReader(result.buf), result.mimeType, int64(len(result.buf)))
	if err != nil {
		slog.Error("review video upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload video")
		return
	}

	if h.mediaSvc != nil {
		go h.mediaSvc.RecordUpload(userID, url, "", result.mimeType, int64(len(result.buf)), mediadomain.ContextReviewVideo)
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *UploadHandler) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	profile, err := h.profiles.GetByOrganizationID(r.Context(), orgID)
	if err != nil {
		slog.Error("get profile failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "profile_error", "failed to get profile")
		return
	}

	profile.PresentationVideoURL = ""
	if err := h.profiles.Update(r.Context(), profile); err != nil {
		slog.Error("delete profile video failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "update_failed", "failed to update profile")
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"message": "video removed"})
}

func (h *UploadHandler) DeleteReferrerVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}
	orgID, ok := middleware.GetOrganizationID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return
	}

	profile, err := h.profiles.GetByOrganizationID(r.Context(), orgID)
	if err != nil {
		slog.Error("get profile failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "profile_error", "failed to get profile")
		return
	}

	profile.ReferrerVideoURL = ""
	if err := h.profiles.Update(r.Context(), profile); err != nil {
		slog.Error("delete profile referrer video failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "update_failed", "failed to update profile")
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"message": "referrer video removed"})
}

const maxPortfolioImageSize = 10 << 20  // 10 MB
const maxPortfolioVideoSize = 100 << 20 // 100 MB

// UploadPortfolioImage handles POST /api/v1/upload/portfolio-image.
//
// Magic-byte completeness check (SOI/EOI for JPEG, PNG signature/IEND
// chunk) is preserved on top of the centralised allowlist so truncated
// JPEG/PNGs uploaded by buggy clients still surface a clear error
// instead of a Skia decode failure on the frontend.
func (h *UploadHandler) UploadPortfolioImage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPortfolioImageSize)
	if err := r.ParseMultipartForm(maxPortfolioImageSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "image must be under 10MB")
		return
	}

	prefix := fmt.Sprintf("portfolios/%s/image", userID.String())
	result, status, code, msg := validateAndBuildKey(r, ScopePhoto, maxPortfolioImageSize, prefix)
	if result == nil {
		res.Error(w, status, code, msg)
		return
	}
	if err := validateImageBytes(result.buf, result.mimeType); err != nil {
		res.Error(w, http.StatusBadRequest, "corrupt_image", err.Error())
		return
	}

	url, err := h.storage.Upload(r.Context(), result.key, bytes.NewReader(result.buf), result.mimeType, int64(len(result.buf)))
	if err != nil {
		slog.Error("portfolio image upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload image")
		return
	}

	if h.mediaSvc != nil {
		go h.mediaSvc.RecordUpload(userID, url, "", result.mimeType, int64(len(result.buf)), mediadomain.ContextPortfolioImage)
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}

// validateImageBytes verifies that the byte buffer looks like a complete
// image of the declared type. The check is intentionally lenient: it confirms
// the file header and looks for the end-of-file marker anywhere in the tail
// (real-world JPEGs/PNGs often have trailing metadata or padding bytes after
// the spec-defined end marker, which is perfectly valid in practice).
func validateImageBytes(buf []byte, contentType string) error {
	if len(buf) < 16 {
		return fmt.Errorf("image is too small")
	}

	// Look at the last 1KB for the end marker — enough to catch truncation
	// without false-positive on legit files with trailing bytes.
	const tailWindow = 1024
	tail := buf
	if len(tail) > tailWindow {
		tail = tail[len(tail)-tailWindow:]
	}

	switch contentType {
	case "image/jpeg":
		// SOI (Start Of Image) must be at the very start.
		if buf[0] != 0xFF || buf[1] != 0xD8 {
			return fmt.Errorf("not a valid JPEG (missing SOI marker)")
		}
		// EOI (End Of Image) must appear somewhere in the tail.
		if !bytes.Contains(tail, []byte{0xFF, 0xD9}) {
			return fmt.Errorf("JPEG file is incomplete (no EOI marker in tail)")
		}
	case "image/png":
		// PNG signature: 89 50 4E 47 0D 0A 1A 0A
		if buf[0] != 0x89 || buf[1] != 0x50 || buf[2] != 0x4E || buf[3] != 0x47 {
			return fmt.Errorf("not a valid PNG (missing signature)")
		}
		// IEND chunk: 49 45 4E 44 AE 42 60 82 — must appear in the tail.
		if !bytes.Contains(tail, []byte{0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82}) {
			return fmt.Errorf("PNG file is incomplete (no IEND chunk in tail)")
		}
	}
	// For other formats (webp, etc.), accept as-is — the magic-bytes
	// detector already vouched for the file type at this point.
	return nil
}

// UploadPortfolioVideo handles POST /api/v1/upload/portfolio-video.
func (h *UploadHandler) UploadPortfolioVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPortfolioVideoSize)
	if err := r.ParseMultipartForm(maxPortfolioVideoSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "video must be under 100MB")
		return
	}

	prefix := fmt.Sprintf("portfolios/%s/video", userID.String())
	result, status, code, msg := validateAndBuildKey(r, ScopeVideo, maxPortfolioVideoSize, prefix)
	if result == nil {
		res.Error(w, status, code, msg)
		return
	}

	url, err := h.storage.Upload(r.Context(), result.key, bytes.NewReader(result.buf), result.mimeType, int64(len(result.buf)))
	if err != nil {
		slog.Error("portfolio video upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload video")
		return
	}

	if h.mediaSvc != nil {
		go h.mediaSvc.RecordUpload(userID, url, "", result.mimeType, int64(len(result.buf)), mediadomain.ContextPortfolioVideo)
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}
