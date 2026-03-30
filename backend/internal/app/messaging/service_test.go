package messaging

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// --- helpers ---

func newTestService(
	msgRepo *mockMessageRepo,
	userRepo *mockUserRepo,
	presence *mockPresenceService,
	broadcaster *mockBroadcaster,
	storage *mockStorageService,
	rateLimiter *mockRateLimiter,
) *Service {
	if msgRepo == nil {
		msgRepo = &mockMessageRepo{}
	}
	if userRepo == nil {
		userRepo = &mockUserRepo{}
	}
	if presence == nil {
		presence = &mockPresenceService{}
	}
	if broadcaster == nil {
		broadcaster = &mockBroadcaster{}
	}
	if storage == nil {
		storage = &mockStorageService{}
	}
	if rateLimiter == nil {
		rateLimiter = &mockRateLimiter{}
	}
	return NewService(ServiceDeps{
		Messages:    msgRepo,
		Users:       userRepo,
		Presence:    presence,
		Broadcaster: broadcaster,
		Storage:     storage,
		RateLimiter: rateLimiter,
	})
}

// --- StartConversation tests ---

func TestStartConversation_Success(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	convID := uuid.New()

	var createdMsg *message.Message
	msgRepo := &mockMessageRepo{
		findOrCreateConversationFn: func(_ context.Context, a, b uuid.UUID) (uuid.UUID, bool, error) {
			return convID, true, nil
		},
		createMessageFn: func(_ context.Context, msg *message.Message) error {
			createdMsg = msg
			return nil
		},
		getParticipantIDsFn: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{senderID, recipientID}, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{ID: id}, nil
		},
	}

	svc := newTestService(msgRepo, userRepo, nil, nil, nil, nil)

	msg, returnedConvID, err := svc.StartConversation(context.Background(), StartConversationInput{
		SenderID:    senderID,
		RecipientID: recipientID,
		Content:     "Hello!",
		Type:        message.MessageTypeText,
	})

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, convID, returnedConvID)
	assert.Equal(t, "Hello!", createdMsg.Content)
	assert.Equal(t, senderID, createdMsg.SenderID)
}

func TestStartConversation_SelfConversation(t *testing.T) {
	selfID := uuid.New()
	svc := newTestService(nil, nil, nil, nil, nil, nil)

	msg, _, err := svc.StartConversation(context.Background(), StartConversationInput{
		SenderID:    selfID,
		RecipientID: selfID,
		Content:     "talking to myself",
		Type:        message.MessageTypeText,
	})

	assert.ErrorIs(t, err, message.ErrSelfConversation)
	assert.Nil(t, msg)
}

func TestStartConversation_RecipientNotFound(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
			return nil, user.ErrUserNotFound
		},
	}

	svc := newTestService(nil, userRepo, nil, nil, nil, nil)

	msg, _, err := svc.StartConversation(context.Background(), StartConversationInput{
		SenderID:    uuid.New(),
		RecipientID: uuid.New(),
		Content:     "hello",
		Type:        message.MessageTypeText,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get recipient")
	assert.Nil(t, msg)
}

func TestStartConversation_RateLimited(t *testing.T) {
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{ID: id}, nil
		},
	}
	rateLimiter := &mockRateLimiter{
		allowFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := newTestService(nil, userRepo, nil, nil, nil, rateLimiter)

	msg, _, err := svc.StartConversation(context.Background(), StartConversationInput{
		SenderID:    uuid.New(),
		RecipientID: uuid.New(),
		Content:     "spam",
		Type:        message.MessageTypeText,
	})

	assert.ErrorIs(t, err, message.ErrRateLimitExceeded)
	assert.Nil(t, msg)
}

// --- SendMessage tests ---

