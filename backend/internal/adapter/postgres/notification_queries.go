package postgres

const queryInsertNotification = `
	INSERT INTO notifications (id, user_id, type, title, body, data, created_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

const queryGetNotificationByID = `
	SELECT id, user_id, type, title, body, data, read_at, created_at
	FROM notifications
	WHERE id = $1`

const queryListNotificationsFirst = `
	SELECT id, user_id, type, title, body, data, read_at, created_at
	FROM notifications
	WHERE user_id = $1
	ORDER BY created_at DESC, id DESC
	LIMIT $2`

const queryListNotificationsWithCursor = `
	SELECT id, user_id, type, title, body, data, read_at, created_at
	FROM notifications
	WHERE user_id = $1
		AND (created_at, id) < ($2, $3)
	ORDER BY created_at DESC, id DESC
	LIMIT $4`

const queryCountUnread = `
	SELECT COUNT(*)
	FROM notifications
	WHERE user_id = $1 AND read_at IS NULL`

const queryNotifMarkAsRead = `
	UPDATE notifications
	SET read_at = now()
	WHERE id = $1 AND user_id = $2 AND read_at IS NULL`

const queryNotifMarkAllAsRead = `
	UPDATE notifications
	SET read_at = now()
	WHERE user_id = $1 AND read_at IS NULL`

const queryDeleteNotification = `
	DELETE FROM notifications
	WHERE id = $1 AND user_id = $2`

const queryGetPreferences = `
	SELECT user_id, notification_type, in_app, push, email
	FROM notification_preferences
	WHERE user_id = $1`

const queryUpsertPreference = `
	INSERT INTO notification_preferences (user_id, notification_type, in_app, push, email)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (user_id, notification_type) DO UPDATE
	SET in_app = EXCLUDED.in_app, push = EXCLUDED.push, email = EXCLUDED.email`

const queryInsertDeviceToken = `
	INSERT INTO device_tokens (id, user_id, token, platform, created_at)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (user_id, token) DO NOTHING`

const queryListDeviceTokens = `
	SELECT id, user_id, token, platform, created_at
	FROM device_tokens
	WHERE user_id = $1`

const queryDeleteDeviceToken = `
	DELETE FROM device_tokens
	WHERE user_id = $1 AND token = $2`
