package notification

import (
	"context"

	"github.com/google/uuid"

	notif "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/domain/user"
	"marketplace-backend/internal/port/repository"
)

// --- mockNotificationRepo implements repository.NotificationRepository ---

type mockNotificationRepo struct {
	createFn           func(ctx context.Context, n *notif.Notification) error
	getByIDFn          func(ctx context.Context, id uuid.UUID) (*notif.Notification, error)
	listFn             func(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*notif.Notification, string, error)
	countUnreadFn      func(ctx context.Context, userID uuid.UUID) (int, error)
	markAsReadFn       func(ctx context.Context, id, userID uuid.UUID) error
	markAllAsReadFn    func(ctx context.Context, userID uuid.UUID) error
	deleteFn           func(ctx context.Context, id, userID uuid.UUID) error
	getPreferencesFn   func(ctx context.Context, userID uuid.UUID) ([]*notif.Preferences, error)
	upsertPreferenceFn func(ctx context.Context, pref *notif.Preferences) error
	createDeviceTokenFn func(ctx context.Context, dt *notif.DeviceToken) error
	listDeviceTokensFn  func(ctx context.Context, userID uuid.UUID) ([]*notif.DeviceToken, error)
	deleteDeviceTokenFn func(ctx context.Context, userID uuid.UUID, token string) error
}

func (m *mockNotificationRepo) Create(ctx context.Context, n *notif.Notification) error {
	if m.createFn != nil {
		return m.createFn(ctx, n)
	}
	return nil
}

func (m *mockNotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*notif.Notification, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, notif.ErrNotFound
}

func (m *mockNotificationRepo) List(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*notif.Notification, string, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userID, cursor, limit)
	}
	return nil, "", nil
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

