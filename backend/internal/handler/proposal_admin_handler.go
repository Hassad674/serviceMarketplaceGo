package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	proposalapp "marketplace-backend/internal/app/proposal"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/system"

	res "marketplace-backend/pkg/response"
)

// ProposalAdminHandler owns the 5 admin-gated proposal endpoints —
// force-activation (testing) and the bonus-credit log moderation
// queue.
//
// SRP rationale: every method here is exposed under /api/v1/admin/...
// and gated by RequireRole("admin") in the router. They have nothing
// to do with the user-facing lifecycle / payment / completion flows
// — separating them here keeps the production handlers free of
// admin-only branches.
//
// Dependencies:
//   - proposalSvc: drives the ConfirmPaymentAndActivate (force-activate)
//     and bonus-log moderation use cases.
type ProposalAdminHandler struct {
	proposalSvc *proposalapp.Service
}

// NewProposalAdminHandler wires the admin handler.
func NewProposalAdminHandler(svc *proposalapp.Service) *ProposalAdminHandler {
	return &ProposalAdminHandler{proposalSvc: svc}
}

// AdminActivateProposal handles POST /api/v1/admin/proposals/{id}/activate.
// Force-activates a proposal to paid+active state. Used for testing
// and to recover from edge cases where the webhook/confirm flow
// failed but the underlying Stripe charge actually settled.
func (h *ProposalAdminHandler) AdminActivateProposal(w http.ResponseWriter, r *http.Request) {
	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}
	// Admin force-activation is a system-actor surface: the admin
	// is acting on a proposal they are NOT a tenant of. Mark the
	// context so the proposal service skips the per-tenant gate
	// inside ConfirmPaymentAndActivate.
	ctx := system.WithSystemActor(r.Context())
	if err := h.proposalSvc.ConfirmPaymentAndActivate(ctx, proposalID); err != nil {
		handleProposalError(w, err)
		return
	}
	res.JSON(w, http.StatusOK, map[string]string{"status": "activated"})
}

// AdminListBonusLog handles GET /api/v1/admin/credits/bonus-log.
// Returns the chronological audit log of every bonus credit operation
// (grants, redemptions, adjustments).
func (h *ProposalAdminHandler) AdminListBonusLog(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	entries, nextCursor, err := h.proposalSvc.ListBonusLog(r.Context(), cursor, limit)
	if err != nil {
		slog.Error("list bonus log", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list bonus log")
		return
	}

	items := make([]response.BonusLogResponse, 0, len(entries))
	for _, e := range entries {
		items = append(items, response.NewBonusLogResponse(e))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

// AdminListPendingBonusLog handles GET /api/v1/admin/credits/bonus-log/pending.
// Filters the bonus log to entries awaiting moderator action.
func (h *ProposalAdminHandler) AdminListPendingBonusLog(w http.ResponseWriter, r *http.Request) {
	cursor := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	entries, nextCursor, err := h.proposalSvc.ListPendingBonusLog(r.Context(), cursor, limit)
	if err != nil {
		slog.Error("list pending bonus log", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to list pending bonus log")
		return
	}

	items := make([]response.BonusLogResponse, 0, len(entries))
	for _, e := range entries {
		items = append(items, response.NewBonusLogResponse(e))
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

// AdminApproveBonusEntry handles POST /api/v1/admin/credits/bonus-log/{id}/approve.
func (h *ProposalAdminHandler) AdminApproveBonusEntry(w http.ResponseWriter, r *http.Request) {
	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.proposalSvc.ApproveBonusEntry(r.Context(), entryID); err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "approved"})
}

// AdminRejectBonusEntry handles POST /api/v1/admin/credits/bonus-log/{id}/reject.
func (h *ProposalAdminHandler) AdminRejectBonusEntry(w http.ResponseWriter, r *http.Request) {
	entryID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_id", "id must be a valid UUID")
		return
	}

	if err := h.proposalSvc.RejectBonusEntry(r.Context(), entryID); err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}
