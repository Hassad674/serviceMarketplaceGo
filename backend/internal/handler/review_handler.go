package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	reviewapp "marketplace-backend/internal/app/review"
	reviewdomain "marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// ReviewHandler handles HTTP requests for reviews.
type ReviewHandler struct {
	reviewSvc *reviewapp.Service
}

// NewReviewHandler creates a new ReviewHandler.
func NewReviewHandler(svc *reviewapp.Service) *ReviewHandler {
	return &ReviewHandler{reviewSvc: svc}
}

// CreateReview handles POST /api/v1/reviews.
func (h *ReviewHandler) CreateReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.CreateReviewRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	proposalID, err := uuid.Parse(req.ProposalID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "proposal_id must be a valid UUID")
		return
	}

	titleVisible := true
	if req.TitleVisible != nil {
		titleVisible = *req.TitleVisible
	}

	rv, err := h.reviewSvc.CreateReview(r.Context(), reviewapp.CreateReviewInput{
		ProposalID:    proposalID,
		ReviewerID:    userID,
		GlobalRating:  req.GlobalRating,
		Timeliness:    req.Timeliness,
		Communication: req.Communication,
		Quality:       req.Quality,
		Comment:       req.Comment,
		VideoURL:      req.VideoURL,
		TitleVisible:  titleVisible,
	})
	if err != nil {
		handleReviewError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, map[string]any{
		"data": response.ReviewFromDomain(rv),
	})
}

// ListByUser handles GET /api/v1/reviews/user/{userId}.
func (h *ReviewHandler) ListByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_user_id", "userId must be a valid UUID")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	reviews, nextCursor, err := h.reviewSvc.ListByUser(r.Context(), userID, cursor, limit)
	if err != nil {
		slog.Error("list reviews by user", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list reviews")
		return
	}

	items := make([]response.ReviewResponse, 0, len(reviews))
	for _, rv := range reviews {
		items = append(items, response.ReviewFromDomain(rv))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

// GetAverageRating handles GET /api/v1/reviews/average/{userId}.
func (h *ReviewHandler) GetAverageRating(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_user_id", "userId must be a valid UUID")
		return
	}

	avg, err := h.reviewSvc.GetAverageRating(r.Context(), userID)
	if err != nil {
		slog.Error("get average rating", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get average rating")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.AverageRatingResponse{
			Average: avg.Average,
			Count:   avg.Count,
		},
	})
}

// CanReview handles GET /api/v1/reviews/can-review/{proposalId}.
func (h *ReviewHandler) CanReview(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "proposalId"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "proposalId must be a valid UUID")
		return
	}

	can, err := h.reviewSvc.CanReview(r.Context(), proposalID, userID)
	if err != nil {
		slog.Error("can review", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to check review eligibility")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": map[string]bool{"can_review": can},
	})
}

func handleReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, reviewdomain.ErrInvalidRating):
		res.Error(w, http.StatusBadRequest, "invalid_rating", err.Error())
	case errors.Is(err, reviewdomain.ErrCommentTooLong):
		res.Error(w, http.StatusBadRequest, "comment_too_long", err.Error())
	case errors.Is(err, reviewdomain.ErrSelfReview):
		res.Error(w, http.StatusBadRequest, "self_review", err.Error())
	case errors.Is(err, reviewdomain.ErrAlreadyReviewed):
		res.Error(w, http.StatusConflict, "already_reviewed", err.Error())
	case errors.Is(err, reviewdomain.ErrNotParticipant):
		res.Error(w, http.StatusForbidden, "not_participant", err.Error())
	case errors.Is(err, reviewdomain.ErrNotCompleted):
		res.Error(w, http.StatusBadRequest, "not_completed", err.Error())
	default:
		slog.Error("unhandled review error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

