package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	mediaapp "marketplace-backend/internal/app/media"
	"marketplace-backend/internal/domain/freelanceprofile"
	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
)

// FreelanceProfileVideoHandler owns the per-persona video endpoints
// for provider_personal orgs. Writes the URL directly onto the
// freelance_profiles row via repository.UpdateVideo so the split
// migration (which removed every provider_personal row from the
// legacy profiles table) cannot strand the upload.
//
// Agency orgs keep using the legacy /api/v1/upload/video handler —
// that path writes to the legacy profiles table which still has
// their rows intact. The two flows intentionally do NOT share code
// so deleting the split persona feature is one file removal.
type FreelanceProfileVideoHandler struct {
	storage       portservice.StorageService
	profiles      repository.FreelanceProfileRepository
	mediaSvc      *mediaapp.Service
	publicURLBase string
}

// NewFreelanceProfileVideoHandler wires the handler. mediaSvc is
// optional (pass nil in worktrees without moderation wired).
func NewFreelanceProfileVideoHandler(
	storage portservice.StorageService,
	profiles repository.FreelanceProfileRepository,
	mediaSvc *mediaapp.Service,
) *FreelanceProfileVideoHandler {
	base := ""
	if storage != nil {
		base = storage.GetPublicURL("")
	}
	return &FreelanceProfileVideoHandler{
		storage:       storage,
		profiles:      profiles,
		mediaSvc:      mediaSvc,
		publicURLBase: base,
	}
}

// Upload handles POST /api/v1/freelance-profile/video.
func (h *FreelanceProfileVideoHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, orgID, ok := readVideoAuthContext(w, r)
	if !ok {
		return
	}

	file, header, ok := parseVideoMultipart(w, r)
	if !ok {
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	key := buildPersonaVideoKey(orgID, header.Filename, "video")
	url, err := h.storage.Upload(ctx, key, file, contentType, header.Size)
	if err != nil {
		slog.Error("freelance video upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload video")
		return
	}

	// Best-effort delete of the previous MinIO object so we don't
	// accumulate orphaned uploads as users replace their video.
	h.deletePreviousObject(ctx, orgID, userID)

	if err := h.profiles.UpdateVideo(ctx, orgID, url); err != nil {
		slog.Error("freelance profile update video failed", "error", err, "user_id", userID)
		if errors.Is(err, freelanceprofile.ErrProfileNotFound) {
			res.Error(w, http.StatusNotFound, "freelance_profile_not_found", "freelance profile not found")
			return
		}
		res.Error(w, http.StatusInternalServerError, "update_failed", "failed to update profile")
		return
	}

	if h.mediaSvc != nil {
		go h.mediaSvc.RecordUpload(
			userID, url, header.Filename, contentType, header.Size,
			mediadomain.ContextProfileVideo,
		)
	}
	res.JSON(w, http.StatusOK, map[string]string{"video_url": url})
}

// Delete handles DELETE /api/v1/freelance-profile/video.
func (h *FreelanceProfileVideoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, orgID, ok := readVideoAuthContext(w, r)
	if !ok {
		return
	}

	h.deletePreviousObject(ctx, orgID, userID)

	if err := h.profiles.UpdateVideo(ctx, orgID, ""); err != nil {
		slog.Error("freelance profile delete video failed", "error", err, "user_id", userID)
		if errors.Is(err, freelanceprofile.ErrProfileNotFound) {
			res.Error(w, http.StatusNotFound, "freelance_profile_not_found", "freelance profile not found")
			return
		}
		res.Error(w, http.StatusInternalServerError, "update_failed", "failed to update profile")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deletePreviousObject fetches the current video_url from the DB
// and, when non-empty, removes the corresponding MinIO object.
// Failures are logged but never propagated — a stale blob is
// preferable to a user-visible upload failure.
func (h *FreelanceProfileVideoHandler) deletePreviousObject(ctx context.Context, orgID, userID uuid.UUID) {
	prev, err := h.profiles.GetVideoURL(ctx, orgID)
	if err != nil || prev == "" {
		return
	}
	key := extractStorageKey(prev, h.publicURLBase)
	if key == "" {
		return
	}
	if err := h.storage.Delete(ctx, key); err != nil {
		slog.Warn("freelance video: previous object delete failed",
			"error", err, "key", key, "user_id", userID)
	}
}

// ---------------------------------------------------------------------------
// Shared helpers (freelance + referrer)
// ---------------------------------------------------------------------------

// readVideoAuthContext pulls the user + organization IDs from the
// request context, writing a 401 response and returning ok=false
// when either is missing.
func readVideoAuthContext(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	userID, okUser := middleware.GetUserID(r.Context())
	if !okUser {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found")
		return uuid.Nil, uuid.Nil, false
	}
	orgID, okOrg := middleware.GetOrganizationID(r.Context())
	if !okOrg {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "organization not found in context")
		return uuid.Nil, uuid.Nil, false
	}
	return userID, orgID, true
}

// parseVideoMultipart enforces the 50MB size cap, parses the
// multipart form, and validates the content type. Returns the
// opened file and header on success; the caller must close the file.
func parseVideoMultipart(w http.ResponseWriter, r *http.Request) (multipart.File, *multipart.FileHeader, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxVideoSize)
	if err := r.ParseMultipartForm(maxVideoSize); err != nil {
		res.Error(w, http.StatusBadRequest, "file_too_large", "video must be under 50MB")
		return nil, nil, false
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_file", "no file provided")
		return nil, nil, false
	}
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "video/") {
		file.Close()
		res.Error(w, http.StatusBadRequest, "invalid_type", "file must be a video")
		return nil, nil, false
	}
	return file, header, true
}

// buildPersonaVideoKey returns the MinIO object key for a newly
// uploaded persona video. Keeps the freelance and referrer videos
// in distinct key namespaces (profiles/<orgID>/video_* vs
// profiles/<orgID>/referrer_video_*) so a single storage listing
// can tell them apart.
func buildPersonaVideoKey(orgID uuid.UUID, filename, prefix string) string {
	ext := filepath.Ext(filename)
	return fmt.Sprintf("profiles/%s/%s_%s%s", orgID.String(), prefix, uuid.New().String(), ext)
}

// extractStorageKey trims the public URL prefix from a stored URL
// so the resulting string can be passed to StorageService.Delete.
// Returns an empty string when the URL does not match the expected
// prefix — defensive, never try to delete a bucket object based on
// an unrelated URL.
func extractStorageKey(fullURL, publicBase string) string {
	if publicBase == "" {
		return ""
	}
	if !strings.HasPrefix(fullURL, publicBase) {
		return ""
	}
	return strings.TrimPrefix(fullURL, publicBase)
}
