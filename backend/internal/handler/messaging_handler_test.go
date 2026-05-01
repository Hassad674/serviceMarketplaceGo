package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"marketplace-backend/internal/app/messaging"
	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/domain/organization"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/repository"
)

// --- mock types for messaging tests ---

type mockMessageRepo struct {
	findOrCreateConversationFn     func(ctx context.Context, a, b, senderOrgID, senderUserID uuid.UUID) (uuid.UUID, bool, error)
	getConversationFn              func(ctx context.Context, id uuid.UUID) (*message.Conversation, error)
	listConversationsFn            func(ctx context.Context, p repository.ListConversationsParams) ([]repository.ConversationSummary, string, error)
	isParticipantFn                func(ctx context.Context, convID, userID uuid.UUID) (bool, error)
	isOrgAuthorizedFn              func(ctx context.Context, convID, orgID uuid.UUID) (bool, error)
	createMessageFn                func(ctx context.Context, m *message.Message, senderOrgID, senderUserID uuid.UUID) error
	getMessageFn                   func(ctx context.Context, id uuid.UUID) (*message.Message, error)
	listMessagesFn                 func(ctx context.Context, p repository.ListMessagesParams) ([]*message.Message, string, error)
	getMessagesSinceSeqFn          func(ctx context.Context, convID uuid.UUID, seq, limit int) ([]*message.Message, error)
	updateMessageFn                func(ctx context.Context, m *message.Message) error
	incrementUnreadForRecipientsFn func(ctx context.Context, convID, senderUserID, senderOrgID uuid.UUID) error
	markAsReadFn                   func(ctx context.Context, convID, userID uuid.UUID, seq int) error
	getTotalUnreadFn               func(ctx context.Context, userID uuid.UUID) (int, error)
	getTotalUnreadBatchFn          func(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error)
	getParticipantIDsFn            func(ctx context.Context, convID uuid.UUID) ([]uuid.UUID, error)
	getOrgMemberRecipientsFn       func(ctx context.Context, convID, excludeUserID uuid.UUID) ([]uuid.UUID, error)
	updateMessageStatusFn          func(ctx context.Context, id uuid.UUID, s message.MessageStatus) error
	markMessagesAsReadFn           func(ctx context.Context, convID, readerID uuid.UUID, seq int) error
	getContactIDsFn                func(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
}

func (m *mockMessageRepo) FindOrCreateConversation(ctx context.Context, a, b, senderOrgID, senderUserID uuid.UUID) (uuid.UUID, bool, error) {
	if m.findOrCreateConversationFn != nil {
		return m.findOrCreateConversationFn(ctx, a, b, senderOrgID, senderUserID)
	}
	return uuid.New(), true, nil
}
func (m *mockMessageRepo) GetConversation(ctx context.Context, id uuid.UUID) (*message.Conversation, error) {
	if m.getConversationFn != nil {
		return m.getConversationFn(ctx, id)
	}
	return nil, message.ErrConversationNotFound
}
func (m *mockMessageRepo) ListConversations(ctx context.Context, p repository.ListConversationsParams) ([]repository.ConversationSummary, string, error) {
	if m.listConversationsFn != nil {
		return m.listConversationsFn(ctx, p)
	}
	return []repository.ConversationSummary{}, "", nil
}
func (m *mockMessageRepo) IsParticipant(ctx context.Context, convID, userID uuid.UUID) (bool, error) {
	if m.isParticipantFn != nil {
		return m.isParticipantFn(ctx, convID, userID)
	}
	return true, nil
}
func (m *mockMessageRepo) IsOrgAuthorizedForConversation(ctx context.Context, convID, orgID uuid.UUID) (bool, error) {
	if m.isOrgAuthorizedFn != nil {
		return m.isOrgAuthorizedFn(ctx, convID, orgID)
	}
	return true, nil
}
func (m *mockMessageRepo) CreateMessage(ctx context.Context, msg *message.Message, senderOrgID, senderUserID uuid.UUID) error {
	if m.createMessageFn != nil {
		return m.createMessageFn(ctx, msg, senderOrgID, senderUserID)
	}
	return nil
}
func (m *mockMessageRepo) GetMessage(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	if m.getMessageFn != nil {
		return m.getMessageFn(ctx, id)
	}
	return nil, message.ErrMessageNotFound
}
func (m *mockMessageRepo) ListMessages(ctx context.Context, p repository.ListMessagesParams) ([]*message.Message, string, error) {
	if m.listMessagesFn != nil {
		return m.listMessagesFn(ctx, p)
	}
	return []*message.Message{}, "", nil
}
func (m *mockMessageRepo) GetMessagesSinceSeq(ctx context.Context, convID uuid.UUID, seq, limit int) ([]*message.Message, error) {
	if m.getMessagesSinceSeqFn != nil {
		return m.getMessagesSinceSeqFn(ctx, convID, seq, limit)
	}
	return []*message.Message{}, nil
}
func (m *mockMessageRepo) ListMessagesSinceTime(_ context.Context, _ uuid.UUID, _ time.Time, _ int) ([]*message.Message, error) {
	return []*message.Message{}, nil
}
func (m *mockMessageRepo) UpdateMessage(ctx context.Context, msg *message.Message) error {
	if m.updateMessageFn != nil {
		return m.updateMessageFn(ctx, msg)
	}
	return nil
}
func (m *mockMessageRepo) IncrementUnreadForRecipients(ctx context.Context, convID, senderUserID, senderOrgID uuid.UUID) error {
	if m.incrementUnreadForRecipientsFn != nil {
		return m.incrementUnreadForRecipientsFn(ctx, convID, senderUserID, senderOrgID)
	}
	return nil
}
func (m *mockMessageRepo) MarkAsRead(ctx context.Context, convID, userID uuid.UUID, seq int) error {
	if m.markAsReadFn != nil {
		return m.markAsReadFn(ctx, convID, userID, seq)
	}
	return nil
}
func (m *mockMessageRepo) GetTotalUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.getTotalUnreadFn != nil {
		return m.getTotalUnreadFn(ctx, userID)
	}
	return 0, nil
}
func (m *mockMessageRepo) GetTotalUnreadBatch(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	if m.getTotalUnreadBatchFn != nil {
		return m.getTotalUnreadBatchFn(ctx, ids)
	}
	return map[uuid.UUID]int{}, nil
}
func (m *mockMessageRepo) GetParticipantIDs(ctx context.Context, convID uuid.UUID) ([]uuid.UUID, error) {
	if m.getParticipantIDsFn != nil {
		return m.getParticipantIDsFn(ctx, convID)
	}
	return []uuid.UUID{}, nil
}
func (m *mockMessageRepo) GetOrgMemberRecipients(ctx context.Context, convID, excludeUserID uuid.UUID) ([]uuid.UUID, error) {
	if m.getOrgMemberRecipientsFn != nil {
		return m.getOrgMemberRecipientsFn(ctx, convID, excludeUserID)
	}
	return []uuid.UUID{}, nil
}
func (m *mockMessageRepo) UpdateMessageStatus(ctx context.Context, id uuid.UUID, s message.MessageStatus) error {
	if m.updateMessageStatusFn != nil {
		return m.updateMessageStatusFn(ctx, id, s)
	}
	return nil
}
func (m *mockMessageRepo) MarkMessagesAsRead(ctx context.Context, convID, readerID uuid.UUID, seq int) error {
	if m.markMessagesAsReadFn != nil {
		return m.markMessagesAsReadFn(ctx, convID, readerID, seq)
	}
	return nil
}
func (m *mockMessageRepo) GetContactIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	if m.getContactIDsFn != nil {
		return m.getContactIDsFn(ctx, userID)
	}
	return []uuid.UUID{}, nil
}

