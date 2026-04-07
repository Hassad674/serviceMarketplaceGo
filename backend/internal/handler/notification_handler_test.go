package handler

import (
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

	notifapp "marketplace-backend/internal/app/notification"
	notifdomain "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/handler/middleware"
)

// --- mock types (notification-specific, not in mocks_test.go) ---

type mockNotificationRepo struct {
	createFn            func(ctx context.Context, n *notifdomain.Notification) error
	getByIDFn           func(ctx context.Context, id uuid.UUID) (*notifdomain.Notification, error)
	listFn              func(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*notifdomain.Notification, string, error)
	countUnreadFn       func(ctx context.Context, userID uuid.UUID) (int, error)
	markAsReadFn        func(ctx context.Context, id, userID uuid.UUID) error
	markAllAsReadFn     func(ctx context.Context, userID uuid.UUID) error
	deleteFn            func(ctx context.Context, id, userID uuid.UUID) error
	getPreferencesFn    func(ctx context.Context, userID uuid.UUID) ([]*notifdomain.Preferences, error)
	upsertPreferenceFn  func(ctx context.Context, pref *notifdomain.Preferences) error
	createDeviceTokenFn func(ctx context.Context, dt *notifdomain.DeviceToken) error
	listDeviceTokensFn  func(ctx context.Context, userID uuid.UUID) ([]*notifdomain.DeviceToken, error)
	deleteDeviceTokenFn func(ctx context.Context, userID uuid.UUID, token string) error
}

func (m *mockNotificationRepo) Create(ctx context.Context, n *notifdomain.Notification) error {
	if m.createFn != nil {
		return m.createFn(ctx, n)
	}
	return nil
}

func (m *mockNotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*notifdomain.Notification, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, notifdomain.ErrNotFound
}

func (m *mockNotificationRepo) List(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*notifdomain.Notification, string, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, cursor, limit)
	}
	return []*notifdomain.Notification{}, "", nil
}

func (m *mockNotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	if m.countUnreadFn != nil {
		return m.countUnreadFn(ctx, userID)
	}
	return 0, nil
}

func (m *mockNotificationRepo) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	if m.markAsReadFn != nil {
		return m.markAsReadFn(ctx, id, userID)
	}
	return nil
}

func (m *mockNotificationRepo) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	if m.markAllAsReadFn != nil {
		return m.markAllAsReadFn(ctx, userID)
	}
	return nil
}

func (m *mockNotificationRepo) Delete(ctx context.Context, id, userID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, userID)
	}
	return nil
}

func (m *mockNotificationRepo) GetPreferences(ctx context.Context, userID uuid.UUID) ([]*notifdomain.Preferences, error) {
	if m.getPreferencesFn != nil {
		return m.getPreferencesFn(ctx, userID)
	}
	return []*notifdomain.Preferences{}, nil
}

func (m *mockNotificationRepo) UpsertPreference(ctx context.Context, pref *notifdomain.Preferences) error {
	if m.upsertPreferenceFn != nil {
		return m.upsertPreferenceFn(ctx, pref)
	}
	return nil
}

func (m *mockNotificationRepo) CreateDeviceToken(ctx context.Context, dt *notifdomain.DeviceToken) error {
	if m.createDeviceTokenFn != nil {
		return m.createDeviceTokenFn(ctx, dt)
	}
	return nil
}

func (m *mockNotificationRepo) ListDeviceTokens(ctx context.Context, userID uuid.UUID) ([]*notifdomain.DeviceToken, error) {
	if m.listDeviceTokensFn != nil {
		return m.listDeviceTokensFn(ctx, userID)
	}
	return []*notifdomain.DeviceToken{}, nil
}

func (m *mockNotificationRepo) DeleteDeviceToken(ctx context.Context, userID uuid.UUID, token string) error {
	if m.deleteDeviceTokenFn != nil {
		return m.deleteDeviceTokenFn(ctx, userID, token)
	}
	return nil
}

