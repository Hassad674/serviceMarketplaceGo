package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/notification"
	"marketplace-backend/pkg/cursor"
)

// NotificationRepository implements repository.NotificationRepository using PostgreSQL.
type NotificationRepository struct {
	db *sql.DB
}

// NewNotificationRepository creates a new PostgreSQL-backed notification repository.
func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// notifScanner is satisfied by both *sql.Row and *sql.Rows.
type notifScanner interface {
	Scan(dest ...any) error
}

func scanNotification(s notifScanner) (*notification.Notification, error) {
	var n notification.Notification
	var nType string
	var data []byte
	var readAt sql.NullTime

	err := s.Scan(&n.ID, &n.UserID, &nType, &n.Title, &n.Body, &data, &readAt, &n.CreatedAt)
	if err != nil {
		return nil, err
	}
	n.Type = notification.NotificationType(nType)
	n.Data = json.RawMessage(data)
	if readAt.Valid {
		n.ReadAt = &readAt.Time
	}
	return &n, nil
}

func (r *NotificationRepository) Create(ctx context.Context, n *notification.Notification) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	data, _ := json.Marshal(n.Data)
	_, err := r.db.ExecContext(ctx, queryInsertNotification,
		n.ID, n.UserID, string(n.Type), n.Title, n.Body, data, n.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}
	return nil
}

func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*notification.Notification, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	n, err := scanNotification(r.db.QueryRowContext(ctx, queryGetNotificationByID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, notification.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get notification by id: %w", err)
	}
	return n, nil
}

func (r *NotificationRepository) List(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*notification.Notification, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryListNotificationsFirst, userID, limit+1)
	} else {
		c, decErr := cursor.Decode(cursorStr)
		if decErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", decErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListNotificationsWithCursor, userID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list notifications: %w", err)
	}
	defer rows.Close()

	var notifs []*notification.Notification
	for rows.Next() {
		n, scanErr := scanNotification(rows)
		if scanErr != nil {
			return nil, "", fmt.Errorf("scan notification: %w", scanErr)
		}
		notifs = append(notifs, n)
	}

	var nextCursor string
	if len(notifs) > limit {
		last := notifs[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		notifs = notifs[:limit]
	}

	return notifs, nextCursor, nil
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var count int
	err := r.db.QueryRowContext(ctx, queryCountUnread, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count unread: %w", err)
	}
	return count, nil
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, queryNotifMarkAsRead, id, userID)
	if err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return notification.ErrNotFound
	}
	return nil
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryNotifMarkAllAsRead, userID)
	if err != nil {
		return fmt.Errorf("mark all as read: %w", err)
	}
	return nil
}

func (r *NotificationRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, queryDeleteNotification, id, userID)
	if err != nil {
		return fmt.Errorf("delete notification: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return notification.ErrNotFound
	}
	return nil
}

func (r *NotificationRepository) GetPreferences(ctx context.Context, userID uuid.UUID) ([]*notification.Preferences, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryGetPreferences, userID)
	if err != nil {
		return nil, fmt.Errorf("get preferences: %w", err)
	}
	defer rows.Close()

	var prefs []*notification.Preferences
	for rows.Next() {
		var p notification.Preferences
		var nType string
		if err := rows.Scan(&p.UserID, &nType, &p.InApp, &p.Push, &p.Email); err != nil {
			return nil, fmt.Errorf("scan preference: %w", err)
		}
		p.NotificationType = notification.NotificationType(nType)
		prefs = append(prefs, &p)
	}
	return prefs, nil
}

func (r *NotificationRepository) UpsertPreference(ctx context.Context, pref *notification.Preferences) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryUpsertPreference,
		pref.UserID, string(pref.NotificationType), pref.InApp, pref.Push, pref.Email,
	)
	if err != nil {
		return fmt.Errorf("upsert preference: %w", err)
	}
	return nil
}

func (r *NotificationRepository) CreateDeviceToken(ctx context.Context, dt *notification.DeviceToken) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryInsertDeviceToken,
		dt.ID, dt.UserID, dt.Token, dt.Platform, dt.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert device token: %w", err)
	}
	return nil
}

func (r *NotificationRepository) ListDeviceTokens(ctx context.Context, userID uuid.UUID) ([]*notification.DeviceToken, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryListDeviceTokens, userID)
	if err != nil {
		return nil, fmt.Errorf("list device tokens: %w", err)
	}
	defer rows.Close()

	var tokens []*notification.DeviceToken
	for rows.Next() {
		var dt notification.DeviceToken
		if err := rows.Scan(&dt.ID, &dt.UserID, &dt.Token, &dt.Platform, &dt.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan device token: %w", err)
		}
		tokens = append(tokens, &dt)
	}
	return tokens, nil
}

func (r *NotificationRepository) DeleteDeviceToken(ctx context.Context, userID uuid.UUID, token string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryDeleteDeviceToken, userID, token)
	if err != nil {
		return fmt.Errorf("delete device token: %w", err)
	}
	return nil
}