func (m *mockNotificationRepo) GetPreferences(ctx context.Context, userID uuid.UUID) ([]*notif.Preferences, error) {
	if m.getPreferencesFn != nil {
		return m.getPreferencesFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockNotificationRepo) UpsertPreference(ctx context.Context, pref *notif.Preferences) error {
	if m.upsertPreferenceFn != nil {
		return m.upsertPreferenceFn(ctx, pref)
	}
	return nil
}

func (m *mockNotificationRepo) CreateDeviceToken(ctx context.Context, dt *notif.DeviceToken) error {
	if m.createDeviceTokenFn != nil {
		return m.createDeviceTokenFn(ctx, dt)
	}
	return nil
}

func (m *mockNotificationRepo) ListDeviceTokens(ctx context.Context, userID uuid.UUID) ([]*notif.DeviceToken, error) {
	if m.listDeviceTokensFn != nil {
		return m.listDeviceTokensFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockNotificationRepo) DeleteDeviceToken(ctx context.Context, userID uuid.UUID, token string) error {
	if m.deleteDeviceTokenFn != nil {
		return m.deleteDeviceTokenFn(ctx, userID, token)
	}
	return nil
}

// --- mockQueue implements NotificationQueue ---

type mockQueue struct {
	enqueueFn func(ctx context.Context, job DeliveryJob) error
	dequeueFn func(ctx context.Context) (*DeliveryJob, string, error)
	ackFn     func(ctx context.Context, messageID string) error
	jobs      []DeliveryJob // track enqueued jobs
}

func (m *mockQueue) Enqueue(ctx context.Context, job DeliveryJob) error {
	m.jobs = append(m.jobs, job)
	if m.enqueueFn != nil {
		return m.enqueueFn(ctx, job)
	}
	return nil
}

func (m *mockQueue) Dequeue(ctx context.Context) (*DeliveryJob, string, error) {
	if m.dequeueFn != nil {
		return m.dequeueFn(ctx)
	}
	return nil, "", nil
}

func (m *mockQueue) Ack(ctx context.Context, messageID string) error {
	if m.ackFn != nil {
		return m.ackFn(ctx, messageID)
	}
	return nil
}

// --- mockPresenceService implements service.PresenceService ---

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
	return nil, nil
}

// --- mockBroadcaster implements service.MessageBroadcaster (only BroadcastNotification used) ---

type mockBroadcaster struct {
	broadcastNotificationFn func(ctx context.Context, userID uuid.UUID, payload []byte) error
}

func (m *mockBroadcaster) BroadcastNewMessage(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}

func (m *mockBroadcaster) BroadcastTyping(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}

func (m *mockBroadcaster) BroadcastStatusUpdate(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}

func (m *mockBroadcaster) BroadcastUnreadCount(_ context.Context, _ uuid.UUID, _ int) error {
	return nil
}

func (m *mockBroadcaster) BroadcastPresence(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}

func (m *mockBroadcaster) BroadcastNotification(ctx context.Context, userID uuid.UUID, payload []byte) error {
	if m.broadcastNotificationFn != nil {
		return m.broadcastNotificationFn(ctx, userID, payload)
	}
	return nil
}

func (m *mockBroadcaster) BroadcastMessageEdited(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}

func (m *mockBroadcaster) BroadcastMessageDeleted(_ context.Context, _ []uuid.UUID, _ []byte) error {
	return nil
}

func (m *mockBroadcaster) BroadcastAccountSuspended(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}

// --- mockPushService implements service.PushService ---

type mockPushService struct {
	sendPushFn func(ctx context.Context, tokens []string, title, body string, data map[string]string) error
}

func (m *mockPushService) SendPush(ctx context.Context, tokens []string, title, body string, data map[string]string) error {
	if m.sendPushFn != nil {
		return m.sendPushFn(ctx, tokens, title, body, data)
	}
	return nil
}

// --- mockEmailService implements service.EmailService ---

type mockEmailService struct {
	sendPasswordResetFn func(ctx context.Context, to, resetURL string) error
	sendNotificationFn  func(ctx context.Context, to, subject, html string) error
}

func (m *mockEmailService) SendPasswordReset(ctx context.Context, to, resetURL string) error {
	if m.sendPasswordResetFn != nil {
		return m.sendPasswordResetFn(ctx, to, resetURL)
	}
	return nil
}

func (m *mockEmailService) SendNotification(ctx context.Context, to, subject, html string) error {
	if m.sendNotificationFn != nil {
		return m.sendNotificationFn(ctx, to, subject, html)
	}
	return nil
}

// --- mockUserRepo implements repository.UserRepository ---

type mockUserRepo struct {
	getByIDFn      func(ctx context.Context, id uuid.UUID) (*user.User, error)
	getByEmailFn   func(ctx context.Context, email string) (*user.User, error)
	createFn       func(ctx context.Context, u *user.User) error
	updateFn       func(ctx context.Context, u *user.User) error
	deleteFn       func(ctx context.Context, id uuid.UUID) error
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
	return nil, nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.getByEmailFn != nil {
		return m.getByEmailFn(ctx, email)
	}
	return nil, nil
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

func (m *mockUserRepo) ListAdmin(_ context.Context, _ repository.AdminUserFilters) ([]*user.User, string, error) {
	return nil, "", nil
}

func (m *mockUserRepo) CountAdmin(_ context.Context, _ repository.AdminUserFilters) (int, error) {
	return 0, nil
}

func (m *mockUserRepo) CountByRole(_ context.Context) (map[string]int, error) {
	return map[string]int{}, nil
}

func (m *mockUserRepo) CountByStatus(_ context.Context) (map[string]int, error) {
	return map[string]int{}, nil
}

func (m *mockUserRepo) RecentSignups(_ context.Context, _ int) ([]*user.User, error) {
	return nil, nil
}

// --- Stripe account stubs (migration 040) ---
func (m *mockUserRepo) GetStripeAccount(_ context.Context, _ uuid.UUID) (string, string, error) {
	return "", "", nil
}
func (m *mockUserRepo) FindUserIDByStripeAccount(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, nil
}
func (m *mockUserRepo) SetStripeAccount(_ context.Context, _ uuid.UUID, _, _ string) error {
	return nil
}
func (m *mockUserRepo) ClearStripeAccount(_ context.Context, _ uuid.UUID) error {
	return nil
}
func (m *mockUserRepo) GetStripeLastState(_ context.Context, _ uuid.UUID) ([]byte, error) {
	return nil, nil
}
func (m *mockUserRepo) SaveStripeLastState(_ context.Context, _ uuid.UUID, _ []byte) error {
	return nil
}
