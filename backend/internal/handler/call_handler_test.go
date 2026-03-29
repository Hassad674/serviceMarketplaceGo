package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	callapp "marketplace-backend/internal/app/call"
	calldomain "marketplace-backend/internal/domain/call"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/handler/middleware"
	"marketplace-backend/internal/port/service"
)

// ---------------------------------------------------------------------------
// Mocks specific to call tests
// ---------------------------------------------------------------------------

type mockLiveKitService struct {
	createRoomFn    func(ctx context.Context, roomName string) error
	generateTokenFn func(roomName, identity, displayName string) (string, error)
	deleteRoomFn    func(ctx context.Context, roomName string) error
}

func (m *mockLiveKitService) CreateRoom(ctx context.Context, roomName string) error {
	if m.createRoomFn != nil {
		return m.createRoomFn(ctx, roomName)
	}
	return nil
}

func (m *mockLiveKitService) GenerateToken(roomName, identity, displayName string) (string, error) {
	if m.generateTokenFn != nil {
		return m.generateTokenFn(roomName, identity, displayName)
	}
	return "test-token", nil
}

func (m *mockLiveKitService) DeleteRoom(ctx context.Context, roomName string) error {
	if m.deleteRoomFn != nil {
		return m.deleteRoomFn(ctx, roomName)
	}
	return nil
}

var _ service.LiveKitService = (*mockLiveKitService)(nil)

type mockCallStateService struct {
	saveActiveCallFn      func(ctx context.Context, c *calldomain.Call) error
	getActiveCallFn       func(ctx context.Context, callID uuid.UUID) (*calldomain.Call, error)
	getActiveCallByUserFn func(ctx context.Context, userID uuid.UUID) (*calldomain.Call, error)
	removeActiveCallFn    func(ctx context.Context, callID uuid.UUID) error
}

func (m *mockCallStateService) SaveActiveCall(ctx context.Context, c *calldomain.Call) error {
	if m.saveActiveCallFn != nil {
		return m.saveActiveCallFn(ctx, c)
	}
	return nil
}

func (m *mockCallStateService) GetActiveCall(ctx context.Context, callID uuid.UUID) (*calldomain.Call, error) {
	if m.getActiveCallFn != nil {
		return m.getActiveCallFn(ctx, callID)
	}
	return nil, calldomain.ErrCallNotFound
}

func (m *mockCallStateService) GetActiveCallByUser(ctx context.Context, userID uuid.UUID) (*calldomain.Call, error) {
	if m.getActiveCallByUserFn != nil {
		return m.getActiveCallByUserFn(ctx, userID)
	}
	return nil, calldomain.ErrCallNotFound
}

func (m *mockCallStateService) RemoveActiveCall(ctx context.Context, callID uuid.UUID) error {
	if m.removeActiveCallFn != nil {
		return m.removeActiveCallFn(ctx, callID)
	}
	return nil
}

var _ service.CallStateService = (*mockCallStateService)(nil)

type mockCallBroadcaster struct{}

func (m *mockCallBroadcaster) BroadcastCallEvent(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}

var _ service.CallBroadcaster = (*mockCallBroadcaster)(nil)

type mockMessageSender struct {
	sendSystemMessageFn func(ctx context.Context, input service.SystemMessageInput) error
}

func (m *mockMessageSender) SendSystemMessage(ctx context.Context, input service.SystemMessageInput) error {
	if m.sendSystemMessageFn != nil {
		return m.sendSystemMessageFn(ctx, input)
	}
	return nil
}

var _ service.MessageSender = (*mockMessageSender)(nil)

// ---------------------------------------------------------------------------
// Test helper
// ---------------------------------------------------------------------------

func newTestCallHandler(
	livekit *mockLiveKitService,
	callState *mockCallStateService,
) *CallHandler {
	if livekit == nil {
		livekit = &mockLiveKitService{}
	}
	if callState == nil {
		callState = &mockCallStateService{}
	}
	svc := callapp.NewService(callapp.ServiceDeps{
		LiveKit:     livekit,
		CallState:   callState,
		Presence:    &mockPresenceService{isOnlineFn: defaultOnline},
		Broadcaster: &mockCallBroadcaster{},
		Messages:    &mockMessageSender{},
		Users:       &mockUserRepo{getByIDFn: defaultGetUser},
	})
	return NewCallHandler(svc)
}

func defaultOnline(_ context.Context, _ uuid.UUID) (bool, error) { return true, nil }

func defaultGetUser(_ context.Context, id uuid.UUID) (*user.User, error) {
	return &user.User{ID: id, DisplayName: "Test User"}, nil
}

func callAuthCtx(req *http.Request, userID uuid.UUID) *http.Request {
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, userID)
	return req.WithContext(ctx)
}

