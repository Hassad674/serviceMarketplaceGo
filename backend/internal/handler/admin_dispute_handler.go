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
	"marketplace-backend/internal/system"
	res "marketplace-backend/pkg/response"
	"marketplace-backend/pkg/validator"
)

// AdminDisputeHandler handles admin-only dispute endpoints.
//
// disputes is narrowed to DisputeReader — admin endpoints only list
// (ListAll) and count (CountAll); every state mutation goes through
// the dispute app service.
type AdminDisputeHandler struct {
	svc       *disputeapp.Service
	disputes  repository.DisputeReader
	isDevMode bool // Enables /force-escalate for dev/staging only.
}

func NewAdminDisputeHandler(svc *disputeapp.Service, disputes repository.DisputeReader, isDevMode bool) *AdminDisputeHandler {
	return &AdminDisputeHandler{svc: svc, disputes: disputes, isDevMode: isDevMode}
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

	// Admin read on a dispute the admin is NOT a tenant of —
	// system-actor surface, gated by middleware.RequireRole("admin")
	// + middleware.RequireAdmin on the route (see routes_admin.go).
	ctx := system.WithSystemActor(r.Context())
	detail, err := h.svc.GetDisputeForAdmin(ctx, disputeID)
	if err != nil {
		handleDisputeError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewAdminDisputeResponse(
		detail.Dispute, detail.Evidence, detail.CounterProposals, detail.ChatMessages))
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

	// Admin resolve fans out to dispute restore + payout
	// distribution on a dispute the admin is NOT a tenant of.
	ctx := system.WithSystemActor(r.Context())
	if err := h.svc.AdminResolve(ctx, disputeapp.AdminResolveInput{
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

// AskAI handles an admin chat question about a dispute. The backend
// loads the chat history from the database — the request body only
// carries the new question. This ensures the AI sees a tamper-proof
// context and that multiple admins working on the same dispute share
// the conversation seamlessly.
func (h *AdminDisputeHandler) AskAI(w http.ResponseWriter, r *http.Request) {
	disputeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid dispute ID")
		return
	}

	var req request.AskAIDisputeRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.Question == "" {
		res.Error(w, http.StatusBadRequest, "empty_question", "question is required")
		return
	}

	// Admin AI chat reads + persists messages on a dispute the
	// admin is NOT a tenant of — system-actor surface.
	ctx := system.WithSystemActor(r.Context())
	out, err := h.svc.AskAI(ctx, disputeapp.AskAIInput{
		DisputeID: disputeID,
		Question:  req.Question,
	})
	if err != nil {
		handleDisputeError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"answer":        out.Answer,
		"input_tokens":  out.InputTokens,
		"output_tokens": out.OutputTokens,
	})
}

// IncreaseAIBudget grants the dispute extra AI tokens via the admin
// "Augmenter le budget" button. Always adds the default increment.
func (h *AdminDisputeHandler) IncreaseAIBudget(w http.ResponseWriter, r *http.Request) {
	disputeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid dispute ID")
		return
	}

	// Admin budget increase is a system-actor surface — the
	// admin is acting on a dispute they are not party to.
	ctx := system.WithSystemActor(r.Context())
	if err := h.svc.IncreaseAIBudget(ctx, disputeID, 0); err != nil {
		handleDisputeError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"status":          "ok",
		"bonus_increment": disputeapp.AIBudgetBonusIncrement,
	})
}

// ForceEscalate is a development-only endpoint that immediately moves a
// dispute to escalated status, bypassing the 7-day inactivity window. It
// returns 404 in production so the route is invisible there even if the
// caller knows the URL.
func (h *AdminDisputeHandler) ForceEscalate(w http.ResponseWriter, r *http.Request) {
	if !h.isDevMode {
		res.Error(w, http.StatusNotFound, "not_found", "")
		return
	}

	disputeID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "invalid dispute ID")
		return
	}

	// Admin force-escalate acts on a dispute the admin is NOT a
	// tenant of — the dispute service routes through its
	// system-actor branch so the read goes through the
	// non-tenant-aware GetByID under a privileged DB role.
	ctx := system.WithSystemActor(r.Context())
	if err := h.svc.ForceEscalate(ctx, disputeID); err != nil {
		handleDisputeError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "escalated"})
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