type mockPresenceService struct {
	isOnlineFn func(ctx context.Context, userID uuid.UUID) (bool, error)
}

func (m *mockPresenceService) SetOnline(_ context.Context, _ uuid.UUID) error  { return nil }
func (m *mockPresenceService) SetOffline(_ context.Context, _ uuid.UUID) error { return nil }
func (m *mockPresenceService) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	if m.isOnlineFn != nil {
		return m.isOnlineFn(ctx, userID)
	}
	return false, nil
}
func (m *mockPresenceService) BulkIsOnline(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]bool, error) {
	return map[uuid.UUID]bool{}, nil
}

type mockMessageBroadcaster struct{}

func (m *mockMessageBroadcaster) BroadcastNewMessage(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockMessageBroadcaster) BroadcastTyping(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockMessageBroadcaster) BroadcastStatusUpdate(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockMessageBroadcaster) BroadcastUnreadCount(_ context.Context, _ uuid.UUID, _ int) error {
	return nil
}
func (m *mockMessageBroadcaster) BroadcastPresence(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockMessageBroadcaster) BroadcastNotification(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockMessageBroadcaster) BroadcastMessageEdited(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockMessageBroadcaster) BroadcastMessageDeleted(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}
func (m *mockMessageBroadcaster) BroadcastAccountSuspended(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (m *mockMessageBroadcaster) BroadcastAdminNotification(_ context.Context, _ []uuid.UUID) error {
	return nil
}

type mockPushService struct{}

func (m *mockPushService) SendPush(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
	return nil
}

// --- helper ---

func newTestNotificationHandler(repo *mockNotificationRepo) *NotificationHandler {
	svc := notifapp.NewService(notifapp.ServiceDeps{
		Notifications: repo,
		Presence:      &mockPresenceService{},
		Broadcaster:   &mockMessageBroadcaster{},
		Push:          &mockPushService{},
		Email:         &mockEmailService{},
		Users:         &mockUserRepo{},
	})
	return NewNotificationHandler(svc)
}

func testNotification(userID uuid.UUID) *notifdomain.Notification {
	return &notifdomain.Notification{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      notifdomain.TypeProposalReceived,
		Title:     "New proposal",
		Body:      "You received a new proposal",
		Data:      json.RawMessage(`{}`),
		CreatedAt: time.Now(),
	}
}

// --- tests ---

func TestNotificationHandler_ListNotifications(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMock  func(*mockNotificationRepo)
		wantStatus int
		wantLen    int
	}{
		{
			name:   "success with results",
			userID: &uid,
			setupMock: func(r *mockNotificationRepo) {
				r.listFn = func(_ context.Context, _ uuid.UUID, _ string, _ int) ([]*notifdomain.Notification, string, error) {
					return []*notifdomain.Notification{testNotification(uid)}, "cursor_abc", nil
				}
			},
			wantStatus: http.StatusOK,
			wantLen:    1,
		},
		{
			name:       "success empty",
			userID:     &uid,
			wantStatus: http.StatusOK,
			wantLen:    0,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockNotificationRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestNotificationHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications", nil)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.ListNotifications(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantStatus == http.StatusOK {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				items := resp["data"].([]any)
				assert.Len(t, items, tc.wantLen)
			}
		})
	}
}

func TestNotificationHandler_GetUnreadCount(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		setupMock  func(*mockNotificationRepo)
		wantStatus int
		wantCount  float64
	}{
		{
			name:   "success",
			userID: &uid,
			setupMock: func(r *mockNotificationRepo) {
				r.countUnreadFn = func(_ context.Context, _ uuid.UUID) (int, error) {
					return 7, nil
				}
			},
			wantStatus: http.StatusOK,
			wantCount:  7,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockNotificationRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestNotificationHandler(repo)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/notifications/unread-count", nil)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.GetUnreadCount(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantStatus == http.StatusOK {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				data := resp["data"].(map[string]any)
				assert.Equal(t, tc.wantCount, data["count"])
			}
		})
	}
}

