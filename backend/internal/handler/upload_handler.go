package handler

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

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

func (h *UploadHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxPhotoSize)
	if err := r.ParseMultipartForm(maxPhotoSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "photo must be under 5MB")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_file", "no file provided")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		res.Error(w, http.StatusBadRequest, "invalid_type", "file must be an image")
		return
	}

	ext := filepath.Ext(header.Filename)
	key := fmt.Sprintf("profiles/%s/photo_%s%s", userID.String(), uuid.New().String(), ext)

	url, err := h.storage.Upload(r.Context(), key, file, contentType, header.Size)
	if err != nil {
		slog.Error("photo upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload photo")
		return
	}

	profile, err := h.profiles.GetByUserID(r.Context(), userID)
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
		go h.mediaSvc.RecordUpload(userID, url, header.Filename, contentType, header.Size, mediadomain.ContextProfilePhoto)
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *UploadHandler) UploadVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxVideoSize)
	if err := r.ParseMultipartForm(maxVideoSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "video must be under 50MB")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_file", "no file provided")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "video/") {
		res.Error(w, http.StatusBadRequest, "invalid_type", "file must be a video")
		return
	}

	ext := filepath.Ext(header.Filename)
	key := fmt.Sprintf("profiles/%s/video_%s%s", userID.String(), uuid.New().String(), ext)

	url, err := h.storage.Upload(r.Context(), key, file, contentType, header.Size)
	if err != nil {
		slog.Error("video upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload video")
		return
	}

	profile, err := h.profiles.GetByUserID(r.Context(), userID)
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
		go h.mediaSvc.RecordUpload(userID, url, header.Filename, contentType, header.Size, mediadomain.ContextProfileVideo)
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *UploadHandler) UploadReferrerVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxVideoSize)
	if err := r.ParseMultipartForm(maxVideoSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "video must be under 50MB")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_file", "no file provided")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "video/") {
		res.Error(w, http.StatusBadRequest, "invalid_type", "file must be a video")
		return
	}

	ext := filepath.Ext(header.Filename)
	key := fmt.Sprintf("profiles/%s/referrer_video_%s%s", userID.String(), uuid.New().String(), ext)

	url, err := h.storage.Upload(r.Context(), key, file, contentType, header.Size)
	if err != nil {
		slog.Error("referrer video upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload video")
		return
	}

	profile, err := h.profiles.GetByUserID(r.Context(), userID)
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
		go h.mediaSvc.RecordUpload(userID, url, header.Filename, contentType, header.Size, mediadomain.ContextReferrerVideo)
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

	file, header, err := r.FormFile("file")
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_file", "no file provided")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "video/") {
		res.Error(w, http.StatusBadRequest, "invalid_type", "file must be a video (mp4, webm, mov)")
		return
	}

	ext := filepath.Ext(header.Filename)
	key := fmt.Sprintf("reviews/%s/video_%s%s", userID.String(), uuid.New().String(), ext)

	url, err := h.storage.Upload(r.Context(), key, file, contentType, header.Size)
	if err != nil {
		slog.Error("review video upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload video")
		return
	}

	if h.mediaSvc != nil {
		go h.mediaSvc.RecordUpload(userID, url, header.Filename, contentType, header.Size, mediadomain.ContextReviewVideo)
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}

func (h *UploadHandler) DeleteVideo(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return
	}

	profile, err := h.profiles.GetByUserID(r.Context(), userID)
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

	profile, err := h.profiles.GetByUserID(r.Context(), userID)
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

	file, header, err := r.FormFile("file")
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_file", "no file provided")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		res.Error(w, http.StatusBadRequest, "invalid_type", "file must be an image")
		return
	}

	// Buffer the file in memory to validate completeness before uploading.
	// Truncated images would otherwise pass through and fail to decode in the
	// frontend (Skia is strict about JPEG/PNG markers).
	buf, err := io.ReadAll(file)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "read_failed", "failed to read uploaded file")
		return
	}
	if err := validateImageBytes(buf, contentType); err != nil {
		res.Error(w, http.StatusBadRequest, "corrupt_image", err.Error())
		return
	}

	ext := filepath.Ext(header.Filename)
	key := fmt.Sprintf("portfolios/%s/image_%s%s", userID.String(), uuid.New().String(), ext)

	url, err := h.storage.Upload(r.Context(), key, bytes.NewReader(buf), contentType, int64(len(buf)))
	if err != nil {
		slog.Error("portfolio image upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload image")
		return
	}

	if h.mediaSvc != nil {
		go h.mediaSvc.RecordUpload(userID, url, header.Filename, contentType, int64(len(buf)), mediadomain.ContextPortfolioImage)
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

	switch {
	case strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg"):
		// SOI (Start Of Image) must be at the very start.
		if buf[0] != 0xFF || buf[1] != 0xD8 {
			return fmt.Errorf("not a valid JPEG (missing SOI marker)")
		}
		// EOI (End Of Image) must appear somewhere in the tail.
		if !bytes.Contains(tail, []byte{0xFF, 0xD9}) {
			return fmt.Errorf("JPEG file is incomplete (no EOI marker in tail)")
		}
	case strings.Contains(contentType, "png"):
		// PNG signature: 89 50 4E 47 0D 0A 1A 0A
		if buf[0] != 0x89 || buf[1] != 0x50 || buf[2] != 0x4E || buf[3] != 0x47 {
			return fmt.Errorf("not a valid PNG (missing signature)")
		}
		// IEND chunk: 49 45 4E 44 AE 42 60 82 — must appear in the tail.
		if !bytes.Contains(tail, []byte{0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82}) {
			return fmt.Errorf("PNG file is incomplete (no IEND chunk in tail)")
		}
	}
	// For other formats (gif, webp, avif, heic, etc.), accept as-is.
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

	file, header, err := r.FormFile("file")
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_file", "no file provided")
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "video/") {
		res.Error(w, http.StatusBadRequest, "invalid_type", "file must be a video")
		return
	}

	ext := filepath.Ext(header.Filename)
	key := fmt.Sprintf("portfolios/%s/video_%s%s", userID.String(), uuid.New().String(), ext)

	url, err := h.storage.Upload(r.Context(), key, file, contentType, header.Size)
	if err != nil {
		slog.Error("portfolio video upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload video")
		return
	}

	if h.mediaSvc != nil {
		go h.mediaSvc.RecordUpload(userID, url, header.Filename, contentType, header.Size, mediadomain.ContextPortfolioVideo)
	}

	res.JSON(w, http.StatusOK, map[string]string{"url": url})
}