func (m *mockMessageRepo) SaveMessageHistory(_ context.Context, _, _ uuid.UUID, _, _ string) error {
	return nil
}

// mockPresenceService and mockMessageBroadcaster are defined in
// notification_handler_test.go and shared across this package's tests.

type mockMessagingRateLimiter struct {
	allowFn func(ctx context.Context, userID uuid.UUID) (bool, error)
}

func (m *mockMessagingRateLimiter) Allow(ctx context.Context, userID uuid.UUID) (bool, error) {
	if m.allowFn != nil {
		return m.allowFn(ctx, userID)
	}
	return true, nil
}

// --- helper ---

func newTestMessagingHandler(
	msgRepo *mockMessageRepo,
	userRepo *mockUserRepo,
	orgRepo *mockOrgRepo,
) *MessagingHandler {
	if orgRepo == nil {
		orgRepo = &mockOrgRepo{}
	}
	svc := messaging.NewService(messaging.ServiceDeps{
		Messages:      msgRepo,
		Users:         userRepo,
		Organizations: orgRepo,
		Presence:      &mockPresenceService{},
		Broadcaster:   &mockMessageBroadcaster{},
		Storage:       &mockStorageService{},
		RateLimiter:   &mockMessagingRateLimiter{},
	})
	return NewMessagingHandler(svc)
}

