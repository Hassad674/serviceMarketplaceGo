package handler

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"marketplace-backend/internal/app/messaging"
	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/handler/dto/response"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/pkg/validator"

	res "marketplace-backend/pkg/response"
)

type MessagingHandler struct {
	messagingSvc *messaging.Service
}

func NewMessagingHandler(messagingSvc *messaging.Service) *MessagingHandler {
	return &MessagingHandler{messagingSvc: messagingSvc}
}

func (h *MessagingHandler) StartConversation(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req struct {
		RecipientID string `json:"recipient_id"`
		Content     string `json:"content"`
		Type        string `json:"type"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	recipientID, err := uuid.Parse(req.RecipientID)
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_recipient_id", "recipient_id must be a valid UUID")
		return
	}

	msgType := message.MessageType(req.Type)
	if req.Type == "" {
		msgType = message.MessageTypeText
	}

	msg, convID, err := h.messagingSvc.StartConversation(r.Context(), messaging.StartConversationInput{
		SenderID:    userID,
		RecipientID: recipientID,
		Content:     req.Content,
		Type:        msgType,
	})
	if err != nil {
		handleMessagingError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, response.StartConversationResponse{
		ConversationID: convID.String(),
		Message:        response.NewMessageResponse(msg),
	})
}

func (h *MessagingHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	cursorStr := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	summaries, nextCursor, err := h.messagingSvc.ListConversations(r.Context(), userID, cursorStr, limit)
	if err != nil {
		handleMessagingError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        response.NewConversationListResponse(summaries),
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

func (h *MessagingHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_conversation_id", "id must be a valid UUID")
		return
	}

	cursorStr := r.URL.Query().Get("cursor")
	limit := parseLimit(r.URL.Query().Get("limit"), 20)

	messages, nextCursor, err := h.messagingSvc.ListMessages(r.Context(), userID, convID, cursorStr, limit)
	if err != nil {
		handleMessagingError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]any{
		"data":        response.NewMessageListResponse(messages),
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

func (h *MessagingHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_conversation_id", "id must be a valid UUID")
		return
	}

	var req struct {
		Content string `json:"content"`
		Type    string `json:"type"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	msgType := message.MessageType(req.Type)
	if req.Type == "" {
		msgType = message.MessageTypeText
	}

	msg, err := h.messagingSvc.SendMessage(r.Context(), messaging.SendMessageInput{
		SenderID:       userID,
		ConversationID: convID,
		Content:        req.Content,
		Type:           msgType,
	})
	if err != nil {
		handleMessagingError(w, err)
		return
	}

	res.JSON(w, http.StatusCreated, response.NewMessageResponse(msg))
}

func (h *MessagingHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	convID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_conversation_id", "id must be a valid UUID")
		return
	}

	var req struct {
		Seq int `json:"seq"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	err = h.messagingSvc.MarkAsRead(r.Context(), messaging.MarkAsReadInput{
		UserID:         userID,
		ConversationID: convID,
		Seq:            req.Seq,
	})
	if err != nil {
		handleMessagingError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *MessagingHandler) EditMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_message_id", "id must be a valid UUID")
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	msg, err := h.messagingSvc.EditMessage(r.Context(), messaging.EditMessageInput{
		UserID:    userID,
		MessageID: msgID,
		Content:   req.Content,
	})
	if err != nil {
		handleMessagingError(w, err)
		return
	}

	res.JSON(w, http.StatusOK, response.NewMessageResponse(msg))
}

func (h *MessagingHandler) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	msgID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_message_id", "id must be a valid UUID")
		return
	}

	err = h.messagingSvc.DeleteMessage(r.Context(), messaging.DeleteMessageInput{
		UserID:    userID,
		MessageID: msgID,
	})
	if err != nil {
		handleMessagingError(w, err)
		return
	}

	res.NoContent(w)
}

func (h *MessagingHandler) GetPresignedURL(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	var req struct {
		Filename    string `json:"filename"`
		ContentType string `json:"content_type"`
	}

	if err := validator.DecodeJSON(r, &req); err != nil {
		res.Error(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	if req.Filename == "" || req.ContentType == "" {
		res.Error(w, http.StatusBadRequest, "validation_error", "filename and content_type are required")
		return
	}

	uploadURL, publicURL, err := h.messagingSvc.GetPresignedUploadURL(r.Context(), messaging.GetPresignedURLInput{
		UserID:      userID,
		Filename:    req.Filename,
		ContentType: req.ContentType,
	})
	if err != nil {
		slog.Error("presigned url generation failed", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to generate upload URL")
		return
	}

	res.JSON(w, http.StatusOK, response.PresignedURLResponse{
		UploadURL: uploadURL,
		PublicURL: publicURL,
	})
}

func (h *MessagingHandler) GetTotalUnread(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		res.Error(w, http.StatusUnauthorized, "unauthorized", "user not found in context")
		return
	}

	count, err := h.messagingSvc.GetTotalUnread(r.Context(), userID)
	if err != nil {
		slog.Error("get unread count failed", "error", err)
		res.Error(w, http.StatusInternalServerError, "internal_error", "failed to get unread count")
		return
	}

	res.JSON(w, http.StatusOK, response.UnreadCountResponse{Count: count})
}

func parseLimit(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(s)
	if err != nil || val <= 0 || val > 100 {
		return defaultVal
	}
	return val
}

func handleMessagingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, message.ErrConversationNotFound):
		res.Error(w, http.StatusNotFound, "conversation_not_found", err.Error())
	case errors.Is(err, message.ErrMessageNotFound):
		res.Error(w, http.StatusNotFound, "message_not_found", err.Error())
	case errors.Is(err, message.ErrNotParticipant):
		res.Error(w, http.StatusForbidden, "not_participant", err.Error())
	case errors.Is(err, message.ErrEmptyContent):
		res.Error(w, http.StatusBadRequest, "empty_content", err.Error())
	case errors.Is(err, message.ErrContentTooLong):
		res.Error(w, http.StatusBadRequest, "content_too_long", err.Error())
	case errors.Is(err, message.ErrInvalidMessageType):
		res.Error(w, http.StatusBadRequest, "invalid_message_type", err.Error())
	case errors.Is(err, message.ErrCannotEditOther):
		res.Error(w, http.StatusForbidden, "cannot_edit_other", err.Error())
	case errors.Is(err, message.ErrCannotDeleteOther):
		res.Error(w, http.StatusForbidden, "cannot_delete_other", err.Error())
	case errors.Is(err, message.ErrMessageDeleted):
		res.Error(w, http.StatusBadRequest, "message_deleted", err.Error())
	case errors.Is(err, message.ErrSelfConversation):
		res.Error(w, http.StatusBadRequest, "self_conversation", err.Error())
	case errors.Is(err, message.ErrRateLimitExceeded):
		res.Error(w, http.StatusTooManyRequests, "rate_limit", err.Error())
	default:
		slog.Error("unhandled messaging error", "error", err.Error())
		res.Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
	}
}
