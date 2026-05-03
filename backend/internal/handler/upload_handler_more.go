package handler

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/handler/middleware"
	res "marketplace-backend/pkg/response"
)

// isMaxBytesError returns true when the error chain contains a
// http.MaxBytesError or matches the string "http: request body too
// large". The standard library variant on Go 1.21+ exposes the
// MaxBytesError type, but the multipart reader sometimes wraps that
// error so we fall back to a string match.
func isMaxBytesError(err error) bool {
	if err == nil {
		return false
	}
	var maxErr *http.MaxBytesError
	if errors.As(err, &maxErr) {
		return true
	}
	return errOrCauseContains(err, "http: request body too large")
}

// errOrCauseContains walks the error chain checking each message for
// the substring. Used as a fallback for environments where the
// MaxBytesError type is wrapped opaquely.
func errOrCauseContains(err error, substr string) bool {
	for err != nil {
		if msg := err.Error(); len(msg) >= len(substr) {
			for i := 0; i+len(substr) <= len(msg); i++ {
				if msg[i:i+len(substr)] == substr {
					return true
				}
			}
		}
		err = errors.Unwrap(err)
	}
	return false
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

	h.trackUpload(r.Context(), trackUploadInput{
		UploaderID: userID,
		FileURL:    url,
		FileType:   result.mimeType,
		FileSize:   int64(len(result.buf)),
		MediaCtx:   mediadomain.ContextProfilePhoto,
	})

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

	h.trackUpload(r.Context(), trackUploadInput{
		UploaderID: userID,
		FileURL:    url,
		FileType:   result.mimeType,
		FileSize:   int64(len(result.buf)),
		MediaCtx:   mediadomain.ContextProfileVideo,
	})

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

	h.trackUpload(r.Context(), trackUploadInput{
		UploaderID: userID,
		FileURL:    url,
		FileType:   result.mimeType,
		FileSize:   int64(len(result.buf)),
		MediaCtx:   mediadomain.ContextReferrerVideo,
	})

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

	h.trackUpload(r.Context(), trackUploadInput{
		UploaderID: userID,
		FileURL:    url,
		FileType:   result.mimeType,
		FileSize:   int64(len(result.buf)),
		MediaCtx:   mediadomain.ContextReviewVideo,
	})

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

	h.trackUpload(r.Context(), trackUploadInput{
		UploaderID: userID,
		FileURL:    url,
		FileType:   result.mimeType,
		FileSize:   int64(len(result.buf)),
		MediaCtx:   mediadomain.ContextPortfolioImage,
	})

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

	h.trackUpload(r.Context(), trackUploadInput{
		UploaderID: userID,
		FileURL:    url,
		FileType:   result.mimeType,
		FileSize:   int64(len(result.buf)),
		MediaCtx:   mediadomain.ContextPortfolioVideo,
	})

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}
