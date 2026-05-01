package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

func TestSendSystemMessage_Success(t *testing.T) {
	convID := uuid.New()
	senderID := uuid.New()
	recipientID := uuid.New()

	var createdMsg *message.Message
	var broadcastCalled bool
	msgRepo := &mockMessageRepo{
		createMessageFn: func(_ context.Context, msg *message.Message, _, _ uuid.UUID) error {
			createdMsg = msg
			return nil
		},
		getParticipantIDsFn: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{senderID, recipientID}, nil
		},
	}
	broadcaster := &mockBroadcaster{
		broadcastNewMessageFn: func(_ context.Context, ids []uuid.UUID, _ []byte) error {
			broadcastCalled = true
			// System messages are sent to ALL participants including sender
			assert.Len(t, ids, 2)
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, broadcaster, nil, nil)

	meta, _ := json.Marshal(map[string]string{"proposal_id": uuid.New().String()})
	err := svc.SendSystemMessage(context.Background(), service.SystemMessageInput{
		ConversationID: convID,
		SenderID:       senderID,
		Content:        "Proposal sent",
		Type:           string(message.MessageTypeProposalSent),
		Metadata:       meta,
	})

	require.NoError(t, err)
	require.NotNil(t, createdMsg)
	assert.Equal(t, convID, createdMsg.ConversationID)
	assert.Equal(t, message.MessageTypeProposalSent, createdMsg.Type)
	assert.True(t, broadcastCalled)
}

func TestSendSystemMessage_InvalidType(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil, nil)

	err := svc.SendSystemMessage(context.Background(), service.SystemMessageInput{
		ConversationID: uuid.New(),
		SenderID:       uuid.New(),
		Content:        "bad type",
		Type:           "not_a_real_type",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "create system message")
}

func TestSendSystemMessage_CreateError(t *testing.T) {
	msgRepo := &mockMessageRepo{
		createMessageFn: func(_ context.Context, _ *message.Message, _, _ uuid.UUID) error {
			return fmt.Errorf("database write failed")
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.SendSystemMessage(context.Background(), service.SystemMessageInput{
		ConversationID: uuid.New(),
		SenderID:       uuid.New(),
		Content:        "Proposal accepted",
		Type:           string(message.MessageTypeProposalAccepted),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "persist system message")
}

func TestSendSystemMessage_IncrementUnreadError(t *testing.T) {
	senderID := uuid.New()
	senderOrgID := uuid.New()

	msgRepo := &mockMessageRepo{
		incrementUnreadForRecipientsFn: func(_ context.Context, _, _, _ uuid.UUID) error {
			return fmt.Errorf("redis error")
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{ID: id, OrganizationID: &senderOrgID}, nil
		},
	}

	svc := newTestServiceWithDeps(testServiceDeps{msgRepo: msgRepo, userRepo: userRepo})

	err := svc.SendSystemMessage(context.Background(), service.SystemMessageInput{
		ConversationID: uuid.New(),
		SenderID:       senderID,
		Content:        "Proposal declined",
		Type:           string(message.MessageTypeProposalDeclined),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "increment unread")
}

// TestSendSystemMessage_NilSender verifies the system-actor branch:
// when SenderID is uuid.Nil (used by runEndOfProjectEffects for
// proposal_completed and evaluation_request), the service must NOT
// try to resolve the sender's org (which would fail "user not found"
// and silently drop the message). It should persist the message and
// trigger the unread fan-out with no exclusion.
func TestSendSystemMessage_NilSender(t *testing.T) {
	convID := uuid.New()
	var createdMsg *message.Message
	var unreadCalled bool
	var observedSenderUserID, observedSenderOrgID uuid.UUID

	msgRepo := &mockMessageRepo{
		createMessageFn: func(_ context.Context, msg *message.Message, _, _ uuid.UUID) error {
			createdMsg = msg
			return nil
		},
		incrementUnreadForRecipientsFn: func(_ context.Context, _, sender, org uuid.UUID) error {
			unreadCalled = true
			observedSenderUserID = sender
			observedSenderOrgID = org
			return nil
		},
	}
	// Fail loudly if anyone tries to look up the nil user — the fix
	// must short-circuit before we get here.
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			t.Fatal("resolveUserOrgID must NOT be called for system-actor sends")
			return nil, fmt.Errorf("unreachable")
		},
	}

	svc := newTestServiceWithDeps(testServiceDeps{
		msgRepo:  msgRepo,
		userRepo: userRepo,
	})

	err := svc.SendSystemMessage(context.Background(), service.SystemMessageInput{
		ConversationID: convID,
		SenderID:       uuid.Nil, // system actor
		Content:        "",
		Type:           string(message.MessageTypeProposalCompleted),
	})

	require.NoError(t, err, "system-actor send must not error")
	require.NotNil(t, createdMsg, "message must be persisted")
	assert.True(t, unreadCalled, "unread fan-out must run")
	assert.Equal(t, uuid.Nil, observedSenderUserID, "sender user passed unchanged")
	assert.Equal(t, uuid.Nil, observedSenderOrgID, "sender org should be nil — no exclusion")
}

func TestSendSystemMessage_ProposalTypes(t *testing.T) {
	tests := []struct {
		name    string
		msgType message.MessageType
	}{
		{name: "proposal sent", msgType: message.MessageTypeProposalSent},
		{name: "proposal accepted", msgType: message.MessageTypeProposalAccepted},
		{name: "proposal declined", msgType: message.MessageTypeProposalDeclined},
		{name: "proposal modified", msgType: message.MessageTypeProposalModified},
		{name: "proposal paid", msgType: message.MessageTypeProposalPaid},
		{name: "proposal completed", msgType: message.MessageTypeProposalCompleted},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var savedType message.MessageType
			msgRepo := &mockMessageRepo{
				createMessageFn: func(_ context.Context, msg *message.Message, _, _ uuid.UUID) error {
					savedType = msg.Type
					return nil
				},
			}

			svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

			err := svc.SendSystemMessage(context.Background(), service.SystemMessageInput{
				ConversationID: uuid.New(),
				SenderID:       uuid.New(),
				Content:        "",
				Type:           string(tt.msgType),
			})

			require.NoError(t, err)
			assert.Equal(t, tt.msgType, savedType)
		})
	}
}