func callChiCtx(req *http.Request, userID uuid.UUID, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	ctx = context.WithValue(ctx, middleware.ContextKeyUserID, userID)
	return req.WithContext(ctx)
}

func encodeJSON(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	require.NoError(t, json.NewEncoder(buf).Encode(v))
	return buf
}

// ---------------------------------------------------------------------------
// InitiateCall
// ---------------------------------------------------------------------------

func TestCallHandler_InitiateCall(t *testing.T) {
	initiatorID := uuid.New()
	recipientID := uuid.New()
	convID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		body       map[string]any
		setupState func(*mockCallStateService)
		setupPres  func() *mockPresenceService
		wantStatus int
		wantCode   string
	}{
		{
			name:   "success",
			userID: &initiatorID,
			body: map[string]any{
				"conversation_id": convID.String(),
				"recipient_id":    recipientID.String(),
				"type":            "audio",
			},
			wantStatus: http.StatusCreated,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			body:       map[string]any{},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:   "recipient offline",
			userID: &initiatorID,
			body: map[string]any{
				"conversation_id": convID.String(),
				"recipient_id":    recipientID.String(),
				"type":            "video",
			},
			setupPres: func() *mockPresenceService {
				return &mockPresenceService{
					isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
						return false, nil
					},
				}
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "recipient_offline",
		},
		{
			name:   "invalid call type",
			userID: &initiatorID,
			body: map[string]any{
				"conversation_id": convID.String(),
				"recipient_id":    recipientID.String(),
				"type":            "invalid",
			},
			wantStatus: http.StatusBadRequest,
			wantCode:   "invalid_call_type",
		},
		{
			name:   "user busy",
			userID: &initiatorID,
			body: map[string]any{
				"conversation_id": convID.String(),
				"recipient_id":    recipientID.String(),
				"type":            "audio",
			},
			setupState: func(cs *mockCallStateService) {
				cs.getActiveCallByUserFn = func(_ context.Context, _ uuid.UUID) (*calldomain.Call, error) {
					return &calldomain.Call{}, nil
				}
			},
			wantStatus: http.StatusConflict,
			wantCode:   "user_busy",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cs := &mockCallStateService{}
			if tc.setupState != nil {
				tc.setupState(cs)
			}

			var h *CallHandler
			if tc.setupPres != nil {
				pres := tc.setupPres()
				svc := callapp.NewService(callapp.ServiceDeps{
					LiveKit:     &mockLiveKitService{},
					CallState:   cs,
					Presence:    pres,
					Broadcaster: &mockCallBroadcaster{},
					Messages:    &mockMessageSender{},
					Users:       &mockUserRepo{getByIDFn: defaultGetUser},
				})
				h = NewCallHandler(svc)
			} else {
				h = newTestCallHandler(nil, cs)
			}

			body := encodeJSON(t, tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/calls", body)
			req.Header.Set("Content-Type", "application/json")
			if tc.userID != nil {
				req = callAuthCtx(req, *tc.userID)
			}

			rec := httptest.NewRecorder()
			h.InitiateCall(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				assertErrorCode(t, rec, tc.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// AcceptCall
// ---------------------------------------------------------------------------

func TestCallHandler_AcceptCall(t *testing.T) {
	initiatorID := uuid.New()
	recipientID := uuid.New()
	callID := uuid.New()

	ringingCall := &calldomain.Call{
		ID:          callID,
		InitiatorID: initiatorID,
		RecipientID: recipientID,
		RoomName:    "call:" + callID.String(),
		Status:      calldomain.StatusRinging,
		Type:        calldomain.TypeAudio,
	}

	tests := []struct {
		name       string
		userID     *uuid.UUID
		paramID    string
		setupState func(*mockCallStateService)
		wantStatus int
		wantCode   string
	}{
		{
			name:    "success",
			userID:  &recipientID,
			paramID: callID.String(),
			setupState: func(cs *mockCallStateService) {
				cs.getActiveCallFn = func(_ context.Context, _ uuid.UUID) (*calldomain.Call, error) {
					cp := *ringingCall
					return &cp, nil
				}
			},
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			paramID:    callID.String(),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:    "not participant",
			userID:  ptrUUID(uuid.New()),
			paramID: callID.String(),
			setupState: func(cs *mockCallStateService) {
				cs.getActiveCallFn = func(_ context.Context, _ uuid.UUID) (*calldomain.Call, error) {
					cp := *ringingCall
					return &cp, nil
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "not_participant",
		},
		{
			name:       "call not found",
			userID:     &recipientID,
			paramID:    uuid.New().String(),
			wantStatus: http.StatusNotFound,
			wantCode:   "call_not_found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cs := &mockCallStateService{}
			if tc.setupState != nil {
				tc.setupState(cs)
			}
			h := newTestCallHandler(nil, cs)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/calls/"+tc.paramID+"/accept", nil)
			if tc.userID != nil {
				req = callChiCtx(req, *tc.userID, "id", tc.paramID)
			} else {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", tc.paramID)
				ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.AcceptCall(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				assertErrorCode(t, rec, tc.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// DeclineCall
// ---------------------------------------------------------------------------

func TestCallHandler_DeclineCall(t *testing.T) {
	initiatorID := uuid.New()
	recipientID := uuid.New()
	callID := uuid.New()

	ringingCall := &calldomain.Call{
		ID:          callID,
		InitiatorID: initiatorID,
		RecipientID: recipientID,
		RoomName:    "call:" + callID.String(),
		Status:      calldomain.StatusRinging,
		Type:        calldomain.TypeAudio,
	}

	tests := []struct {
		name       string
		userID     *uuid.UUID
		paramID    string
		setupState func(*mockCallStateService)
		wantStatus int
		wantCode   string
	}{
		{
			name:    "success as recipient",
			userID:  &recipientID,
			paramID: callID.String(),
			setupState: func(cs *mockCallStateService) {
				cs.getActiveCallFn = func(_ context.Context, _ uuid.UUID) (*calldomain.Call, error) {
					cp := *ringingCall
					return &cp, nil
				}
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:    "success as initiator",
			userID:  &initiatorID,
			paramID: callID.String(),
			setupState: func(cs *mockCallStateService) {
				cs.getActiveCallFn = func(_ context.Context, _ uuid.UUID) (*calldomain.Call, error) {
					cp := *ringingCall
					return &cp, nil
				}
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			paramID:    callID.String(),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cs := &mockCallStateService{}
			if tc.setupState != nil {
				tc.setupState(cs)
			}
			h := newTestCallHandler(nil, cs)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/calls/"+tc.paramID+"/decline", nil)
			if tc.userID != nil {
				req = callChiCtx(req, *tc.userID, "id", tc.paramID)
			} else {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", tc.paramID)
				ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.DeclineCall(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				assertErrorCode(t, rec, tc.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// EndCall
// ---------------------------------------------------------------------------

func TestCallHandler_EndCall(t *testing.T) {
	initiatorID := uuid.New()
	recipientID := uuid.New()
	callID := uuid.New()

	activeCall := &calldomain.Call{
		ID:          callID,
		InitiatorID: initiatorID,
		RecipientID: recipientID,
		RoomName:    "call:" + callID.String(),
		Status:      calldomain.StatusActive,
		Type:        calldomain.TypeVideo,
	}

	tests := []struct {
		name       string
		userID     *uuid.UUID
		paramID    string
		body       map[string]any
		setupState func(*mockCallStateService)
		wantStatus int
		wantCode   string
	}{
		{
			name:    "success",
			userID:  &initiatorID,
			paramID: callID.String(),
			body:    map[string]any{"duration": 120},
			setupState: func(cs *mockCallStateService) {
				cs.getActiveCallFn = func(_ context.Context, _ uuid.UUID) (*calldomain.Call, error) {
					cp := *activeCall
					return &cp, nil
				}
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			paramID:    callID.String(),
			body:       map[string]any{"duration": 60},
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
		{
			name:    "not participant",
			userID:  ptrUUID(uuid.New()),
			paramID: callID.String(),
			body:    map[string]any{"duration": 60},
			setupState: func(cs *mockCallStateService) {
				cs.getActiveCallFn = func(_ context.Context, _ uuid.UUID) (*calldomain.Call, error) {
					cp := *activeCall
					return &cp, nil
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "not_participant",
		},
		{
			name:       "call not found",
			userID:     &initiatorID,
			paramID:    uuid.New().String(),
			body:       map[string]any{"duration": 60},
			wantStatus: http.StatusNotFound,
			wantCode:   "call_not_found",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cs := &mockCallStateService{}
			if tc.setupState != nil {
				tc.setupState(cs)
			}
			h := newTestCallHandler(nil, cs)

			body := encodeJSON(t, tc.body)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/calls/"+tc.paramID+"/end", body)
			req.Header.Set("Content-Type", "application/json")
			if tc.userID != nil {
				req = callChiCtx(req, *tc.userID, "id", tc.paramID)
			} else {
				rctx := chi.NewRouteContext()
				rctx.URLParams.Add("id", tc.paramID)
				ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
				req = req.WithContext(ctx)
			}

			rec := httptest.NewRecorder()
			h.EndCall(rec, req)

			assert.Equal(t, tc.wantStatus, rec.Code)
			if tc.wantCode != "" {
				assertErrorCode(t, rec, tc.wantCode)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func ptrUUID(id uuid.UUID) *uuid.UUID { return &id }

func assertErrorCode(t *testing.T, rec *httptest.ResponseRecorder, wantCode string) {
	t.Helper()
	if wantCode == "" {
		return
	}
	var resp map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, wantCode, resp["error"])
}
