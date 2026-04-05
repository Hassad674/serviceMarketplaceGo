package notification

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	notif "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/service"
)

func TestService_Send_Success(t *testing.T) {
	userID := uuid.New()
	var persisted bool
	var broadcasted bool

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			createFn: func(_ context.Context, n *notif.Notification) error {
				persisted = true
				assert.Equal(t, userID, n.UserID)
				assert.Equal(t, notif.TypeProposalReceived, n.Type)
				return nil
			},
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return nil, nil // use defaults
			},
		},
		Presence: &mockPresenceService{},
		Broadcaster: &mockBroadcaster{
			broadcastNotificationFn: func(_ context.Context, id uuid.UUID, _ []byte) error {
				broadcasted = true
				assert.Equal(t, userID, id)
				return nil
			},
		},
	})

	err := svc.Send(context.Background(), service.NotificationInput{
		UserID: userID,
		Type:   "proposal_received",
		Title:  "New Proposal",
		Body:   "You received a new proposal",
	})

	assert.NoError(t, err)
	assert.True(t, persisted, "notification must be persisted")
	assert.True(t, broadcasted, "notification must be broadcast in-app")
}

func TestService_Send_InvalidType(t *testing.T) {
	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{},
		Presence:      &mockPresenceService{},
		Broadcaster:   &mockBroadcaster{},
	})

	err := svc.Send(context.Background(), service.NotificationInput{
		UserID: uuid.New(),
		Type:   "totally_invalid_type",
		Title:  "Test",
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, notif.ErrInvalidType)
}

func TestService_List(t *testing.T) {
	userID := uuid.New()
	expected := []*notif.Notification{
		{ID: uuid.New(), UserID: userID, Type: notif.TypeNewMessage, Title: "Hello"},
		{ID: uuid.New(), UserID: userID, Type: notif.TypeReviewReceived, Title: "Review"},
	}

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			listFn: func(_ context.Context, uid uuid.UUID, cursor string, limit int) ([]*notif.Notification, string, error) {
				assert.Equal(t, userID, uid)
				assert.Equal(t, "", cursor)
				assert.Equal(t, 20, limit)
				return expected, "next123", nil
			},
		},
		Presence:    &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
	})

	result, nextCursor, err := svc.List(context.Background(), userID, "", 20)

	assert.NoError(t, err)
	assert.Len(t, result, 2)
	assert.Equal(t, "next123", nextCursor)
}

func TestService_GetUnreadCount(t *testing.T) {
	userID := uuid.New()

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			countUnreadFn: func(_ context.Context, uid uuid.UUID) (int, error) {
				assert.Equal(t, userID, uid)
				return 5, nil
			},
		},
		Presence:    &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
	})

	count, err := svc.GetUnreadCount(context.Background(), userID)

	assert.NoError(t, err)
	assert.Equal(t, 5, count)
}

func TestService_MarkAsRead_Success(t *testing.T) {
	notifID := uuid.New()
	userID := uuid.New()
	var called bool

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			markAsReadFn: func(_ context.Context, id, uid uuid.UUID) error {
				called = true
				assert.Equal(t, notifID, id)
				assert.Equal(t, userID, uid)
				return nil
			},
		},
		Presence:    &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
	})

	err := svc.MarkAsRead(context.Background(), notifID, userID)

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestService_MarkAsRead_NotFound(t *testing.T) {
	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			markAsReadFn: func(_ context.Context, _, _ uuid.UUID) error {
				return notif.ErrNotFound
			},
		},
		Presence:    &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
	})

	err := svc.MarkAsRead(context.Background(), uuid.New(), uuid.New())

	assert.ErrorIs(t, err, notif.ErrNotFound)
}

