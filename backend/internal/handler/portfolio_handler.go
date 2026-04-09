package handler

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	portfolioapp "marketplace-backend/internal/app/portfolio"
	portfoliodomain "marketplace-backend/internal/domain/portfolio"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

// PortfolioHandler handles HTTP requests for portfolio items.
type PortfolioHandler struct {
	portfolioSvc *portfolioapp.Service
}

// NewPortfolioHandler creates a new PortfolioHandler.
func NewPortfolioHandler(svc *portfolioapp.Service) *PortfolioHandler {
	return &PortfolioHandler{portfolioSvc: svc}
}

// CreatePortfolioItem handles POST /api/v1/portfolio.
func (h *PortfolioHandler) CreatePortfolioItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.CreatePortfolioItemRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	media := make([]portfolioapp.MediaInput, len(req.Media))
	for i, m := range req.Media {
		media[i] = portfolioapp.MediaInput{
			MediaURL:     m.MediaURL,
			MediaType:    m.MediaType,
			ThumbnailURL: m.ThumbnailURL,
			Position:     m.Position,
		}
	}

	item, err := h.portfolioSvc.CreateItem(r.Context(), portfolioapp.CreateItemInput{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		LinkURL:     req.LinkURL,
		Position:    req.Position,
		Media:       media,
	})
	if err != nil {
		handlePortfolioError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, map[string]any{
		"data": response.PortfolioItemFromDomain(item),
	})
}

// GetPortfolioItem handles GET /api/v1/portfolio/{id}.
func (h *PortfolioHandler) GetPortfolioItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	item, err := h.portfolioSvc.GetByID(r.Context(), id)
	if err != nil {
		handlePortfolioError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.PortfolioItemFromDomain(item),
	})
}

// ListPortfolioByUser handles GET /api/v1/portfolio/user/{userId}.
func (h *PortfolioHandler) ListPortfolioByUser(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_user_id", "userId must be a valid UUID")
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	items, nextCursor, err := h.portfolioSvc.ListByUser(r.Context(), userID, cursor, limit)
	if err != nil {
		slog.Error("list portfolio by user", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list portfolio")
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        response.PortfolioListFromDomain(items),
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

// UpdatePortfolioItem handles PUT /api/v1/portfolio/{id}.
func (h *PortfolioHandler) UpdatePortfolioItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	var req request.UpdatePortfolioItemRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var media []portfolioapp.MediaInput
	if req.Media != nil {
		media = make([]portfolioapp.MediaInput, len(req.Media))
		for i, m := range req.Media {
			media[i] = portfolioapp.MediaInput{
				MediaURL:     m.MediaURL,
				MediaType:    m.MediaType,
				ThumbnailURL: m.ThumbnailURL,
				Position:     m.Position,
			}
		}
	}

	item, err := h.portfolioSvc.UpdateItem(r.Context(), userID, id, portfolioapp.UpdateItemInput{
		Title:       req.Title,
		Description: req.Description,
		LinkURL:     req.LinkURL,
		Media:       media,
	})
	if err != nil {
		handlePortfolioError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data": response.PortfolioItemFromDomain(item),
	})
}

// DeletePortfolioItem handles DELETE /api/v1/portfolio/{id}.
func (h *PortfolioHandler) DeletePortfolioItem(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.portfolioSvc.DeleteItem(r.Context(), userID, id); err != nil {
		handlePortfolioError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ReorderPortfolio handles PUT /api/v1/portfolio/reorder.
func (h *PortfolioHandler) ReorderPortfolio(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.ReorderPortfolioRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	itemIDs := make([]uuid.UUID, len(req.ItemIDs))
	for i, raw := range req.ItemIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			res.Error(w, http.StatusBadRequest, "invalid_id", "item_ids must be valid UUIDs")
			return
		}
		itemIDs[i] = id
	}

	if err := h.portfolioSvc.ReorderItems(r.Context(), userID, itemIDs); err != nil {
		slog.Error("reorder portfolio", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to reorder")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func handlePortfolioError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, portfoliodomain.ErrNotFound):
		res.Error(w, http.StatusNotFound, "not_found", err.Error())
	case errors.Is(err, portfoliodomain.ErrNotOwner):
		res.Error(w, http.StatusForbidden, "forbidden", err.Error())
	case errors.Is(err, portfoliodomain.ErrTooManyItems):
		res.Error(w, http.StatusConflict, "too_many_items", err.Error())
	case errors.Is(err, portfoliodomain.ErrTooManyMedia):
		res.Error(w, http.StatusBadRequest, "too_many_media", err.Error())
	case errors.Is(err, portfoliodomain.ErrMissingTitle),
		errors.Is(err, portfoliodomain.ErrTitleTooLong),
		errors.Is(err, portfoliodomain.ErrDescriptionTooLong),
		errors.Is(err, portfoliodomain.ErrLinkURLTooLong),
		errors.Is(err, portfoliodomain.ErrInvalidLinkURL),
		errors.Is(err, portfoliodomain.ErrInvalidPosition),
		errors.Is(err, portfoliodomain.ErrInvalidMediaType),
		errors.Is(err, portfoliodomain.ErrMissingMediaURL):
		res.Error(w, http.StatusBadRequest, "validation_error", err.Error())
	default:
		slog.Error("unhandled portfolio error", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}
