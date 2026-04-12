package notification

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	notif "marketplace-backend/internal/domain/notification"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/internal/port/service"
)

// ServiceDeps groups dependencies for the notification service.
type ServiceDeps struct {
	Notifications repository.NotificationRepository
	Presence      service.PresenceService
	Broadcaster   service.MessageBroadcaster
	Push          service.PushService  // nil if FCM not configured
	Email         service.EmailService // nil if email not configured
	Users         repository.UserRepository
	Queue         NotificationQueue // nil for synchronous fallback
}

// Service implements the notification use cases.
type Service struct {
	notifications repository.NotificationRepository
	presence      service.PresenceService
	broadcaster   service.MessageBroadcaster
	push          service.PushService
	email         service.EmailService
	users         repository.UserRepository
	queue         NotificationQueue
}

// NewService creates a new notification Service.
func NewService(deps ServiceDeps) *Service {
	return &Service{
		notifications: deps.Notifications,
		presence:      deps.Presence,
		broadcaster:   deps.Broadcaster,
		push:          deps.Push,
		email:         deps.Email,
		users:         deps.Users,
		queue:         deps.Queue,
	}
}

// Send creates and dispatches a notification across all enabled channels.
// This implements the NotificationSender interface.
func (s *Service) Send(ctx context.Context, input service.NotificationInput) error {
	n, err := notif.NewNotification(notif.NewNotificationInput{
		UserID: input.UserID,
		Type:   notif.NotificationType(input.Type),
		Title:  input.Title,
		Body:   input.Body,
		Data:   input.Data,
	})
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}

	// 1. Always persist to database
	if err := s.notifications.Create(ctx, n); err != nil {
		return fmt.Errorf("persist notification: %w", err)
	}

	// 2. Load preferences (use defaults if no row exists)
	prefs := s.getPreferencesForType(ctx, n.UserID, n.Type)

	// 3. In-app channel: broadcast via WebSocket (always synchronous)
	if prefs.InApp {
		s.broadcastInApp(ctx, n)
	}

	// 4. Async delivery: push + email via worker queue
	if s.queue != nil {
		s.enqueueDelivery(ctx, n)
	} else {
		// Fallback: synchronous delivery (no queue configured)
		if prefs.Push {
			s.sendPushIfOffline(ctx, n)
		}
		if prefs.Email && n.Type != notif.TypeNewMessage {
			// Respect the global email kill-switch
			if s.users != nil {
				if u, err := s.users.GetByID(ctx, n.UserID); err == nil && !u.EmailNotificationsEnabled {
					slog.Debug("email skipped (globally disabled)", "user_id", n.UserID)
				} else {
					s.sendEmail(ctx, n)
				}
			} else {
				s.sendEmail(ctx, n)
			}
		}
	}

	return nil
}

// List returns paginated notifications for a user.
func (s *Service) List(ctx context.Context, userID uuid.UUID, cursor string, limit int) ([]*notif.Notification, string, error) {
	return s.notifications.List(ctx, userID, cursor, limit)
}

// GetUnreadCount returns the number of unread notifications for a user.
func (s *Service) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.notifications.CountUnread(ctx, userID)
}

// MarkAsRead marks a single notification as read.
func (s *Service) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	return s.notifications.MarkAsRead(ctx, id, userID)
}

// MarkAllAsRead marks all notifications as read for a user.
func (s *Service) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return s.notifications.MarkAllAsRead(ctx, userID)
}

// Delete removes a notification.
func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.notifications.Delete(ctx, id, userID)
}

// GetPreferences returns the notification preferences for a user, merged with defaults.
func (s *Service) GetPreferences(ctx context.Context, userID uuid.UUID) ([]*notif.Preferences, error) {
	saved, err := s.notifications.GetPreferences(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get preferences: %w", err)
	}

	// Build a map of saved preferences
	savedMap := make(map[notif.NotificationType]*notif.Preferences, len(saved))
	for _, p := range saved {
		savedMap[p.NotificationType] = p
	}

	// Merge with defaults for ALL valid notification types
	allTypes := notif.AllTypes()

	result := make([]*notif.Preferences, 0, len(allTypes))
	for _, t := range allTypes {
		if p, ok := savedMap[t]; ok {
			result = append(result, p)
		} else {
			result = append(result, notif.DefaultPreferences(userID, t))
		}
	}
	return result, nil
}

// UpdatePreferences upserts notification preferences.
func (s *Service) UpdatePreferences(ctx context.Context, userID uuid.UUID, prefs []*notif.Preferences) error {
	for _, p := range prefs {
		p.UserID = userID
		if !p.NotificationType.IsValid() {
			continue
		}
		if err := s.notifications.UpsertPreference(ctx, p); err != nil {
			return fmt.Errorf("upsert preference: %w", err)
		}
	}
	return nil
}

// SetAllEmailPreferences enables or disables email notifications globally
// for a user by toggling the email_notifications_enabled column on the
// users table. This is a kill-switch: when false, no email is sent
// regardless of per-type preferences.
func (s *Service) SetAllEmailPreferences(ctx context.Context, userID uuid.UUID, enabled bool) error {
	return s.users.UpdateEmailNotificationsEnabled(ctx, userID, enabled)
}