func TestService_MarkAllAsRead(t *testing.T) {
	userID := uuid.New()
	var called bool

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			markAllAsReadFn: func(_ context.Context, uid uuid.UUID) error {
				called = true
				assert.Equal(t, userID, uid)
				return nil
			},
		},
		Presence:    &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
	})

	err := svc.MarkAllAsRead(context.Background(), userID)

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestService_Delete_Success(t *testing.T) {
	notifID := uuid.New()
	userID := uuid.New()
	var called bool

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			deleteFn: func(_ context.Context, id, uid uuid.UUID) error {
				called = true
				assert.Equal(t, notifID, id)
				assert.Equal(t, userID, uid)
				return nil
			},
		},
		Presence:    &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
	})

	err := svc.Delete(context.Background(), notifID, userID)

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestService_GetPreferences_MergesDefaults(t *testing.T) {
	userID := uuid.New()

	// Only one preference saved; the rest should come from defaults
	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{
					{
						UserID:           userID,
						NotificationType: notif.TypeNewMessage,
						InApp:            false,
						Push:             false,
						Email:            false,
					},
				}, nil
			},
		},
		Presence:    &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
	})

	prefs, err := svc.GetPreferences(context.Background(), userID)

	assert.NoError(t, err)
	assert.Len(t, prefs, 12, "should return preferences for all 12 notification types")

	// Find the new_message preference — it should be the saved one (all false)
	for _, p := range prefs {
		if p.NotificationType == notif.TypeNewMessage {
			assert.False(t, p.InApp)
			assert.False(t, p.Push)
			assert.False(t, p.Email)
		}
		// proposal_received should be a default (InApp=true, Push=true, Email=true)
		if p.NotificationType == notif.TypeProposalReceived {
			assert.True(t, p.InApp)
			assert.True(t, p.Push)
			assert.True(t, p.Email) // default for proposal_received
		}
	}
}

func TestService_RegisterDevice(t *testing.T) {
	userID := uuid.New()
	var called bool

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			createDeviceTokenFn: func(_ context.Context, dt *notif.DeviceToken) error {
				called = true
				assert.Equal(t, userID, dt.UserID)
				assert.Equal(t, "fcm-token-123", dt.Token)
				assert.Equal(t, "android", dt.Platform)
				return nil
			},
		},
		Presence:    &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
	})

	err := svc.RegisterDevice(context.Background(), userID, "fcm-token-123", "android")

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestService_Send_WithEmailChannel(t *testing.T) {
	userID := uuid.New()
	var emailSent bool

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			createFn: func(_ context.Context, _ *notif.Notification) error {
				return nil
			},
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{
					{
						UserID:           userID,
						NotificationType: notif.TypeProposalAccepted,
						InApp:            true,
						Push:             false,
						Email:            true,
					},
				}, nil
			},
		},
		Presence:    &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
		Email: &mockEmailService{
			sendNotificationFn: func(_ context.Context, to, subject, html string) error {
				emailSent = true
				assert.Equal(t, "user@example.com", to)
				assert.Contains(t, subject, "Proposal Accepted")
				return nil
			},
		},
		Users: &mockUserRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
				return &user.User{ID: id, Email: "user@example.com"}, nil
			},
		},
	})

	err := svc.Send(context.Background(), service.NotificationInput{
		UserID: userID,
		Type:   "proposal_accepted",
		Title:  "Proposal Accepted",
		Body:   "Your proposal has been accepted",
		Data:   json.RawMessage(`{"proposal_id":"abc"}`),
	})

	assert.NoError(t, err)
	assert.True(t, emailSent, "email should be sent for proposal_accepted with email preference on")
}

func TestService_Send_NoEmailForNewMessage(t *testing.T) {
	userID := uuid.New()
	var emailSent bool

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			createFn: func(_ context.Context, _ *notif.Notification) error {
				return nil
			},
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{
					{
						UserID:           userID,
						NotificationType: notif.TypeNewMessage,
						InApp:            true,
						Push:             true,
						Email:            true, // even with email=true, new_message should NOT send email
					},
				}, nil
			},
		},
		Presence:    &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
		Email: &mockEmailService{
			sendNotificationFn: func(_ context.Context, _, _, _ string) error {
				emailSent = true
				return nil
			},
		},
		Users: &mockUserRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
				return &user.User{ID: id, Email: "user@example.com"}, nil
			},
		},
	})

	err := svc.Send(context.Background(), service.NotificationInput{
		UserID: userID,
		Type:   "new_message",
		Title:  "New Message",
		Body:   "You have a new message",
	})

	assert.NoError(t, err)
	assert.False(t, emailSent, "email must never be sent for new_message type")
}

