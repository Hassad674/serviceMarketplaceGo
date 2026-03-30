package notification

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	notif "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/user"
)

func TestWorker_ProcessJob_PushAndEmail(t *testing.T) {
	userID := uuid.New()
	var pushCalled, emailCalled bool

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
				return false, nil // offline
			},
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				pushCalled = true
				return nil
			},
		},
		email: &mockEmailService{
			sendNotificationFn: func(_ context.Context, _, _, _ string) error {
				emailCalled = true
				return nil
			},
		},
		users: &mockUserRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return &user.User{Email: "test@test.com"}, nil
			},
		},
		notifs: &mockNotificationRepo{
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
				return []*notif.DeviceToken{{Token: "fcm-token-123"}}, nil
			},
		},
		queue: &mockQueue{},
	}

	err := w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeProposalReceived),
		Title:          "Test",
		Body:           "Test body",
	})

	assert.NoError(t, err)
	assert.True(t, pushCalled, "push should be called when user is offline")
	assert.True(t, emailCalled, "email should be called for proposal_received")
}

func TestWorker_ProcessJob_UserOnline_NoPush(t *testing.T) {
	userID := uuid.New()
	var pushCalled, emailCalled bool

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) {
				return true, nil // online
			},
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				pushCalled = true
				return nil
			},
		},
		email: &mockEmailService{
			sendNotificationFn: func(_ context.Context, _, _, _ string) error {
				emailCalled = true
				return nil
			},
		},
		users: &mockUserRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return &user.User{Email: "test@test.com"}, nil
			},
		},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					Push:             true,
					Email:            true,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "token"}}, nil
			},
		},
		queue: &mockQueue{},
	}

	err := w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeProposalReceived),
		Title:          "Test",
	})

	assert.NoError(t, err)
	assert.False(t, pushCalled, "push should NOT be called when user is online")
	assert.True(t, emailCalled, "email should still be called regardless of presence")
}

func TestWorker_ProcessJob_NewMessage_NoEmail(t *testing.T) {
	userID := uuid.New()
	var emailCalled bool

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil },
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error { return nil },
		},
		email: &mockEmailService{
			sendNotificationFn: func(_ context.Context, _, _, _ string) error {
				emailCalled = true
				return nil
			},
		},
		users: &mockUserRepo{
			getByIDFn: func(_ context.Context, _ uuid.UUID) (*user.User, error) {
				return &user.User{Email: "test@test.com"}, nil
			},
		},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeNewMessage,
					Push:             true,
					Email:            true,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "token"}}, nil
			},
		},
		queue: &mockQueue{},
	}

	_ = w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeNewMessage),
		Title:          "New msg",
	})

	assert.False(t, emailCalled, "email should NEVER be sent for new_message type")
}

func TestWorker_ProcessJob_PushFails_Retries(t *testing.T) {
	userID := uuid.New()
	q := &mockQueue{}

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil },
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				return errors.New("FCM timeout")
			},
		},
		email:  nil, // no email configured
		users:  &mockUserRepo{},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					Push:             true,
					Email:            false,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "token"}}, nil
			},
		},
		queue: q,
	}

	_ = w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeProposalReceived),
		Title:          "Test",
		Attempt:        0,
	})

	assert.Len(t, q.jobs, 1, "should re-enqueue failed job")
	assert.Equal(t, 1, q.jobs[0].Attempt, "attempt should be incremented")
}

func TestWorker_ProcessJob_MaxRetries_DeadLetter(t *testing.T) {
	userID := uuid.New()
	q := &mockQueue{}

	w := &Worker{
		presence: &mockPresenceService{
			isOnlineFn: func(_ context.Context, _ uuid.UUID) (bool, error) { return false, nil },
		},
		push: &mockPushService{
			sendPushFn: func(_ context.Context, _ []string, _, _ string, _ map[string]string) error {
				return errors.New("FCM down")
			},
		},
		email:  nil,
		users:  &mockUserRepo{},
		notifs: &mockNotificationRepo{
			getPreferencesFn: func(_ context.Context, _ uuid.UUID) ([]*notif.Preferences, error) {
				return []*notif.Preferences{{
					UserID:           userID,
					NotificationType: notif.TypeProposalReceived,
					Push:             true,
					Email:            false,
				}}, nil
			},
			listDeviceTokensFn: func(_ context.Context, _ uuid.UUID) ([]*notif.DeviceToken, error) {
				return []*notif.DeviceToken{{Token: "token"}}, nil
			},
		},
		queue: q,
	}

	_ = w.processJob(context.Background(), DeliveryJob{
		NotificationID: uuid.New().String(),
		UserID:         userID.String(),
		Type:           string(notif.TypeProposalReceived),
		Title:          "Test",
		Attempt:        2, // maxRetries-1 = 2
	})

	assert.Empty(t, q.jobs, "should NOT re-enqueue after max retries (dead letter)")
}