func TestNotificationHandler_MarkAsRead(t *testing.T) {
	uid := uuid.New()
	notifID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		urlParam   string
		setupMock  func(*mockNotificationRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "success",
			userID:     &uid,
			urlParam:   notifID.String(),
			wantStatus: http.StatusOK,
		},
		{
			name:     "not found",
			userID:   &uid,
			urlParam: notifID.String(),
			setupMock: func(r *mockNotificationRepo) {
				r.markAsReadFn = func(_ context.Context, _, _ uuid.UUID) error {
					return notifdomain.ErrNotFound
				}
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:     "not owner",
			userID:   &uid,
			urlParam: notifID.String(),
			setupMock: func(r *mockNotificationRepo) {
				r.markAsReadFn = func(_ context.Context, _, _ uuid.UUID) error {
					return notifdomain.ErrNotOwner
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "forbidden",
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			urlParam:   notifID.String(),
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockNotificationRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestNotificationHandler(repo)

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/"+tc.urlParam+"/read", nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.urlParam)
			ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			if tc.userID != nil {
				ctx = context.WithValue(ctx, middleware.ContextKeyUserID, *tc.userID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.MarkAsRead(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestNotificationHandler_MarkAllAsRead(t *testing.T) {
	uid := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		wantStatus int
		wantCode   string
	}{
		{
			name:       "success",
			userID:     &uid,
			wantStatus: http.StatusOK,
		},
		{
			name:       "unauthenticated",
			userID:     nil,
			wantStatus: http.StatusUnauthorized,
			wantCode:   "unauthorized",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestNotificationHandler(&mockNotificationRepo{})

			req := httptest.NewRequest(http.MethodPatch, "/api/v1/notifications/read-all", nil)
			if tc.userID != nil {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, *tc.userID)
				req = req.WithContext(ctx)
			}
			rec := httptest.NewRecorder()

			h.MarkAllAsRead(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}

func TestNotificationHandler_DeleteNotification(t *testing.T) {
	uid := uuid.New()
	notifID := uuid.New()

	tests := []struct {
		name       string
		userID     *uuid.UUID
		urlParam   string
		setupMock  func(*mockNotificationRepo)
		wantStatus int
		wantCode   string
	}{
		{
			name:       "success",
			userID:     &uid,
			urlParam:   notifID.String(),
			wantStatus: http.StatusOK,
		},
		{
			name:     "not found",
			userID:   &uid,
			urlParam: notifID.String(),
			setupMock: func(r *mockNotificationRepo) {
				r.deleteFn = func(_ context.Context, _, _ uuid.UUID) error {
					return notifdomain.ErrNotFound
				}
			},
			wantStatus: http.StatusNotFound,
			wantCode:   "not_found",
		},
		{
			name:     "not owner",
			userID:   &uid,
			urlParam: notifID.String(),
			setupMock: func(r *mockNotificationRepo) {
				r.deleteFn = func(_ context.Context, _, _ uuid.UUID) error {
					return notifdomain.ErrNotOwner
				}
			},
			wantStatus: http.StatusForbidden,
			wantCode:   "forbidden",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockNotificationRepo{}
			if tc.setupMock != nil {
				tc.setupMock(repo)
			}
			h := newTestNotificationHandler(repo)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/"+tc.urlParam, nil)
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tc.urlParam)
			ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			if tc.userID != nil {
				ctx = context.WithValue(ctx, middleware.ContextKeyUserID, *tc.userID)
			}
			req = req.WithContext(ctx)
			rec := httptest.NewRecorder()

			h.DeleteNotification(rec, req)
			assert.Equal(t, tc.wantStatus, rec.Code)

			if tc.wantCode != "" {
				var resp map[string]any
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
				assert.Equal(t, tc.wantCode, resp["error"])
			}
		})
	}
}