func TestService_Send_WithQueue_Enqueues(t *testing.T) {
	userID := uuid.New()
	q := &mockQueue{}
	var pushCalled bool

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			createFn: func(_ context.Context, _ *notif.Notification) error {
				return nil
			},
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return nil, nil // defaults
			},
		},
		Presence: &mockPresenceService{},
		Broadcaster: &mockBroadcaster{},
		Push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				pushCalled = true
				return nil
			},
		},
		Email: &mockEmailService{
			sendNotificationFn: func(_ context.Context, _, _, _ string) error {
				t.Error("email should not be called directly when queue is set")
				return nil
			},
		},
		Queue: q,
	})

	err := svc.Send(context.Background(), service.NotificationInput{
		UserID: userID,
		Type:   "proposal_received",
		Title:  "New Proposal",
		Body:   "You received a new proposal",
		Data:   json.RawMessage(`{"proposal_id":"abc"}`),
	})

	assert.NoError(t, err)
	assert.Len(t, q.jobs, 1, "should enqueue exactly one delivery job")
	assert.Equal(t, userID.String(), q.jobs[0].UserID)
	assert.Equal(t, "proposal_received", q.jobs[0].Type)
	assert.Equal(t, "New Proposal", q.jobs[0].Title)
	assert.Equal(t, 0, q.jobs[0].Attempt)
	assert.False(t, pushCalled, "push should NOT be called directly when queue is set")
}

func TestService_Send_WithQueue_StillBroadcastsInApp(t *testing.T) {
	userID := uuid.New()
	q := &mockQueue{}
	var broadcasted bool

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			createFn: func(_ context.Context, _ *notif.Notification) error {
				return nil
			},
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return nil, nil
			},
		},
		Presence: &mockPresenceService{},
		Broadcaster: &mockBroadcaster{
			broadcastNotificationFn: func(_ context.Context, id uuid.UUID, _ []byte) error {
				broadcasted = true
				assert.Equal(t, userID, id)
				return nil
			},
		},
		Queue: q,
	})

	err := svc.Send(context.Background(), service.NotificationInput{
		UserID: userID,
		Type:   "proposal_received",
		Title:  "Test",
		Body:   "Test body",
	})

	assert.NoError(t, err)
	assert.True(t, broadcasted, "WS broadcast must happen synchronously even with queue")
	assert.Len(t, q.jobs, 1, "delivery job must be enqueued")
}

func TestService_Send_QueueFallback(t *testing.T) {
	userID := uuid.New()
	var pushCalled, emailCalled bool

	q := &mockQueue{
		enqueueFn: func(_ context.Context, _ DeliveryJob) error {
			return errors.New("redis down")
		},
	}

	svc := NewService(ServiceDeps{
		Notifications: &mockNotificationRepo{
			createFn: func(_ context.Context, _ *notif.Notification) error {
				return nil
			},
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					InApp:            true,
					Push:             true,
					Email:            true,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "fcm-token"}}, nil
			},
		},
		Presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
				return false, nil
			},
		},
		Broadcaster: &mockBroadcaster{},
		Push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				pushCalled = true
				return nil
			},
		},
		Email: &mockEmailService{
			sendNotificationFn: func(_ context.Context, _, _, _ string) error {
				emailCalled = true
				return nil
			},
		},
		Users: &mockUserRepo{
			getByIDFn: func(_ context.Context, id uuid.UUID) (*user.User, error) {
				return &user.User{ID: id, Email: "user@example.com"}, nil
			},
		},
		Queue: q,
	})

	err := svc.Send(context.Background(), service.NotificationInput{
		UserID: userID,
		Type:   "proposal_received",
		Title:  "New Proposal",
		Body:   "Proposal body",
	})

	assert.NoError(t, err)
	assert.True(t, pushCalled, "push should be called as sync fallback when enqueue fails")
	assert.True(t, emailCalled, "email should be called as sync fallback when enqueue fails")
}
