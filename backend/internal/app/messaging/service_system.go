package messaging

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/service"
)

// FindOrCreateConversation finds or creates a conversation between two users
// and sends an initial system message. Returns the conversation ID.
//
// Cross-feature callers (referral, job application) hit this from a
// background context. We resolve UserA's org so the postgres adapter
// can install RLS tenant context on the underlying transaction. A
// solo-provider UserA has no org and we fall back to uuid.Nil — the
// participant escape hatch on the conversations policy still admits
// the row through app.current_user_id.
func (s *Service) FindOrCreateConversation(ctx context.Context, input service.FindOrCreateConversationInput) (uuid.UUID, error) {
	var senderOrgID uuid.UUID
	if input.UserA != uuid.Nil {
		resolved, err := s.resolveUserOrgID(ctx, input.UserA)
		if err != nil {
			return uuid.Nil, fmt.Errorf("resolve userA org: %w", err)
		}
		senderOrgID = resolved
	}

	convID, _, err := s.messages.FindOrCreateConversation(ctx, input.UserA, input.UserB, senderOrgID, input.UserA)
	if err != nil {
		return uuid.Nil, fmt.Errorf("find or create conversation: %w", err)
	}

	if input.Content != "" {
		sysInput := service.SystemMessageInput{
			ConversationID: convID,
			SenderID:       input.UserA,
			Content:        input.Content,
			Type:           input.Type,
		}
		if err := s.SendSystemMessage(ctx, sysInput); err != nil {
			return uuid.Nil, fmt.Errorf("send system message: %w", err)
		}
	}

	return convID, nil
}

// SendSystemMessage injects a system-level message into a conversation.
// This is used by other features (e.g. proposals) to send event messages
// without rate limiting or participant verification.
//
// A zero SenderID (uuid.Nil) denotes a SYSTEM ACTOR — used by the
// scheduler worker and the end-of-project effects that do not attribute
// to a specific user. In that case the org-based exclusion is skipped
// (there is no sender org to exclude) and every participant gets the
// +1 unread bump, same as the broadcast path.
func (s *Service) SendSystemMessage(ctx context.Context, input service.SystemMessageInput) error {
	msgType := message.MessageType(input.Type)

	msg, err := message.NewMessage(message.NewMessageInput{
		ConversationID: input.ConversationID,
		SenderID:       input.SenderID,
		Content:        input.Content,
		Type:           msgType,
		Metadata:       input.Metadata,
	})
	if err != nil {
		return fmt.Errorf("create system message: %w", err)
	}

	// Resolve the sender's org so the +1 unread bump excludes their
	// whole team. For system-actor sends (uuid.Nil) there is no org
	// to look up: pass uuid.Nil through and let the repo bump every
	// participant. Previously this path called resolveUserOrgID with
	// uuid.Nil which returned "user not found" and caused the whole
	// message to silently drop — notably the proposal_completed and
	// evaluation_request notifications at end of project.
	var senderOrgID uuid.UUID
	if input.SenderID != uuid.Nil {
		senderOrgID, err = s.resolveUserOrgID(ctx, input.SenderID)
		if err != nil {
			return fmt.Errorf("resolve system sender org: %w", err)
		}
	}

	if err := s.messages.CreateMessage(ctx, msg, senderOrgID, input.SenderID); err != nil {
		return fmt.Errorf("persist system message: %w", err)
	}
	if err := s.messages.IncrementUnreadForRecipients(ctx, input.ConversationID, input.SenderID, senderOrgID); err != nil {
		return fmt.Errorf("increment unread: %w", err)
	}

	s.broadcastSystemMessage(ctx, input.ConversationID, input.SenderID, msg)

	return nil
}