func authCtx(req *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
	// Use the same UUID for the org id so tests that exercise org-
	// scoped endpoints (ListConversations, ListMyJobs, …) pass the
	// middleware check without needing a separate org fixture.
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, userID)
	return req.WithContext(ctx)
}

func chiCtx(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}

func chiAuthCtx(req *http.Request, userID uuid.UUID, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)
	// Inject a synthetic organization_id so org-scoped handlers
	// (ListMessages, SendMessage, MarkAsRead, …) pass the middleware
	// check in tests that use this helper. Mirrors authCtx.
	ctx = context.WithValue(ctx, middleware.ContextKeyOrganizationID, userID)
	return req.WithContext(ctx)
}

// --- tests ---

func TestMessagingHandler_StartConversation(t *testing.T) {
	uid := uuid.New()
	recipientUserID := uuid.New()
	recipientOrgID := uuid.New()
	convID := uuid.New()

	// Org resolution: the test bodies send a recipient_org_id, and the
	// service looks up the owner user id via OrganizationRepository.
	// These hooks make that mapping explicit.
	successOrgRepo := func() *mockOrgRepo {
		return &mockOrgRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{ID: recipientOrgID, OwnerUserID: recipientUserID}, nil
			},
		}
	}
	selfOrgRepo := func() *mockOrgRepo {
		// A Provider's personal org is owned by themselves, so messaging
		// their own org resolves to the caller's user id and trips the
		// self-conversation guard.
		return &mockOrgRepo{
			findByIDFn: func(_ context.Context, _ uuid.UUID) (*organization.Organization, error) {
				return &organization.Organization{ID: uid, OwnerUserID: uid}, nil
			},
		}
	}

	tests := []struct {
		name       string
		userID     *uuid.UUID
		body       map[string]string
		orgRepo    *mockOrgRepo
		setupMocks func(*mockUserRepo, *mockMessageRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:    "success",
			userID:  &uid,
			body:    map[string]string{"recipient_org_id": recipientOrgID.String(), "content": "hello"},
			orgRepo: successOrgRepo(),
			setupMocks: func(ur *mockUserRepo, mr *mockMessageRepo) {
				ur.getByIDFn = func(_ context.Context, _ uuid.UUID) (*user.User, error) {
					return testUser(recipientUserID, user.RoleProvider), nil
				}
				mr.findOrCreateConversationFn = func(_ context.Context, _, _, _, _ uuid.UUID) (uuid.UUID, bool, error) {
					return convID, true, nil
				}
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "self conversation",
			userID:     &uid,
			body:       map[string]string{"recipient_org_id": uid.String(), "content": "hi"},
			orgRepo:    selfOrgRepo(),
			wantStatus: http.StatusBadRequest,
			wantCode:   "self_conversation",
		},
		{
			name:       "unauthenticated",
			body:       map[string]string{"recipient_org_id": recipientOrgID.String(), "content": "hi"},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgRepo := &mockMessageRepo{}
			userRepo := &mockUserRepo{}
			if tc.setupMocks != nil {
				tc.setupMocks(userRepo, msgRepo)
			}
			h := newTestMessagingHandler(msgRepo, userRepo, tc.orgRepo)

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.userID != nil {
				req = authCtx(req, *tc.userID)
			}
			rec := httptest.NewRecorder()

			h.StartConversation(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestMessagingHandler_ListConversations(t *testing.T) {
	uid := uuid.New()
	otherID := uuid.New()
	now := time.Now()
	lastMsg := "hey"

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMock  func(*mockMessageRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:   "success",
			userID: &uid,
			setupMock: func(mr *mockMessageRepo) {
				mr.listConversationsFn = func(_ context.Context, _ repository.ListConversationsParams) ([]repository.ConversationSummary, string, error) {
					return []repository.ConversationSummary{{
						ConversationID: uuid.New(),
						OtherOrgID:    otherID,
						OtherOrgName:  "Alice",
						LastMessage:    &lastMsg,
						LastMessageAt:  &now,
						UnreadCount:    2,
					}}, "", nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgRepo := &mockMessageRepo{}
			if tc.setupMock != nil {
				tc.setupMock(msgRepo)
			}
			h := newTestMessagingHandler(msgRepo, &mockUserRepo{}, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
			if tc.userID != nil {
				req = authCtx(req, *tc.userID)
			}
			rec := httptest.NewRecorder()

			h.ListConversations(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestMessagingHandler_SendMessage(t *testing.T) {
	uid := uuid.New()
	convID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		convParam  string
		body       map[string]string
		setupMock  func(*mockMessageRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:      "success",
			userID:    &uid,
			convParam: convID.String(),
			body:      map[string]string{"content": "hello"},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "empty content",
			userID:     &uid,
			convParam:  convID.String(),
			body:       map[string]string{"content": ""},
			wantStatus: http.StatusBadRequest,
			wantCode:   "empty_content",
		},
		{
			name:      "not participant",
			userID:    &uid,
			convParam: convID.String(),
			body:      map[string]string{"content": "hello"},
			setupMock: func(mr *mockMessageRepo) {
				mr.isOrgAuthorizedFn = func(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
					return false, nil
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "not_participant",
		},
		{
			name:       "unauthenticated",
			convParam:  convID.String(),
			body:       map[string]string{"content": "hi"},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgRepo := &mockMessageRepo{}
			if tc.setupMock != nil {
				tc.setupMock(msgRepo)
			}
			h := newTestMessagingHandler(msgRepo, &mockUserRepo{}, nil)

			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/"+tc.convParam+"/messages", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			if tc.userID != nil {
				req = chiAuthCtx(req, *tc.userID, "id", tc.convParam)
			} else {
				req = chiCtx(req, "id", tc.convParam)
			}
			rec := httptest.NewRecorder()

			h.SendMessage(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestMessagingHandler_ListMessages(t *testing.T) {
	uid := uuid.New()
	convID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		convParam  string
		setupMock  func(*mockMessageRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:      "success",
			userID:    &uid,
			convParam: convID.String(),
			setupMock: func(mr *mockMessageRepo) {
				mr.listMessagesFn = func(_ context.Context, _ repository.ListMessagesParams) ([]*message.Message, string, error) {
					return []*message.Message{{
						ID: uuid.New(), ConversationID: convID, SenderID: uid,
						Content: "hi", Type: message.MessageTypeText,
						Status: message.MessageStatusSent, CreatedAt: time.Now(),
					}}, "", nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:      "not participant",
			userID:    &uid,
			convParam: convID.String(),
			setupMock: func(mr *mockMessageRepo) {
				mr.isOrgAuthorizedFn = func(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
					return false, nil
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "not_participant",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgRepo := &mockMessageRepo{}
			if tc.setupMock != nil {
				tc.setupMock(msgRepo)
			}
			h := newTestMessagingHandler(msgRepo, &mockUserRepo{}, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+tc.convParam+"/messages", nil)
			req = chiAuthCtx(req, *tc.userID, "id", tc.convParam)
			rec := httptest.NewRecorder()

			h.ListMessages(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestMessagingHandler_MarkAsRead(t *testing.T) {
	uid := uuid.New()
	convID := uuid.New()

	h := newTestMessagingHandler(&mockMessageRepo{}, &mockUserRepo{}, nil)

	body, _ := json.Marshal(map[string]int{"seq": 5})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/"+convID.String()+"/read", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = chiAuthCtx(req, uid, "id", convID.String())
	rec := httptest.NewRecorder()

	h.MarkAsRead(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "ok", resp["status"])
}

func TestMessagingHandler_GetTotalUnread(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMock  func(*mockMessageRepo)
		wantStatus int
		wantCount  int
		wantCode   string
	}{
		{
			name:   "success",
			userID: &uid,
			setupMock: func(mr *mockMessageRepo) {
				mr.getTotalUnreadFn = func(_ context.Context, _ uuid.UUID) (int, error) {
					return 7, nil
				}
			},
			wantStatus: http.StatusOK,
			wantCount:  7,
		},
		{
			name:       "unauthenticated",
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			msgRepo := &mockMessageRepo{}
			if tc.setupMock != nil {
				tc.setupMock(msgRepo)
			}
			h := newTestMessagingHandler(msgRepo, &mockUserRepo{}, nil)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/unread", nil)
			if tc.userID != nil {
				req = authCtx(req, *tc.userID)
			}
			rec := httptest.NewRecorder()

			h.GetTotalUnread(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
			if tc.wantStatus == http.StatusOK {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, float64(tc.wantCount), resp["count"])
			}
		})
	}
}
func (m *mockMessageRepo) UpdateMessageModeration(_ context.Context, _ uuid.UUID, _ string, _ float64, _ []byte) error {
	return nil
}

// TestMessagingHandler_ListMessages_OrgOperator exercises the R11 fix
// end-to-end at the HTTP boundary: an operator whose user id is NOT
// in conversation_participants (Bob joined the org after the
// conversation existed) should receive 200 and the messages — the
// old user-level IsParticipant guard would have returned 403.
func TestMessagingHandler_ListMessages_OrgOperator(t *testing.T) {
	bobUserID := uuid.New()
	convID := uuid.New()
	aliceUserID := uuid.New()

	msgRepo := &mockMessageRepo{
		// Org-level check passes: Bob's org has Alice as a direct
		// participant, so any operator of that org is authorized.
		isOrgAuthorizedFn: func(_ context.Context, c, _ uuid.UUID) (bool, error) {
			return c == convID, nil
		},
		// The old user-level check would have rejected Bob — the
		// handler must NOT consult it for the authorization path.
		isParticipantFn: func(_ context.Context, _, userID uuid.UUID) (bool, error) {
			return userID == aliceUserID, nil
		},
		listMessagesFn: func(_ context.Context, _ repository.ListMessagesParams) ([]*message.Message, string, error) {
			return []*message.Message{{
				ID:             uuid.New(),
				ConversationID: convID,
				SenderID:       aliceUserID,
				Content:        "original message from Alice",
				Type:           message.MessageTypeText,
				Status:         message.MessageStatusSent,
				CreatedAt:      time.Now(),
			}}, "", nil
		},
	}

	h := newTestMessagingHandler(msgRepo, &mockUserRepo{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/"+convID.String()+"/messages", nil)
	req = chiAuthCtx(req, bobUserID, "id", convID.String())
	rec := httptest.NewRecorder()

	h.ListMessages(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "Bob (org operator, not direct participant) must read Alice's conversation")

	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	dataArr, ok := resp["data"].([]any)
	require.True(t, ok, "response should contain a data array")
	require.Len(t, dataArr, 1, "should return the single seeded message")
}
