package postgres

import (
	"fmt"

	"marketplace-backend/pkg/cursor"
)

// SQL queries for admin conversation endpoints.

const queryAdminCountAllConversations = `SELECT COUNT(*) FROM conversations`

const queryAdminCountReportedConversations = `
	SELECT COUNT(*) FROM conversations c
	WHERE EXISTS (
		SELECT 1 FROM reports r
		WHERE r.status = 'pending'
		AND (r.conversation_id = c.id
			OR (r.target_type = 'user' AND r.target_id IN (
				SELECT user_id FROM conversation_participants WHERE conversation_id = c.id)))
	)`

const queryAdminCountReportedConvOnly = `
	SELECT COUNT(*) FROM conversations c
	WHERE EXISTS (
		SELECT 1 FROM reports r
		WHERE r.status = 'pending'
		AND r.target_type = 'user'
		AND r.target_id IN (
			SELECT user_id FROM conversation_participants WHERE conversation_id = c.id)
	)`

const queryAdminCountReportedMsgOnly = `
	SELECT COUNT(*) FROM conversations c
	WHERE EXISTS (
		SELECT 1 FROM reports r
		JOIN messages m ON m.id = r.target_id
		WHERE r.status = 'pending'
		AND r.target_type = 'message'
		AND m.conversation_id = c.id
	)`

const queryAdminPendingReportCounts = `
	SELECT c_id, COUNT(*) FROM (
		SELECT r.conversation_id AS c_id
		FROM reports r
		WHERE r.status = 'pending'
			AND r.conversation_id = ANY($1)
		UNION ALL
		SELECT cp.conversation_id AS c_id
		FROM reports r
		JOIN conversation_participants cp ON cp.user_id = r.target_id
		WHERE r.status = 'pending'
			AND r.target_type = 'user'
			AND cp.conversation_id = ANY($1)
	) sub
	GROUP BY c_id`

const queryAdminGetConversation = `
	SELECT
		c.id,
		(SELECT COUNT(*) FROM messages WHERE conversation_id = c.id) AS message_count,
		lm.content,
		lm.created_at,
		c.created_at
	FROM conversations c
	LEFT JOIN LATERAL (
		SELECT content, created_at
		FROM messages
		WHERE conversation_id = c.id
		ORDER BY created_at DESC
		LIMIT 1
	) lm ON true
	WHERE c.id = $1`

const queryAdminConversationParticipants = `
	SELECT u.id, COALESCE(u.display_name, u.first_name || ' ' || u.last_name), u.email, u.role
	FROM conversation_participants cp
	JOIN users u ON u.id = cp.user_id
	WHERE cp.conversation_id = $1`

const queryAdminListMessagesFirst = `
	SELECT
		m.id, m.conversation_id, m.sender_id, m.content,
		m.msg_type, m.metadata, m.reply_to_id, m.created_at,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name),
		COALESCE(u.role, '')
	FROM messages m
	JOIN users u ON u.id = m.sender_id
	WHERE m.conversation_id = $1 AND m.deleted_at IS NULL
	ORDER BY m.created_at ASC, m.id ASC
	LIMIT $2`

const queryAdminListMessagesWithCursor = `
	SELECT
		m.id, m.conversation_id, m.sender_id, m.content,
		m.msg_type, m.metadata, m.reply_to_id, m.created_at,
		COALESCE(u.display_name, u.first_name || ' ' || u.last_name),
		COALESCE(u.role, '')
	FROM messages m
	JOIN users u ON u.id = m.sender_id
	WHERE m.conversation_id = $1 AND m.deleted_at IS NULL
		AND (m.created_at, m.id) > ($2, $3)
	ORDER BY m.created_at ASC, m.id ASC
	LIMIT $4`

const queryAdminReportedMessage = `
	SELECT m.content FROM messages m
	JOIN reports r ON r.target_type = 'message' AND r.target_id = m.id AND r.status = 'pending'
	WHERE m.conversation_id = $1
	ORDER BY r.created_at DESC LIMIT 1`

// buildAdminConversationListQuery builds the dynamic SQL for listing conversations.
func buildAdminConversationListQuery(cursorStr string, sort string, filter string, limit int, page int, useOffset bool) (string, []any) {
	base := `SELECT
		c.id,
		(SELECT COUNT(*) FROM messages WHERE conversation_id = c.id) AS message_count,
		lm.content,
		lm.created_at,
		c.created_at
	FROM conversations c
	LEFT JOIN LATERAL (
		SELECT content, created_at
		FROM messages
		WHERE conversation_id = c.id
		ORDER BY created_at DESC
		LIMIT 1
	) lm ON true`

	where := adminConvFilterWhereClause(filter)
	var args []any
	paramIdx := 1

	if !useOffset && cursorStr != "" {
		c, err := cursor.Decode(cursorStr)
		if err == nil {
			cursorWhere := fmt.Sprintf(" (c.created_at, c.id) < ($%d, $%d)", paramIdx, paramIdx+1)
			if where == "" {
				where = " WHERE" + cursorWhere
			} else {
				where += " AND" + cursorWhere
			}
			args = append(args, c.CreatedAt, c.ID)
			paramIdx += 2
		}
	}

	orderBy := adminConvOrderByClause(sort)

	var offsetClause string
	if useOffset {
		offsetClause = fmt.Sprintf(" OFFSET $%d", paramIdx)
		args = append(args, (page-1)*limit)
		paramIdx++
	}

	limitClause := fmt.Sprintf(" LIMIT $%d", paramIdx)
	args = append(args, limit+1)

	return base + where + " " + orderBy + limitClause + offsetClause, args
}

func adminConvOrderByClause(sort string) string {
	switch sort {
	case "oldest":
		return "ORDER BY c.created_at ASC, c.id ASC"
	case "most_messages":
		return "ORDER BY message_count DESC, c.id DESC"
	default:
		return "ORDER BY COALESCE(lm.created_at, c.created_at) DESC, c.id DESC"
	}
}

func adminConvFilterWhereClause(filter string) string {
	switch filter {
	case "reported":
		return ` WHERE EXISTS (
			SELECT 1 FROM reports r
			WHERE r.status = 'pending'
			AND (r.conversation_id = c.id
				OR (r.target_type = 'user' AND r.target_id IN (
					SELECT user_id FROM conversation_participants WHERE conversation_id = c.id)))
		)`
	case "reported_conversations":
		return ` WHERE EXISTS (
			SELECT 1 FROM reports r
			WHERE r.status = 'pending'
			AND r.target_type = 'user'
			AND r.target_id IN (
				SELECT user_id FROM conversation_participants WHERE conversation_id = c.id)
		)`
	case "reported_messages":
		return ` WHERE EXISTS (
			SELECT 1 FROM reports r
			JOIN messages m ON m.id = r.target_id
			WHERE r.status = 'pending'
			AND r.target_type = 'message'
			AND m.conversation_id = c.id
		)`
	default:
		return ""
	}
}