// GetEmailNotificationsEnabled returns the global email notification
// kill-switch state for a user.
func (s *Service) GetEmailNotificationsEnabled(ctx context.Context, userID uuid.UUID) (bool, error) {
	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return true, fmt.Errorf("get user for email enabled: %w", err)
	}
	return u.EmailNotificationsEnabled, nil
}

// RegisterDevice registers a device token for push notifications.
func (s *Service) RegisterDevice(ctx context.Context, userID uuid.UUID, token, platform string) error {
	dt := &notif.DeviceToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		Platform:  platform,
		CreatedAt: time.Now(),
	}
	return s.notifications.CreateDeviceToken(ctx, dt)
}

// UnregisterDevice removes a device token.
func (s *Service) UnregisterDevice(ctx context.Context, userID uuid.UUID, token string) error {
	return s.notifications.DeleteDeviceToken(ctx, userID, token)
}

// --- Private dispatch helpers ---

func (s *Service) enqueueDelivery(ctx context.Context, n *notif.Notification) {
	job := DeliveryJob{
		NotificationID: n.ID.String(),
		UserID:         n.UserID.String(),
		Type:           string(n.Type),
		Title:          n.Title,
		Body:           n.Body,
		Data:           n.Data,
		Attempt:        0,
		CreatedAt:      n.CreatedAt.Format(time.RFC3339),
	}
	if err := s.queue.Enqueue(ctx, job); err != nil {
		slog.Error("failed to enqueue notification delivery", "error", err, "notification_id", n.ID)
		// Fallback to synchronous delivery
		s.sendPushIfOffline(ctx, n)
		if n.Type != notif.TypeNewMessage {
			s.sendEmail(ctx, n)
		}
	}
}

func (s *Service) getPreferencesForType(ctx context.Context, userID uuid.UUID, nType notif.NotificationType) *notif.Preferences {
	saved, err := s.notifications.GetPreferences(ctx, userID)
	if err != nil {
		slog.Warn("failed to load notification preferences, using defaults", "error", err)
		return notif.DefaultPreferences(userID, nType)
	}
	for _, p := range saved {
		if p.NotificationType == nType {
			return p
		}
	}
	return notif.DefaultPreferences(userID, nType)
}

func (s *Service) broadcastInApp(ctx context.Context, n *notif.Notification) {
	payload, err := json.Marshal(map[string]any{
		"id":         n.ID.String(),
		"user_id":    n.UserID.String(),
		"type":       string(n.Type),
		"title":      n.Title,
		"body":       n.Body,
		"data":       n.Data,
		"created_at": n.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		slog.Error("marshal notification payload", "error", err)
		return
	}
	if err := s.broadcaster.BroadcastNotification(ctx, n.UserID, payload); err != nil {
		slog.Error("broadcast notification", "error", err)
	}
}

func (s *Service) sendPushIfOffline(ctx context.Context, n *notif.Notification) {
	if s.push == nil {
		return
	}
	online, err := s.presence.IsOnline(ctx, n.UserID)
	if err != nil {
		slog.Warn("presence check failed", "error", err)
		return
	}
	if online {
		return // user is connected via WS, skip push
	}

	tokens, err := s.notifications.ListDeviceTokens(ctx, n.UserID)
	if err != nil || len(tokens) == 0 {
		return
	}

	tokenStrings := make([]string, 0, len(tokens))
	for _, dt := range tokens {
		tokenStrings = append(tokenStrings, dt.Token)
	}

	data := make(map[string]string)
	if n.Data != nil {
		var m map[string]any
		if json.Unmarshal(n.Data, &m) == nil {
			for k, v := range m {
				data[k] = fmt.Sprintf("%v", v)
			}
		}
	}
	data["notification_type"] = string(n.Type)
	data["notification_id"] = n.ID.String()

	if err := s.push.SendPush(ctx, tokenStrings, n.Title, n.Body, data); err != nil {
		slog.Error("send push notification", "error", err, "user_id", n.UserID)
	}
}

func (s *Service) sendEmail(ctx context.Context, n *notif.Notification) {
	if s.email == nil || s.users == nil {
		return
	}
	user, err := s.users.GetByID(ctx, n.UserID)
	if err != nil {
		slog.Warn("load user for email notification", "error", err)
		return
	}
	subject := n.Title + " — Marketplace Service"
	html := fmt.Sprintf(`
		<div style="font-family: sans-serif; max-width: 600px; margin: 0 auto;">
			<h2 style="color: #F43F5E;">%s</h2>
			<p>%s</p>
			<hr style="border: none; border-top: 1px solid #E2E8F0; margin: 24px 0;">
			<p style="color: #64748B; font-size: 12px;">Marketplace Service</p>
		</div>
	`, n.Title, n.Body)

	if err := s.email.SendNotification(ctx, user.Email, subject, html); err != nil {
		slog.Error("send email notification", "error", err, "user_id", n.UserID)
	}
}
