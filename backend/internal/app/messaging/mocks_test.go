package messaging

import (
	"context"
	"io"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// --- mockMessageRepository ---

type mockMessageRepo struct {
	findOrCreateConversationFn func(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, bool, error)
	getConversationFn          func(ctx context.Context, id uuid.UUID) (*message.Conversation, error)
	listConversationsFn        func(ctx context.Context, params repository.ListConversationsParams) ([]repository.ConversationSummary, string, error)
	isParticipantFn            func(ctx context.Context, conversationID, userID uuid.UUID) (bool, error)
	createMessageFn            func(ctx context.Context, msg *message.Message) error
	getMessageFn               func(ctx context.Context, id uuid.UUID) (*message.Message, error)
	listMessagesFn             func(ctx context.Context, params repository.ListMessagesParams) ([]*message.Message, string, error)
	getMessagesSinceSeqFn      func(ctx context.Context, conversationID uuid.UUID, sinceSeq int, limit int) ([]*message.Message, error)
	updateMessageFn            func(ctx context.Context, msg *message.Message) error
	incrementUnreadFn          func(ctx context.Context, conversationID, senderID uuid.UUID) error
	markAsReadFn               func(ctx context.Context, conversationID, userID uuid.UUID, seq int) error
	getTotalUnreadFn           func(ctx context.Context, userID uuid.UUID) (int, error)
	getTotalUnreadBatchFn      func(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]int, error)
	getParticipantIDsFn        func(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error)
	updateMessageStatusFn      func(ctx context.Context, messageID uuid.UUID, status message.MessageStatus) error
	markMessagesAsReadFn       func(ctx context.Context, conversationID, readerID uuid.UUID, upToSeq int) error
	getContactIDsFn            func(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

func (m *mockMessageRepo) FindOrCreateConversation(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, bool, error) {
	if m.findOrCreateConversationFn != nil {
		return m.findOrCreateConversationFn(ctx, userA, userB)
	}
	return uuid.New(), true, nil
}

func (m *mockMessageRepo) GetConversation(ctx context.Context, id uuid.UUID) (*message.Conversation, error) {
	if m.getConversationFn != nil {
		return m.getConversationFn(ctx, id)
	}
	return nil, message.ErrConversationNotFound
}

func (m *mockMessageRepo) ListConversations(ctx context.Context, params repository.ListConversationsParams) ([]repository.ConversationSummary, string, error) {
	if m.listConversationsFn != nil {
		return m.listConversationsFn(ctx, params)
	}
	return nil, "", nil
}

func (m *mockMessageRepo) IsParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	if m.isParticipantFn != nil {
		return m.isParticipantFn(ctx, conversationID, userID)
	}
	return true, nil
}

func (m *mockMessageRepo) CreateMessage(ctx context.Context, msg *message.Message) error {
	if m.createMessageFn != nil {
		return m.createMessageFn(ctx, msg)
	}
	return nil
}

func (m *mockMessageRepo) GetMessage(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	if m.getMessageFn != nil {
		return m.getMessageFn(ctx, id)
	}
	return nil, message.ErrMessageNotFound
}

func (m *mockMessageRepo) ListMessages(ctx context.Context, params repository.ListMessagesParams) ([]*message.Message, string, error) {
	if m.listMessagesFn != nil {
		return m.listMessagesFn(ctx, params)
	}
	return nil, "", nil
}

func (m *mockMessageRepo) GetMessagesSinceSeq(ctx context.Context, conversationID uuid.UUID, sinceSeq int, limit int) ([]*message.Message, error) {
	if m.getMessagesSinceSeqFn != nil {
		return m.getMessagesSinceSeqFn(ctx, conversationID, sinceSeq, limit)
	}
	return nil, nil
}

func (m *mockMessageRepo) UpdateMessage(ctx context.Context, msg *message.Message) error {
	if m.updateMessageFn != nil {
		return m.updateMessageFn(ctx, msg)
	}
	return nil
}

func (m *mockMessageRepo) IncrementUnread(ctx context.Context, conversationID, senderID uuid.UUID) error {
	if m.incrementUnreadFn != nil {
		return m.incrementUnreadFn(ctx, conversationID, senderID)
	}
	return nil
}

func (m *mockMessageRepo) MarkAsRead(ctx context.Context, conversationID, userID uuid.UUID, seq int) error {
	if m.markAsReadFn != nil {
		return m.markAsReadFn(ctx, conversationID, userID, seq)
	}
	return nil
}

func (m *mockMessageRepo) GetTotalUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.getTotalUnreadFn != nil {
		return m.getTotalUnreadFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockMessageRepo) GetTotalUnreadBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	if m.getTotalUnreadBatchFn != nil {
		return m.getTotalUnreadBatchFn(ctx, userIDs)
	}
	// Fall back to individual calls for backward compat with existing tests
	result := make(map[uuid.UUID]int, len(userIDs))
	for _, uid := range userIDs {
		count, err := m.GetTotalUnread(ctx, uid)
		if err != nil {
			return nil, err
		}
		result[uid] = count
	}
	return result, nil
}

func (m *mockMessageRepo) GetParticipantIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	if m.getParticipantIDsFn != nil {
		return m.getParticipantIDsFn(ctx, conversationID)
	}
	return nil, nil
}

func (m *mockMessageRepo) UpdateMessageStatus(ctx context.Context, messageID uuid.UUID, status message.MessageStatus) error {
	if m.updateMessageStatusFn != nil {
		return m.updateMessageStatusFn(ctx, messageID, status)
	}
	return nil
}