func TestSendMessage_Success(t *testing.T) {
	senderID := uuid.New()
	convID := uuid.New()
	recipientID := uuid.New()

	var broadcastCalled bool
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		getParticipantIDsFn: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{senderID, recipientID}, nil
		},
	}
	broadcaster := &mockBroadcaster{
		broadcastNewMessageFn: func(_ context.Context, ids []uuid.UUID, _ []byte) error {
			broadcastCalled = true
			assert.Len(t, ids, 1)
			assert.Equal(t, recipientID, ids[0])
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, broadcaster, nil, nil)

	msg, err := svc.SendMessage(context.Background(), SendMessageInput{
		SenderID:       senderID,
		ConversationID: convID,
		Content:        "Hi there",
		Type:           message.MessageTypeText,
	})

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, "Hi there", msg.Content)
	assert.True(t, broadcastCalled, "broadcast should be called for new message")
}

func TestSendMessage_NotParticipant(t *testing.T) {
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	msg, err := svc.SendMessage(context.Background(), SendMessageInput{
		SenderID:       uuid.New(),
		ConversationID: uuid.New(),
		Content:        "unauthorized",
		Type:           message.MessageTypeText,
	})

	assert.ErrorIs(t, err, message.ErrNotParticipant)
	assert.Nil(t, msg)
}

func TestSendMessage_EmptyContent(t *testing.T) {
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	msg, err := svc.SendMessage(context.Background(), SendMessageInput{
		SenderID:       uuid.New(),
		ConversationID: uuid.New(),
		Content:        "",
		Type:           message.MessageTypeText,
	})

	assert.ErrorIs(t, err, message.ErrEmptyContent)
	assert.Nil(t, msg)
}

func TestSendMessage_RateLimited(t *testing.T) {
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	rateLimiter := &mockRateLimiter{
		allowFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, rateLimiter)

	msg, err := svc.SendMessage(context.Background(), SendMessageInput{
		SenderID:       uuid.New(),
		ConversationID: uuid.New(),
		Content:        "spam",
		Type:           message.MessageTypeText,
	})

	assert.ErrorIs(t, err, message.ErrRateLimitExceeded)
	assert.Nil(t, msg)
}

// --- EditMessage tests ---

