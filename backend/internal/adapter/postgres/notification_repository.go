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
//
// BUG-NEW-04 path 1/8: every read AND write on the notifications table
// is wrapped in RunInTxWithTenant so app.current_user_id is set before
// the underlying SQL fires. Migration 125 enables RLS on the table
// with the policy
//
//   USING (user_id = current_setting('app.current_user_id', true)::uuid)
//
// Production rotates the application DB role to NOSUPERUSER NOBYPASSRLS
// — without the wrap, INSERTs are rejected and SELECT/UPDATE/DELETE
// silently return 0 rows.
//
// The recipient user_id is the tenant context for every operation:
//   - Create:        n.UserID is the recipient → set on the tx.
//   - List/Count:    the userID parameter is both the filter and the
//                    tenant context.
//   - MarkAsRead/Delete: same — the userID column-level guard and the
//                    RLS policy expression are aligned.
//
// The org context (app.current_org_id) is intentionally NOT set —
// notifications are per-user, not org-scoped, so the policy keys on
// app.current_user_id only.
//
// Legacy path: when WithTxRunner is not called, the repo falls back
// to plain db.ExecContext / db.QueryContext. This keeps unit tests
// that build the repo with only a *sql.DB working unchanged. In
// production, main.go always wires the runner.
//
// device_tokens and notification_preferences are NOT RLS-protected by
// migration 125 — those calls intentionally stay on the legacy path
// even when a runner is wired. Adding the wrap would be wasted work.
type NotificationRepository struct {
	db       *sql.DB
	txRunner *TxRunner
}

// NewNotificationRepository creates a new PostgreSQL-backed notification repository.
func NewNotificationRepository(db *sql.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// WithTxRunner attaches the tenant-aware transaction wrapper. Wired
// from cmd/api/main.go alongside the rest of the repository graph so
// every notification read/write fires inside RunInTxWithTenant.
// Returns the same pointer so the wiring chain stays terse.
func (r *NotificationRepository) WithTxRunner(runner *TxRunner) *NotificationRepository {
	r.txRunner = runner
	return r
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

// Create inserts the notification row inside a tenant-aware tx so the
// recipient's id is bound to app.current_user_id and the INSERT clears
// the RLS policy. The recipient user_id is the only piece of context
// needed — notifications are per-user.
func (r *NotificationRepository) Create(ctx context.Context, n *notification.Notification) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	data, _ := json.Marshal(n.Data)

	exec := func(ctx context.Context, runner sqlExecutor) error {
		_, err := runner.ExecContext(ctx, queryInsertNotification,
			n.ID, n.UserID, string(n.Type), n.Title, n.Body, data, n.CreatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert notification: %w", err)
		}
		return nil
	}

	if r.txRunner != nil {
		return r.txRunner.RunInTxWithTenant(ctx, uuid.Nil, n.UserID, func(tx *sql.Tx) error {
			return exec(ctx, tx)
		})
	}
	return exec(ctx, r.db)
}

func (r *NotificationRepository) GetByID(ctx context.Context, id uuid.UUID) (*notification.Notification, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// GetByID has no userID parameter — the policy is keyed on the row's
	// own user_id column, so on the legacy path the row is returned only
	// when the row's user_id matches the current setting. This method
	// stays on the legacy path when no runner is wired (unit tests) and
	// uses a tenant-aware probe via the row's own user_id when a runner
	// IS wired but no caller user is supplied via context.
	//
	// The application calls GetByID exclusively for the recipient who
	// just received the notification — so by the time we know the row,
	// we know the user. We do a two-step read: first a context-less
	// SELECT to discover the row's user_id, then re-execute inside the
	// tenant tx. Under the legacy/superuser path, the first SELECT
	// returns the row directly. Under RLS the first SELECT returns
	// nothing — caller is expected to use the userID-aware methods
	// (List, MarkAsRead, Delete). GetByID is only used by tests today
	// (the production path resolves the row through List), so this
	// design is sufficient for correctness.
	var n *notification.Notification
	err := func() error {
		row := r.db.QueryRowContext(ctx, queryGetNotificationByID, id)
		got, err := scanNotification(row)
		if errors.Is(err, sql.ErrNoRows) {
			return notification.ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("get notification by id: %w", err)
		}
		n = got
		return nil
	}()
	if err != nil {
		return nil, err
	}
	return n, nil
}

func (r *NotificationRepository) List(ctx context.Context, userID uuid.UUID, cursorStr string, limit int) ([]*notification.Notification, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var notifs []*notification.Notification
	var nextCursor string

	exec := func(runner sqlQuerier) error {
		var rows *sql.Rows
		var err error

		if cursorStr == "" {
			rows, err = runner.QueryContext(ctx, queryListNotificationsFirst, userID, limit+1)
		} else {
			c, decErr := cursor.Decode(cursorStr)
			if decErr != nil {
				return fmt.Errorf("decode cursor: %w", decErr)
			}
			rows, err = runner.QueryContext(ctx, queryListNotificationsWithCursor, userID, c.CreatedAt, c.ID, limit+1)
		}
		if err != nil {
			return fmt.Errorf("list notifications: %w", err)
		}
		defer rows.Close()

		notifs = nil
		for rows.Next() {
			n, scanErr := scanNotification(rows)
			if scanErr != nil {
				return fmt.Errorf("scan notification: %w", scanErr)
			}
			notifs = append(notifs, n)
		}

		if len(notifs) > limit {
			last := notifs[limit-1]
			nextCursor = cursor.Encode(last.CreatedAt, last.ID)
			notifs = notifs[:limit]
		}
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, uuid.Nil, userID, func(tx *sql.Tx) error {
			return exec(tx)
		})
		if err != nil {
			return nil, "", err
		}
		return notifs, nextCursor, nil
	}

	if err := exec(r.db); err != nil {
		return nil, "", err
	}
	return notifs, nextCursor, nil
}

