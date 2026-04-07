package handler

import (
	"log/slog"
	"net/http"

	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
)

// ListModerationItems handles GET /api/v1/admin/moderation.
func (h *AdminHandler) ListModerationItems(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r.URL.Query().Get("limit"), 20)
	page := parsePage(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	filters := repository.ModerationFilters{
		Source: r.URL.Query().Get("source"),
		Type:   r.URL.Query().Get("type"),
		Status: r.URL.Query().Get("status"),
		Sort:   r.URL.Query().Get("sort"),
		Page:   page,
		Limit:  limit,
	}

	items, total, err := h.svc.ListModerationItems(r.Context(), filters)
	if err != nil {
		slog.Error("admin list moderation items", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list moderation items")
		return
	}

	data := make([]response.ModerationItemResponse, 0, len(items))
	for _, item := range items {
		data = append(data, response.NewModerationItemResponse(item))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        data,
		"total":       total,
		"page":        page,
		"total_pages": totalPages(total, limit),
	})
}

// ModerationCount handles GET /api/v1/admin/moderation/count.
func (h *AdminHandler) ModerationCount(w http.ResponseWriter, r *http.Request) {
	count, err := h.svc.ModerationPendingCount(r.Context())
	if err != nil {
		slog.Error("admin moderation count", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get moderation count")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"count": count,
	})
}
