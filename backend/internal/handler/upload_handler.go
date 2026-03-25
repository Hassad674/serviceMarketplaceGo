package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
)

type UploadHandler struct {
	storage  portservice.StorageService
	profiles repository.ProfileRepository
}

func NewUploadHandler(
	storage portservice.StorageService,
	profiles repository.ProfileRepository,
) *UploadHandler {
	return &UploadHandler{storage: storage, profiles: profiles}
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