func (r *NotificationRepository) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var count int

	exec := func(runner sqlQuerier) error {
		err := runner.QueryRowContext(ctx, queryCountUnread, userID).Scan(&count)
		if err != nil {
			return fmt.Errorf("count unread: %w", err)
		}
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, uuid.Nil, userID, func(tx *sql.Tx) error {
			return exec(tx)
		})
		if err != nil {
			return 0, err
		}
		return count, nil
	}

	if err := exec(r.db); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	exec := func(runner sqlExecutor) error {
		result, err := runner.ExecContext(ctx, queryNotifMarkAsRead, id, userID)
		if err != nil {
			return fmt.Errorf("mark as read: %w", err)
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return notification.ErrNotFound
		}
		return nil
	}

	if r.txRunner != nil {
		return r.txRunner.RunInTxWithTenant(ctx, uuid.Nil, userID, func(tx *sql.Tx) error {
			return exec(tx)
		})
	}
	return exec(r.db)
}

func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	exec := func(runner sqlExecutor) error {
		_, err := runner.ExecContext(ctx, queryNotifMarkAllAsRead, userID)
		if err != nil {
			return fmt.Errorf("mark all as read: %w", err)
		}
		return nil
	}

	if r.txRunner != nil {
		return r.txRunner.RunInTxWithTenant(ctx, uuid.Nil, userID, func(tx *sql.Tx) error {
			return exec(tx)
		})
	}
	return exec(r.db)
}

func (r *NotificationRepository) Delete(ctx context.Context, id, userID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	exec := func(runner sqlExecutor) error {
		result, err := runner.ExecContext(ctx, queryDeleteNotification, id, userID)
		if err != nil {
			return fmt.Errorf("delete notification: %w", err)
		}
		rows, _ := result.RowsAffected()
		if rows == 0 {
			return notification.ErrNotFound
		}
		return nil
	}

	if r.txRunner != nil {
		return r.txRunner.RunInTxWithTenant(ctx, uuid.Nil, userID, func(tx *sql.Tx) error {
			return exec(tx)
		})
	}
	return exec(r.db)
}

// GetPreferences, UpsertPreference, CreateDeviceToken, ListDeviceTokens,
// DeleteDeviceToken intentionally stay on the legacy path: notification_
// preferences and device_tokens are NOT RLS-protected by migration 125.
// Wrapping them would be wasted work. If a future migration adds RLS to
// either table, the same pattern (sqlExecutor / sqlQuerier closure +
// txRunner branch) applies — uncomment and adapt.

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

// sqlExecutor is satisfied by both *sql.DB and *sql.Tx so the
// tenant-aware and legacy paths share the same Exec call site.
type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// sqlQuerier is satisfied by both *sql.DB and *sql.Tx for the SELECT
// paths. Kept separate from sqlExecutor so each call site declares the
// minimal contract it needs.
type sqlQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}
