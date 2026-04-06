package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/port/repository"
)

// ListConversations returns all conversations with participants and stats for admin.
func (s *Service) ListConversations(ctx context.Context, cursorStr string, limit int, page int, sort string, filter string) ([]repository.AdminConversation, string, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	filters := repository.AdminConversationFilters{
		Cursor: cursorStr,
		Limit:  limit,
		Page:   page,
		Sort:   sort,
		Filter: filter,
	}

	conversations, nextCursor, total, err := s.adminConversations.List(ctx, filters)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list conversations: %w", err)
	}

	return conversations, nextCursor, total, nil
}

// GetConversation returns a single conversation with participants and stats for admin.
func (s *Service) GetConversation(ctx context.Context, conversationID uuid.UUID) (*repository.AdminConversation, error) {
	conv, err := s.adminConversations.GetByID(ctx, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	return conv, nil
}

// GetConversationMessages returns messages for a conversation (admin view, no user filter).
func (s *Service) GetConversationMessages(ctx context.Context, conversationID uuid.UUID, cursorStr string, limit int) ([]repository.AdminMessage, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	messages, nextCursor, err := s.adminConversations.ListMessages(ctx, conversationID, cursorStr, limit)
	if err != nil {
		return nil, "", fmt.Errorf("get conversation messages: %w", err)
	}

	return messages, nextCursor, nil
}
