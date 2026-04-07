package admin

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// ApproveMessageModeration clears the moderation flag on a message, marking it clean.
func (s *Service) ApproveMessageModeration(ctx context.Context, messageID uuid.UUID) error {
	if err := s.adminConversations.UpdateMessageModeration(ctx, messageID, "clean", 0, nil); err != nil {
		return fmt.Errorf("approve message moderation: %w", err)
	}
	return nil
}

// HideMessage hides a message from users by setting its moderation status to 'hidden'.
func (s *Service) HideMessage(ctx context.Context, messageID uuid.UUID) error {
	if err := s.adminConversations.HideMessage(ctx, messageID); err != nil {
		return fmt.Errorf("hide message: %w", err)
	}
	return nil
}
