package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	proposalapp "marketplace-backend/internal/app/proposal"
	proposaldomain "marketplace-backend/internal/domain/proposal"
	"marketplace-backend/internal/handler/dto/request"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

type ProposalHandler struct {
	proposalSvc *proposalapp.Service
}

func NewProposalHandler(svc *proposalapp.Service) *ProposalHandler {
	return &ProposalHandler{proposalSvc: svc}
}

func (h *ProposalHandler) CreateProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req request.CreateProposalRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	recipientID, err := uuid.Parse(req.RecipientID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_recipient_id", "recipient_id must be a valid UUID")
		return
	}

	convID, err := uuid.Parse(req.ConversationID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_conversation_id", "conversation_id must be a valid UUID")
		return
	}

	var deadline *time.Time
	if req.Deadline != "" {
		t, err := time.Parse(time.RFC3339, req.Deadline)
		if err != nil {
			t, err = time.Parse("2006-01-02", req.Deadline)
			if err != nil {
				res.Error(w, http.StatusBadRequest, "invalid_deadline", "deadline must be RFC3339 or YYYY-MM-DD")
				return
			}
		}
		deadline = &t
	}

	docs := convertDocumentInputs(req.Documents)

	p, err := h.proposalSvc.CreateProposal(r.Context(), proposalapp.CreateProposalInput{
		ConversationID: convID,
		SenderID:       userID,
		RecipientID:    recipientID,
		Title:          req.Title,
		Description:    req.Description,
		Amount:         req.Amount,
		Deadline:       deadline,
		Documents:      docs,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, response.NewProposalResponse(p, nil))
}

func (h *ProposalHandler) GetProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	p, docs, err := h.proposalSvc.GetProposal(r.Context(), userID, proposalID)
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewProposalResponse(p, docs))
}

func (h *ProposalHandler) AcceptProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.AcceptProposal(r.Context(), proposalapp.AcceptProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (h *ProposalHandler) DeclineProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.DeclineProposal(r.Context(), proposalapp.DeclineProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "declined"})
}

func (h *ProposalHandler) ModifyProposal(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	var req request.ModifyProposalRequest
	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	var deadline *time.Time
	if req.Deadline != "" {
		t, err := time.Parse(time.RFC3339, req.Deadline)
		if err != nil {
			t, err = time.Parse("2006-01-02", req.Deadline)
			if err != nil {
				res.Error(w, http.StatusBadRequest, "invalid_deadline", "deadline must be RFC3339 or YYYY-MM-DD")
				return
			}
		}
		deadline = &t
	}

	docs := convertDocumentInputs(req.Documents)

	p, err := h.proposalSvc.ModifyProposal(r.Context(), proposalapp.ModifyProposalInput{
		ProposalID:  proposalID,
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		Amount:      req.Amount,
		Deadline:    deadline,
		Documents:   docs,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewProposalResponse(p, nil))
}

func (h *ProposalHandler) SimulatePayment(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	proposalID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_proposal_id", "id must be a valid UUID")
		return
	}

	err = h.proposalSvc.SimulatePayment(r.Context(), proposalapp.PayProposalInput{
		ProposalID: proposalID,
		UserID:     userID,
	})
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "paid"})
}

func (h *ProposalHandler) ListActiveProjects(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	cursorStr := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	proposals, nextCursor, err := h.proposalSvc.ListActiveProjects(r.Context(), userID, cursorStr, limit)
	if err != nil {
		handleProposalError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewProjectListResponse(proposals, nextCursor))
}

func convertDocumentInputs(inputs []request.DocumentInput) []proposalapp.DocumentInput {
	docs := make([]proposalapp.DocumentInput, len(inputs))
	for i, d := range inputs {
		docs[i] = proposalapp.DocumentInput{
			Filename: d.Filename,
			URL:      d.URL,
			Size:     d.Size,
			MimeType: d.MimeType,
		}
	}
	return docs
}

func handleProposalError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, proposaldomain.ErrProposalNotFound):
		res.Error(w, http.StatusNotFound, "proposal_not_found", err.Error())
	case errors.Is(err, proposaldomain.ErrNotAuthorized):
		res.Error(w, http.StatusForbidden, "not_authorized", err.Error())
	case errors.Is(err, proposaldomain.ErrCannotModify):
		res.Error(w, http.StatusForbidden, "cannot_modify", err.Error())
	case errors.Is(err, proposaldomain.ErrInvalidStatus):
		res.Error(w, http.StatusConflict, "invalid_status", err.Error())
	case errors.Is(err, proposaldomain.ErrSameUser):
		res.Error(w, http.StatusBadRequest, "same_user", err.Error())
	case errors.Is(err, proposaldomain.ErrInvalidRoleCombination):
		res.Error(w, http.StatusBadRequest, "invalid_role_combination", err.Error())
	case errors.Is(err, proposaldomain.ErrEmptyTitle):
		res.Error(w, http.StatusBadRequest, "empty_title", err.Error())
	case errors.Is(err, proposaldomain.ErrEmptyDescription):
		res.Error(w, http.StatusBadRequest, "empty_description", err.Error())
	case errors.Is(err, proposaldomain.ErrInvalidAmount):
		res.Error(w, http.StatusBadRequest, "invalid_amount", err.Error())
	default:
		slog.Error("unhandled proposal error", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