func (m *mockMessageRepo) MarkMessagesAsRead(ctx context.Context, conversationID, readerID uuid.UUID, upToSeq int) error {
	if m.markMessagesAsReadFn != nil {
		return m.markMessagesAsReadFn(ctx, conversationID, readerID, upToSeq)
	}
	return nil
}

func (m *mockMessageRepo) GetContactIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if m.getContactIDsFn != nil {
		return m.getContactIDsFn(ctx, userID)
	}
	return nil, nil
}

// --- mockUserRepo ---

type mockUserRepo struct {
	createFn        func(ctx context.Context, u *user.User) error
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*user.User, error)
	getByEmailFn    func(ctx context.Context, email string) (*user.User, error)
	updateFn        func(ctx context.Context, u *user.User) error
	deleteFn        func(ctx context.Context, id uuid.UUID) error
	existsByEmailFn func(ctx context.Context, email string) (bool, error)
}

func (m *mockUserRepo) Create(ctx context.Context, u *user.User) error {
	if m.createFn != nil {
		return m.createFn(ctx, u)
	}
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return &user.User{ID: id}, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, user.ErrUserNotFound
}

func (m *mockUserRepo) Update(ctx context.Context, u *user.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, u)
	}
	return nil
}

func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockUserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.existsByEmailFn != nil {
		return m.existsByEmailFn(ctx, email)
	}
	return false, nil
}

// --- mockPresenceService ---

type mockPresenceService struct {
	setOnlineFn    func(ctx context.Context, userID uuid.UUID) error
	setOfflineFn   func(ctx context.Context, userID uuid.UUID) error
	isOnlineFn     func(ctx context.Context, userID uuid.UUID) (bool, error)
	bulkIsOnlineFn func(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

func (m *mockPresenceService) SetOnline(ctx context.Context, userID uuid.UUID) error {
	if m.setOnlineFn != nil {
		return m.setOnlineFn(ctx, userID)
	}
	return nil
}

func (m *mockPresenceService) SetOffline(ctx context.Context, userID uuid.UUID) error {
	if m.setOfflineFn != nil {
		return m.setOfflineFn(ctx, userID)
	}
	return nil
}

func (m *mockPresenceService) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	if m.isOnlineFn != nil {
		return m.isOnlineFn(ctx, userID)
	}
	return false, nil
}

func (m *mockPresenceService) BulkIsOnline(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	if m.bulkIsOnlineFn != nil {
		return m.bulkIsOnlineFn(ctx, userIDs)
	}
	return make(map[uuid.UUID]bool), nil
}

// --- mockBroadcaster ---

type mockBroadcaster struct {
	broadcastNewMessageFn    func(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error
	broadcastTypingFn        func(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error
	broadcastStatusUpdateFn  func(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error
	broadcastUnreadCountFn   func(ctx context.Context, userID uuid.UUID, count int) error
	broadcastPresenceFn      func(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error
}

func (m *mockBroadcaster) BroadcastNewMessage(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	if m.broadcastNewMessageFn != nil {
		return m.broadcastNewMessageFn(ctx, recipientIDs, payload)
	}
	return nil
}

func (m *mockBroadcaster) BroadcastTyping(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	if m.broadcastTypingFn != nil {
		return m.broadcastTypingFn(ctx, recipientIDs, payload)
	}
	return nil
}

func (m *mockBroadcaster) BroadcastStatusUpdate(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	if m.broadcastStatusUpdateFn != nil {
		return m.broadcastStatusUpdateFn(ctx, recipientIDs, payload)
	}
	return nil
}

func (m *mockBroadcaster) BroadcastUnreadCount(ctx context.Context, userID uuid.UUID, count int) error {
	if m.broadcastUnreadCountFn != nil {
		return m.broadcastUnreadCountFn(ctx, userID, count)
	}
	return nil
}

func (m *mockBroadcaster) BroadcastPresence(ctx context.Context, recipientIDs []uuid.UUID, payload []byte) error {
	if m.broadcastPresenceFn != nil {
		return m.broadcastPresenceFn(ctx, recipientIDs, payload)
	}
	return nil
}

// --- mockStorageService ---

type mockStorageService struct {
	uploadFn              func(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error)
	deleteFn              func(ctx context.Context, key string) error
	getPublicURLFn        func(key string) string
	getPresignedUploadFn  func(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error)
}

func (m *mockStorageService) Upload(ctx context.Context, key string, reader io.Reader, contentType string, size int64) (string, error) {
	if m.uploadFn != nil {
		return m.uploadFn(ctx, key, reader, contentType, size)
	}
	return "https://storage.example.com/" + key, nil
}

func (m *mockStorageService) Delete(ctx context.Context, key string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, key)
	}
	return nil
}

func (m *mockStorageService) GetPublicURL(key string) string {
	if m.getPublicURLFn != nil {
		return m.getPublicURLFn(key)
	}
	return "https://storage.example.com/" + key
}

func (m *mockStorageService) GetPresignedUploadURL(ctx context.Context, key string, contentType string, expiry time.Duration) (string, error) {
	if m.getPresignedUploadFn != nil {
		return m.getPresignedUploadFn(ctx, key, contentType, expiry)
	}
	return "https://storage.example.com/presigned/" + key, nil
}

// --- mockRateLimiter ---

type mockRateLimiter struct {
	allowFn func(ctx context.Context, userID uuid.UUID) (bool, error)
}

func (m *mockRateLimiter) Allow(ctx context.Context, userID uuid.UUID) (bool, error) {
	if m.allowFn != nil {
		return m.allowFn(ctx, userID)
	}
	return true, nil
}