func TestEditMessage_Success(t *testing.T) {
	ownerID := uuid.New()
	msgID := uuid.New()
	existingMsg := &message.Message{
		ID:        msgID,
		SenderID:  ownerID,
		Content:   "original",
		Type:      message.MessageTypeText,
		Status:    message.MessageStatusSent,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var updatedMsg *message.Message
	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, id uuid.UUID) (*message.Message, error) {
			if id == msgID {
				return existingMsg, nil
			}
			return nil, message.ErrMessageNotFound
		},
		updateMessageFn: func(_ context.Context, msg *message.Message) error {
			updatedMsg = msg
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	result, err := svc.EditMessage(context.Background(), EditMessageInput{
		UserID:    ownerID,
		MessageID: msgID,
		Content:   "edited content",
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "edited content", result.Content)
	assert.NotNil(t, result.EditedAt)
	assert.NotNil(t, updatedMsg)
}

func TestEditMessage_NotOwner(t *testing.T) {
	msgID := uuid.New()
	existingMsg := &message.Message{
		ID:       msgID,
		SenderID: uuid.New(),
		Content:  "original",
	}

	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
			return existingMsg, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	result, err := svc.EditMessage(context.Background(), EditMessageInput{
		UserID:    uuid.New(), // different from sender
		MessageID: msgID,
		Content:   "hacked",
	})

	assert.ErrorIs(t, err, message.ErrCannotEditOther)
	assert.Nil(t, result)
}

func TestEditMessage_DeletedMessage(t *testing.T) {
	ownerID := uuid.New()
	msgID := uuid.New()
	deletedAt := time.Now()
	existingMsg := &message.Message{
		ID:        msgID,
		SenderID:  ownerID,
		Content:   "",
		DeletedAt: &deletedAt,
	}

	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
			return existingMsg, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	result, err := svc.EditMessage(context.Background(), EditMessageInput{
		UserID:    ownerID,
		MessageID: msgID,
		Content:   "edit deleted",
	})

	assert.ErrorIs(t, err, message.ErrMessageDeleted)
	assert.Nil(t, result)
}

func TestEditMessage_EmptyContent(t *testing.T) {
	ownerID := uuid.New()
	msgID := uuid.New()
	existingMsg := &message.Message{
		ID:        msgID,
		SenderID:  ownerID,
		Content:   "original",
		CreatedAt: time.Now(),
	}

	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
			return existingMsg, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	result, err := svc.EditMessage(context.Background(), EditMessageInput{
		UserID:    ownerID,
		MessageID: msgID,
		Content:   "",
	})

	assert.ErrorIs(t, err, message.ErrEmptyContent)
	assert.Nil(t, result)
}

// --- DeleteMessage tests ---

func TestDeleteMessage_Success(t *testing.T) {
	ownerID := uuid.New()
	msgID := uuid.New()
	existingMsg := &message.Message{
		ID:        msgID,
		SenderID:  ownerID,
		Content:   "to be deleted",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	var updatedMsg *message.Message
	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
			return existingMsg, nil
		},
		updateMessageFn: func(_ context.Context, msg *message.Message) error {
			updatedMsg = msg
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.DeleteMessage(context.Background(), DeleteMessageInput{
		UserID:    ownerID,
		MessageID: msgID,
	})

	require.NoError(t, err)
	require.NotNil(t, updatedMsg)
	assert.Empty(t, updatedMsg.Content)
	assert.NotNil(t, updatedMsg.DeletedAt)
}

func TestDeleteMessage_NotOwner(t *testing.T) {
	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
			return &message.Message{
				ID:       uuid.New(),
				SenderID: uuid.New(),
				Content:  "not yours",
			}, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.DeleteMessage(context.Background(), DeleteMessageInput{
		UserID:    uuid.New(),
		MessageID: uuid.New(),
	})

	assert.ErrorIs(t, err, message.ErrCannotDeleteOther)
}

// --- MarkAsRead tests ---

func TestMarkAsRead_Success(t *testing.T) {
	userID := uuid.New()
	convID := uuid.New()

	var readCalled bool
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		markAsReadFn: func(_ context.Context, cID, uID uuid.UUID, seq int) error {
			readCalled = true
			assert.Equal(t, convID, cID)
			assert.Equal(t, userID, uID)
			assert.Equal(t, 42, seq)
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.MarkAsRead(context.Background(), MarkAsReadInput{
		UserID:         userID,
		ConversationID: convID,
		Seq:            42,
	})

	require.NoError(t, err)
	assert.True(t, readCalled)
}

func TestMarkAsRead_NotParticipant(t *testing.T) {
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.MarkAsRead(context.Background(), MarkAsReadInput{
		UserID:         uuid.New(),
		ConversationID: uuid.New(),
		Seq:            10,
	})

	assert.ErrorIs(t, err, message.ErrNotParticipant)
}

// --- ListConversations tests ---

func TestListConversations_Success(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()

	summaries := []repository.ConversationSummary{
		{
			ConversationID: uuid.New(),
			OtherUserID:    otherUserID,
			OtherUserName:  "Alice",
			OtherUserRole:  "provider",
			UnreadCount:    3,
		},
	}

	msgRepo := &mockMessageRepo{
		listConversationsFn: func(_ context.Context, params repository.ListConversationsParams) ([]repository.ConversationSummary, string, error) {
			assert.Equal(t, userID, params.UserID)
			return summaries, "next", nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	result, nextCursor, err := svc.ListConversations(context.Background(), userID, "", 20)

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Alice", result[0].OtherUserName)
	assert.Equal(t, "next", nextCursor)
}

// --- ListMessages tests ---

func TestListMessages_NotParticipant(t *testing.T) {
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	msgs, _, err := svc.ListMessages(context.Background(), uuid.New(), uuid.New(), "", 20)

	assert.ErrorIs(t, err, message.ErrNotParticipant)
	assert.Nil(t, msgs)
}

func TestListMessages_Success(t *testing.T) {
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		listMessagesFn: func(_ context.Context, _ repository.ListMessagesParams) ([]*message.Message, string, error) {
			return []*message.Message{
				{ID: uuid.New(), Content: "msg1"},
				{ID: uuid.New(), Content: "msg2"},
			}, "cursor", nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	msgs, nextCursor, err := svc.ListMessages(context.Background(), uuid.New(), uuid.New(), "", 20)

	require.NoError(t, err)
	assert.Len(t, msgs, 2)
	assert.Equal(t, "cursor", nextCursor)
}

// --- GetTotalUnread tests ---

func TestGetTotalUnread_Success(t *testing.T) {
	userID := uuid.New()
	msgRepo := &mockMessageRepo{
		getTotalUnreadFn: func(_ context.Context, id uuid.UUID) (int, error) {
			assert.Equal(t, userID, id)
			return 7, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	count, err := svc.GetTotalUnread(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, 7, count)
}

func TestGetTotalUnread_RepoError(t *testing.T) {
	msgRepo := &mockMessageRepo{
		getTotalUnreadFn: func(_ context.Context, _ uuid.UUID) (int, error) {
			return 0, fmt.Errorf("database error")
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	count, err := svc.GetTotalUnread(context.Background(), uuid.New())

	assert.Error(t, err)
	assert.Equal(t, 0, count)
}

// --- GetPresignedUploadURL tests ---

func TestGetPresignedUploadURL_Success(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil, nil)

	result, err := svc.GetPresignedUploadURL(context.Background(), GetPresignedURLInput{
		UserID:      uuid.New(),
		Filename:    "document.pdf",
		ContentType: "application/pdf",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result.UploadURL)
	assert.NotEmpty(t, result.FileKey)
	assert.NotEmpty(t, result.PublicURL)
}

// --- DeliverMessage tests ---

func TestDeliverMessage_Success(t *testing.T) {
	userID := uuid.New()
	msgID := uuid.New()
	convID := uuid.New()

	var statusUpdated bool
	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, id uuid.UUID) (*message.Message, error) {
			return &message.Message{
				ID:             id,
				ConversationID: convID,
				SenderID:       uuid.New(),
				Status:         message.MessageStatusSent,
			}, nil
		},
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		updateMessageStatusFn: func(_ context.Context, id uuid.UUID, status message.MessageStatus) error {
			statusUpdated = true
			assert.Equal(t, msgID, id)
			assert.Equal(t, message.MessageStatusDelivered, status)
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.DeliverMessage(context.Background(), msgID, userID)

	require.NoError(t, err)
	assert.True(t, statusUpdated)
}

func TestDeliverMessage_NotParticipant(t *testing.T) {
	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, id uuid.UUID) (*message.Message, error) {
			return &message.Message{
				ID:             id,
				ConversationID: uuid.New(),
				Status:         message.MessageStatusSent,
			}, nil
		},
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.DeliverMessage(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, message.ErrNotParticipant)
}

// --- StartConversation: existing conversation ---

func TestStartConversation_ExistingConversation(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	existingConvID := uuid.New()

	var createdMsg *message.Message
	msgRepo := &mockMessageRepo{
		findOrCreateConversationFn: func(_ context.Context, a, b uuid.UUID) (uuid.UUID, bool, error) {
			return existingConvID, false, nil // false = already existed
		},
		createMessageFn: func(_ context.Context, msg *message.Message) error {
			createdMsg = msg
			return nil
		},
		getParticipantIDsFn: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{senderID, recipientID}, nil
		},
	}
	userRepo := &mockUserRepo{
		getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
			return &user.User{ID: id}, nil
		},
	}

	svc := newTestService(msgRepo, userRepo, nil, nil, nil, nil)

	msg, returnedConvID, err := svc.StartConversation(context.Background(), StartConversationInput{
		SenderID:    senderID,
		RecipientID: recipientID,
		Content:     "Hey again!",
		Type:        message.MessageTypeText,
	})

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, existingConvID, returnedConvID)
	assert.Equal(t, "Hey again!", createdMsg.Content)
	assert.Equal(t, existingConvID, createdMsg.ConversationID)
}

// --- EditMessage: content too long ---

func TestEditMessage_ContentTooLong(t *testing.T) {
	ownerID := uuid.New()
	msgID := uuid.New()
	existingMsg := &message.Message{
		ID:        msgID,
		SenderID:  ownerID,
		Content:   "original",
		CreatedAt: time.Now(),
	}

	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, _ uuid.UUID) (*message.Message, error) {
			return existingMsg, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	longContent := make([]byte, message.MaxContentLength+1)
	for i := range longContent {
		longContent[i] = 'a'
	}

	result, err := svc.EditMessage(context.Background(), EditMessageInput{
		UserID:    ownerID,
		MessageID: msgID,
		Content:   string(longContent),
	})

	assert.ErrorIs(t, err, message.ErrContentTooLong)
	assert.Nil(t, result)
}

// --- DeliverMessage: already delivered (no-op) ---

func TestDeliverMessage_AlreadyDelivered(t *testing.T) {
	userID := uuid.New()
	msgID := uuid.New()
	convID := uuid.New()

	var statusUpdateCalled bool
	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, id uuid.UUID) (*message.Message, error) {
			return &message.Message{
				ID:             id,
				ConversationID: convID,
				SenderID:       uuid.New(),
				Status:         message.MessageStatusDelivered,
			}, nil
		},
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		updateMessageStatusFn: func(_ context.Context, _ uuid.UUID, _ message.MessageStatus) error {
			statusUpdateCalled = true
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.DeliverMessage(context.Background(), msgID, userID)

	require.NoError(t, err)
	assert.False(t, statusUpdateCalled, "should not update status when already delivered")
}

func TestDeliverMessage_AlreadyRead(t *testing.T) {
	userID := uuid.New()
	msgID := uuid.New()
	convID := uuid.New()

	var statusUpdateCalled bool
	msgRepo := &mockMessageRepo{
		getMessageFn: func(_ context.Context, id uuid.UUID) (*message.Message, error) {
			return &message.Message{
				ID:             id,
				ConversationID: convID,
				SenderID:       uuid.New(),
				Status:         message.MessageStatusRead,
			}, nil
		},
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		updateMessageStatusFn: func(_ context.Context, _ uuid.UUID, _ message.MessageStatus) error {
			statusUpdateCalled = true
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	err := svc.DeliverMessage(context.Background(), msgID, userID)

	require.NoError(t, err)
	assert.False(t, statusUpdateCalled, "should not update status when already read")
}

// --- GetMessagesSinceSeq tests ---

func TestGetMessagesSinceSeq_Success(t *testing.T) {
	userID := uuid.New()
	convID := uuid.New()

	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		getMessagesSinceSeqFn: func(_ context.Context, cID uuid.UUID, sinceSeq int, limit int) ([]*message.Message, error) {
			assert.Equal(t, convID, cID)
			assert.Equal(t, 10, sinceSeq)
			assert.Equal(t, 50, limit)
			return []*message.Message{
				{ID: uuid.New(), Content: "msg1", Seq: 11},
				{ID: uuid.New(), Content: "msg2", Seq: 12},
			}, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	msgs, err := svc.GetMessagesSinceSeq(context.Background(), userID, convID, 10)

	require.NoError(t, err)
	assert.Len(t, msgs, 2)
}

func TestGetMessagesSinceSeq_NotParticipant(t *testing.T) {
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	msgs, err := svc.GetMessagesSinceSeq(context.Background(), uuid.New(), uuid.New(), 5)

	assert.ErrorIs(t, err, message.ErrNotParticipant)
	assert.Nil(t, msgs)
}

// --- ListConversations: presence enrichment ---

func TestListConversations_PresenceEnrichment(t *testing.T) {
	userID := uuid.New()
	otherUserID1 := uuid.New()
	otherUserID2 := uuid.New()

	summaries := []repository.ConversationSummary{
		{ConversationID: uuid.New(), OtherUserID: otherUserID1, OtherUserName: "Alice"},
		{ConversationID: uuid.New(), OtherUserID: otherUserID2, OtherUserName: "Bob"},
	}

	msgRepo := &mockMessageRepo{
		listConversationsFn: func(_ context.Context, _ repository.ListConversationsParams) ([]repository.ConversationSummary, string, error) {
			return summaries, "", nil
		},
	}
	presence := &mockPresenceService{
		bulkIsOnlineFn: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{
				otherUserID1: true,
				otherUserID2: false,
			}, nil
		},
	}

	svc := newTestService(msgRepo, nil, presence, nil, nil, nil)

	result, _, err := svc.ListConversations(context.Background(), userID, "", 20)

	require.NoError(t, err)
	assert.Len(t, result, 2)
	assert.True(t, result[0].Online, "Alice should be online")
	assert.False(t, result[1].Online, "Bob should be offline")
}

// --- ListConversations: presence error is graceful ---

func TestListConversations_PresenceErrorGraceful(t *testing.T) {
	userID := uuid.New()

	summaries := []repository.ConversationSummary{
		{ConversationID: uuid.New(), OtherUserID: uuid.New(), OtherUserName: "Alice"},
	}

	msgRepo := &mockMessageRepo{
		listConversationsFn: func(_ context.Context, _ repository.ListConversationsParams) ([]repository.ConversationSummary, string, error) {
			return summaries, "", nil
		},
	}
	presence := &mockPresenceService{
		bulkIsOnlineFn: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]bool, error) {
			return nil, fmt.Errorf("redis connection error")
		},
	}

	svc := newTestService(msgRepo, nil, presence, nil, nil, nil)

	result, _, err := svc.ListConversations(context.Background(), userID, "", 20)

	require.NoError(t, err, "presence errors should not fail the request")
	assert.Len(t, result, 1)
	assert.False(t, result[0].Online, "should default to offline when presence fails")
}

// --- MarkAsRead: broadcasts read receipt ---

func TestMarkAsRead_BroadcastsReadReceipt(t *testing.T) {
	userID := uuid.New()
	convID := uuid.New()
	otherUserID := uuid.New()

	var broadcastCalled bool
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		getParticipantIDsFn: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{userID, otherUserID}, nil
		},
	}
	broadcaster := &mockBroadcaster{
		broadcastStatusUpdateFn: func(_ context.Context, ids []uuid.UUID, _ []byte) error {
			broadcastCalled = true
			assert.Len(t, ids, 1)
			assert.Equal(t, otherUserID, ids[0])
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, broadcaster, nil, nil)

	err := svc.MarkAsRead(context.Background(), MarkAsReadInput{
		UserID:         userID,
		ConversationID: convID,
		Seq:            42,
	})

	require.NoError(t, err)
	assert.True(t, broadcastCalled, "should broadcast read receipt to other participant")
}

// --- SendMessage: broadcasts to recipient with unread count ---

func TestSendMessage_BroadcastsUnreadCount(t *testing.T) {
	senderID := uuid.New()
	recipientID := uuid.New()
	convID := uuid.New()

	var unreadCountBroadcasted bool
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		getParticipantIDsFn: func(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
			return []uuid.UUID{senderID, recipientID}, nil
		},
		getTotalUnreadFn: func(_ context.Context, id uuid.UUID) (int, error) {
			if id == recipientID {
				return 5, nil
			}
			return 0, nil
		},
	}
	broadcaster := &mockBroadcaster{
		broadcastUnreadCountFn: func(_ context.Context, uid uuid.UUID, count int) error {
			if uid == recipientID {
				unreadCountBroadcasted = true
				assert.Equal(t, 5, count)
			}
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, broadcaster, nil, nil)

	msg, err := svc.SendMessage(context.Background(), SendMessageInput{
		SenderID:       senderID,
		ConversationID: convID,
		Content:        "Hello",
		Type:           message.MessageTypeText,
	})

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.True(t, unreadCountBroadcasted, "should broadcast unread count to recipient")
}

// --- GetPresignedUploadURL: key format ---

func TestGetPresignedUploadURL_KeyFormat(t *testing.T) {
	userID := uuid.New()
	var capturedKey string

	storage := &mockStorageService{
		getPresignedUploadFn: func(_ context.Context, key string, _ string, _ time.Duration) (string, error) {
			capturedKey = key
			return "https://presigned.example.com", nil
		},
		getPublicURLFn: func(key string) string {
			return "https://public.example.com/" + key
		},
	}

	svc := newTestService(nil, nil, nil, nil, storage, nil)

	result, err := svc.GetPresignedUploadURL(context.Background(), GetPresignedURLInput{
		UserID:      userID,
		Filename:    "document.pdf",
		ContentType: "application/pdf",
	})

	require.NoError(t, err)
	assert.Contains(t, capturedKey, "messaging/"+userID.String())
	// Filename is now randomized (UUID) but preserves the original extension
	assert.Contains(t, capturedKey, ".pdf")
	assert.NotContains(t, capturedKey, "document.pdf", "original filename should be replaced by UUID")
	assert.Contains(t, result.FileKey, "messaging/"+userID.String())
	assert.Contains(t, result.PublicURL, "messaging/"+userID.String())
}

func TestGetPresignedUploadURL_InvalidFileType(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil, nil)

	result, err := svc.GetPresignedUploadURL(context.Background(), GetPresignedURLInput{
		UserID:      uuid.New(),
		Filename:    "malware.exe",
		ContentType: "application/octet-stream",
	})

	assert.ErrorIs(t, err, message.ErrInvalidFileType)
	assert.Empty(t, result.UploadURL)
}

func TestGetPresignedUploadURL_AllowedTypes(t *testing.T) {
	svc := newTestService(nil, nil, nil, nil, nil, nil)

	allowedFiles := []string{"photo.jpg", "doc.pdf", "sheet.xlsx", "archive.zip", "notes.txt", "voice.m4a", "voice.webm"}
	for _, filename := range allowedFiles {
		result, err := svc.GetPresignedUploadURL(context.Background(), GetPresignedURLInput{
			UserID:      uuid.New(),
			Filename:    filename,
			ContentType: "application/octet-stream",
		})
		assert.NoError(t, err, "should allow %s", filename)
		assert.NotEmpty(t, result.UploadURL)
	}
}

// --- GetPresignedUploadURL: storage error ---

func TestGetPresignedUploadURL_StorageError(t *testing.T) {
	storage := &mockStorageService{
		getPresignedUploadFn: func(_ context.Context, _ string, _ string, _ time.Duration) (string, error) {
			return "", fmt.Errorf("storage unavailable")
		},
	}

	svc := newTestService(nil, nil, nil, nil, storage, nil)

	result, err := svc.GetPresignedUploadURL(context.Background(), GetPresignedURLInput{
		UserID:      uuid.New(),
		Filename:    "file.pdf",
		ContentType: "application/pdf",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "get presigned url")
	assert.Empty(t, result.UploadURL)
	assert.Empty(t, result.FileKey)
}

// --- SendMessage: file type with metadata ---

func TestSendMessage_FileType(t *testing.T) {
	senderID := uuid.New()
	convID := uuid.New()
	metadata := []byte(`{"url":"https://storage.example.com/file.pdf","filename":"file.pdf","size":1024,"mime_type":"application/pdf"}`)

	var createdMsg *message.Message
	msgRepo := &mockMessageRepo{
		isParticipantFn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return true, nil
		},
		createMessageFn: func(_ context.Context, msg *message.Message) error {
			createdMsg = msg
			return nil
		},
	}

	svc := newTestService(msgRepo, nil, nil, nil, nil, nil)

	msg, err := svc.SendMessage(context.Background(), SendMessageInput{
		SenderID:       senderID,
		ConversationID: convID,
		Content:        "file.pdf",
		Type:           message.MessageTypeFile,
		Metadata:       metadata,
	})

	require.NoError(t, err)
	require.NotNil(t, msg)
	assert.Equal(t, message.MessageTypeFile, createdMsg.Type)
	assert.Equal(t, "file.pdf", createdMsg.Content)
	assert.NotNil(t, createdMsg.Metadata)
}
