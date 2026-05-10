package handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	mediaapp "marketplace-backend/internal/app/media"
	mediadomain "marketplace-backend/internal/domain/media"
	"marketplace-backend/internal/domain/referrerprofile"
	"marketplace-backend/internal/port/repository"
	portservice "marketplace-backend/internal/port/service"
	res "marketplace-backend/pkg/response"
)

// ReferrerProfileVideoHandler owns the per-persona video endpoints
// for the referrer persona of a provider_personal org. Mirrors the
// freelance handler shape one-for-one and writes to the dedicated
// referrer_profiles row via UpdateVideo.
type ReferrerProfileVideoHandler struct {
	storage       portservice.StorageService
	profiles      repository.ReferrerProfileRepository
	mediaSvc      *mediaapp.Service
	recorder      mediaRecorder
	publicURLBase string
}

// NewReferrerProfileVideoHandler wires the handler. mediaSvc is
// optional (pass nil when moderation is not wired).
func NewReferrerProfileVideoHandler(
	storage portservice.StorageService,
	profiles repository.ReferrerProfileRepository,
	mediaSvc *mediaapp.Service,
) *ReferrerProfileVideoHandler {
	base := ""
	if storage != nil {
		base = storage.GetPublicURL("")
	}
	h := &ReferrerProfileVideoHandler{
		storage:       storage,
		profiles:      profiles,
		mediaSvc:      mediaSvc,
		publicURLBase: base,
	}
	if mediaSvc != nil {
		h.recorder = mediaSvc
	}
	return h
}

// withRecorder lets unit tests inject a fake mediaRecorder so the
// goroutine context-propagation behaviour can be asserted without
// instantiating a real *mediaapp.Service. Production callers use
// NewReferrerProfileVideoHandler which wires recorder = mediaSvc
// transparently. Test-only by convention — kept unexported.
func (h *ReferrerProfileVideoHandler) withRecorder(rec mediaRecorder) *ReferrerProfileVideoHandler {
	h.recorder = rec
	return h
}

// Upload handles POST /api/v1/referrer-profile/video.
func (h *ReferrerProfileVideoHandler) Upload(w http.ResponseWriter, r *http.Request) {
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
	key := buildPersonaVideoKey(orgID, header.Filename, "referrer_video")
	url, err := h.storage.Upload(ctx, key, file, contentType, header.Size)
	if err != nil {
		slog.Error("referrer video upload failed", "error", err, "user_id", userID)
		res.Error(w, http.StatusInternalServerError, "upload_failed", "failed to upload video")
		return
	}

	h.deletePreviousObject(ctx, orgID, userID)

	if err := h.profiles.UpdateVideo(ctx, orgID, url); err != nil {
		slog.Error("referrer profile update video failed", "error", err, "user_id", userID)
		if errors.Is(err, referrerprofile.ErrProfileNotFound) {
			res.Error(w, http.StatusNotFound, "referrer_profile_not_found", "referrer profile not found")
			return
		}
		res.Error(w, http.StatusInternalServerError, "update_failed", "failed to update profile")
		return
	}

	if h.recorder != nil {
		// Detach from the request lifetime so the moderation pipeline
		// (Rekognition video + S3 download) survives the response,
		// while still inheriting trace/baggage values for log
		// correlation. Mirrors the BUG-17 pattern in upload_handler.go
		// and closes CodeQL #64 (go/goroutine-with-background-context)
		// — the previous `context.Background()` discarded request_id
		// and user values that downstream loggers rely on.
		bgCtx := context.WithoutCancel(r.Context())
		go h.recorder.RecordUpload( // #nosec G118 -- detached after request lifetime; RecordUpload applies its own 60s timeout
			bgCtx,
			userID, url, header.Filename, contentType, header.Size,
			mediadomain.ContextReferrerVideo,
		)
	}
	res.JSON(w, http.StatusOK, map[string]string{"video_url": url})
}

// Delete handles DELETE /api/v1/referrer-profile/video.
func (h *ReferrerProfileVideoHandler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, orgID, ok := readVideoAuthContext(w, r)
	if !ok {
		return
	}

	h.deletePreviousObject(ctx, orgID, userID)

	if err := h.profiles.UpdateVideo(ctx, orgID, ""); err != nil {
		slog.Error("referrer profile delete video failed", "error", err, "user_id", userID)
		if errors.Is(err, referrerprofile.ErrProfileNotFound) {
			res.Error(w, http.StatusNotFound, "referrer_profile_not_found", "referrer profile not found")
			return
		}
		res.Error(w, http.StatusInternalServerError, "update_failed", "failed to update profile")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// deletePreviousObject fetches the current video_url from the DB
// and, when non-empty, removes the corresponding MinIO object.
func (h *ReferrerProfileVideoHandler) deletePreviousObject(ctx context.Context, orgID, userID uuid.UUID) {
	prev, err := h.profiles.GetVideoURL(ctx, orgID)
	if err != nil || prev == "" {
		return
	}
	key := extractStorageKey(prev, h.publicURLBase)
	if key == "" {
		return
	}
	if err := h.storage.Delete(ctx, key); err != nil {
		slog.Warn("referrer video: previous object delete failed",
			"error", err, "key", key, "user_id", userID)
	}
}
