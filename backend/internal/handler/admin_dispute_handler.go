package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	disputeapp "marketplace-backend/internal/app/dispute"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
	res "marketplace-backend/pkg/response"
	"marketplace-backend/pkg/validator"
)

// AdminDisputeHandler handles admin-only dispute endpoints.
type AdminDisputeHandler struct {
	svc      *disputeapp.Service
	disputes repository.DisputeRepository
}

func NewAdminDisputeHandler(svc *disputeapp.Service, disputes repository.DisputeRepository) *AdminDisputeHandler {
	return &AdminDisputeHandler{svc: svc, disputes: disputes}
}

func (h *AdminDisputeHandler) ListDisputes(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	statusFilter := r.URL.Query().Get("status")

	disputes, nextCursor, err := h.disputes.ListAll(r.Context(), cursor, 20, statusFilter)
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list disputes")
		return
	}

	items := make([]response.DisputeResponse, 0, len(disputes))
	for _, d := range disputes {
		items = append(items, response.NewDisputeListItem(d))
	}

	res.JSON(w, http.StatusOK, response.DisputeListResponse{
		Data: items, NextCursor: nextCursor, HasMore: nextCursor != "",
	})
}

func (h *AdminDisputeHandler) GetAdminDispute(w http.ResponseWriter, r *http.Request) {
	disputeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid dispute ID")
		return
	}

	detail, err := h.svc.GetDisputeForAdmin(r.Context(), disputeID)
	if err != nil {
		handleDisputeError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewAdminDisputeResponse(
		detail.Dispute, detail.Evidence, detail.CounterProposals))
}

func (h *AdminDisputeHandler) ResolveDispute(w http.ResponseWriter, r *http.Request) {
	adminID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "admin not found in context")
		return
	}

	disputeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid dispute ID")
		return
	}

	var req request.AdminResolveDisputeRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	if err := h.svc.AdminResolve(r.Context(), disputeapp.AdminResolveInput{
		DisputeID:      disputeID,
		AdminID:        adminID,
		AmountClient:   req.AmountClient,
		AmountProvider: req.AmountProvider,
		Note:           req.Note,
	}); err != nil {
		handleDisputeError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "resolved"})
}

func (h *AdminDisputeHandler) CountDisputes(w http.ResponseWriter, r *http.Request) {
	total, open, escalated, err := h.disputes.CountAll(r.Context())
	if err != nil {
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to count disputes")
		return
	}

	res.JSON(w, http.StatusOK, map[string]int{
		"total": total, "open": open, "escalated": escalated,
	})
}
