package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"marketplace-backend/internal/domain/review"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
)

// ListReviews handles GET /api/v1/admin/reviews.
func (h *AdminHandler) ListReviews(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	page := parsePage(r.URL.Query().Get("page"))

	filters := repository.AdminReviewFilters{
		Search: r.URL.Query().Get("search"),
		Rating: parseRatingFilter(r.URL.Query().Get("rating")),
		Sort:   r.URL.Query().Get("sort"),
		Filter: r.URL.Query().Get("filter"),
		Page:   page,
		Limit:  limit,
	}

	items, total, err := h.svc.ListReviews(r.Context(), filters)
	if err != nil {
		slog.Error("admin list reviews", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list reviews")
		return
	}

	data := make([]response.AdminReviewResponse, 0, len(items))
	for _, item := range items {
		data = append(data, response.NewAdminReviewResponse(item))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        data,
		"total":       total,
		"page":        page,
		"total_pages": totalPages(total, limit),
	})
}

// GetReview handles GET /api/v1/admin/reviews/{id}.
func (h *AdminHandler) GetReview(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	item, err := h.svc.GetReview(r.Context(), id)
	if err != nil {
		handleAdminReviewError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.NewAdminReviewResponse(*item),
	})
}

// DeleteReview handles DELETE /api/v1/admin/reviews/{id}.
func (h *AdminHandler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.svc.DeleteReview(r.Context(), id); err != nil {
		handleAdminReviewError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListReviewReports handles GET /api/v1/admin/reviews/{id}/reports.
func (h *AdminHandler) ListReviewReports(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	reports, err := h.svc.ListReviewReports(r.Context(), id)
	if err != nil {
		slog.Error("admin list review reports", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list reports")
		return
	}

	items := make([]response.AdminReportResponse, 0, len(reports))
	for _, rp := range reports {
		items = append(items, response.NewAdminReportResponse(rp))
	}

	res.JSON(w, http.StatusOK, map[string]any{"data": items})
}

func parseRatingFilter(s string) int {
	if s == "" {
		return 0
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 1 || v > 5 {
		return 0
	}
	return v
}

func handleAdminReviewError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, review.ErrNotFound):
		res.Error(w, http.StatusNotFound, "review_not_found", "review not found")
	default:
		slog.Error("unhandled review error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
