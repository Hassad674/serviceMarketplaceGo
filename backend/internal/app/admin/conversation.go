package admin

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/pkg/cursor"
)

// AdminConversation represents a conversation for admin moderation view.
type AdminConversation struct {
	ID                 uuid.UUID
	Participants       []ConversationParticipant
	MessageCount       int
	LastMessage        *string
	LastMessageAt      *time.Time
	CreatedAt          time.Time
	PendingReportCount int
	ReportedMessage    *string
}

// ConversationParticipant is a lightweight user representation for conversation listing.
type ConversationParticipant struct {
	ID          uuid.UUID
	DisplayName string
	Email       string
	Role        string
}

// AdminMessage represents a message for admin moderation view.
type AdminMessage struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderID       uuid.UUID
	SenderName     string
	SenderRole     string
	Content        string
	Type           string
	Metadata       json.RawMessage
	ReplyToID      *uuid.UUID
	CreatedAt      time.Time
}

// ListConversations returns all conversations with participants and stats for admin.
func (s *Service) ListConversations(ctx context.Context, cursorStr string, limit int, page int, sort string, filter string) ([]AdminConversation, string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 20
	}

	total, err := s.countConversations(ctx, filter)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list conversations: %w", err)
	}

	conversations, nextCursor, err := s.queryConversations(ctx, cursorStr, limit, page, sort, filter)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list conversations: %w", err)
	}

	reportCounts, err := s.loadPendingReportCounts(ctx, conversations)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list conversations: %w", err)
	}
	for i := range conversations {
		conversations[i].PendingReportCount = reportCounts[conversations[i].ID]
	}

	if filter == "reported_messages" || filter == "reported" {
		if err := s.loadReportedMessages(ctx, conversations); err != nil {
			return nil, "", 0, fmt.Errorf("list conversations: %w", err)
		}
	}

	return conversations, nextCursor, total, nil
}

func (s *Service) countConversations(ctx context.Context, filter string) (int, error) {
	var total int
	var err error
	switch filter {
	case "reported":
		err = s.db.QueryRowContext(ctx, queryAdminCountReportedConversations).Scan(&total)
	case "reported_conversations":
		err = s.db.QueryRowContext(ctx, queryAdminCountReportedConvOnly).Scan(&total)
	case "reported_messages":
		err = s.db.QueryRowContext(ctx, queryAdminCountReportedMsgOnly).Scan(&total)
	default:
		err = s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM conversations").Scan(&total)
	}
	if err != nil {
		return 0, fmt.Errorf("count conversations: %w", err)
	}
	return total, nil
}

func (s *Service) queryConversations(ctx context.Context, cursorStr string, limit int, page int, sort string, filter string) ([]AdminConversation, string, error) {
	useOffset := page > 0 && cursorStr == ""
	query, args := buildConversationListQuery(cursorStr, sort, filter, limit, page, useOffset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query conversations: %w", err)
	}
	defer rows.Close()

	conversations, nextCursor, err := scanAdminConversations(rows, limit)
	if err != nil {
		return nil, "", err
	}

	for i := range conversations {
		participants, pErr := s.loadParticipants(ctx, conversations[i].ID)
		if pErr != nil {
			return nil, "", fmt.Errorf("load participants: %w", pErr)
		}
		conversations[i].Participants = participants
	}

	return conversations, nextCursor, nil
}

func scanAdminConversations(rows *sql.Rows, limit int) ([]AdminConversation, string, error) {
	var results []AdminConversation

	for rows.Next() {
		var c AdminConversation
		var lastMsg sql.NullString
		var lastMsgAt sql.NullTime

		if err := rows.Scan(
			&c.ID, &c.MessageCount, &lastMsg, &lastMsgAt, &c.CreatedAt,
		); err != nil {
			return nil, "", fmt.Errorf("scan conversation: %w", err)
		}

		if lastMsg.Valid {
			c.LastMessage = &lastMsg.String
		}
		if lastMsgAt.Valid {
			c.LastMessageAt = &lastMsgAt.Time
		}

		results = append(results, c)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration: %w", err)
	}

	if results == nil {
		results = []AdminConversation{}
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}

	return results, nextCursor, nil
}

func (s *Service) loadParticipants(ctx context.Context, conversationID uuid.UUID) ([]ConversationParticipant, error) {
	rows, err := s.db.QueryContext(ctx, queryAdminConversationParticipants, conversationID)
	if err != nil {
		return nil, fmt.Errorf("query participants: %w", err)
	}
	defer rows.Close()

	var participants []ConversationParticipant
	for rows.Next() {
		var p ConversationParticipant
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.Email, &p.Role); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		participants = append(participants, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if participants == nil {
		participants = []ConversationParticipant{}
	}

	return participants, nil
}

// GetConversation returns a single conversation with participants and stats for admin.
func (s *Service) GetConversation(ctx context.Context, conversationID uuid.UUID) (*AdminConversation, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	var c AdminConversation
	var lastMsg sql.NullString
	var lastMsgAt sql.NullTime

	err := s.db.QueryRowContext(ctx, queryAdminGetConversation, conversationID).Scan(
		&c.ID, &c.MessageCount, &lastMsg, &lastMsgAt, &c.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	if lastMsg.Valid {
		c.LastMessage = &lastMsg.String
	}
	if lastMsgAt.Valid {
		c.LastMessageAt = &lastMsgAt.Time
	}

	participants, err := s.loadParticipants(ctx, c.ID)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	c.Participants = participants

	return &c, nil
}

// GetConversationMessages returns messages for a conversation (admin view, no user filter).
func (s *Service) GetConversationMessages(ctx context.Context, conversationID uuid.UUID, cursorStr string, limit int) ([]AdminMessage, string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = s.db.QueryContext(ctx, queryAdminListMessagesFirst, conversationID, limit+1)
	} else {
		c, cErr := cursor.Decode(cursorStr)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = s.db.QueryContext(ctx, queryAdminListMessagesWithCursor,
			conversationID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	return scanAdminMessages(rows, limit)
}

func scanAdminMessages(rows *sql.Rows, limit int) ([]AdminMessage, string, error) {
	var results []AdminMessage

	for rows.Next() {
		var m AdminMessage
		var metadata []byte

		if err := rows.Scan(
			&m.ID, &m.ConversationID, &m.SenderID, &m.Content,
			&m.Type, &metadata, &m.ReplyToID, &m.CreatedAt,
			&m.SenderName, &m.SenderRole,
		); err != nil {
			return nil, "", fmt.Errorf("scan message: %w", err)
		}

		if len(metadata) > 0 {
			m.Metadata = metadata
		}

		results = append(results, m)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration: %w", err)
	}

	if results == nil {
		results = []AdminMessage{}
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}

	return results, nextCursor, nil
}

func (s *Service) loadReportedMessages(ctx context.Context, conversations []AdminConversation) error {
	for i := range conversations {
		var content sql.NullString
		err := s.db.QueryRowContext(ctx, queryAdminReportedMessage, conversations[i].ID).Scan(&content)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("load reported message for %s: %w", conversations[i].ID, err)
		}
		if content.Valid {
			conversations[i].ReportedMessage = &content.String
		}
	}
	return nil
}

func (s *Service) loadPendingReportCounts(ctx context.Context, conversations []AdminConversation) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int, len(conversations))
	if len(conversations) == 0 {
		return counts, nil
	}

	ids := make([]uuid.UUID, len(conversations))
	for i, c := range conversations {
		ids[i] = c.ID
	}

	rows, err := s.db.QueryContext(ctx, queryAdminPendingReportCounts, pq.Array(ids))
	if err != nil {
		return nil, fmt.Errorf("load pending report counts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var convID uuid.UUID
		var count int
		if err := rows.Scan(&convID, &count); err != nil {
			return nil, fmt.Errorf("scan report count: %w", err)
		}
		counts[convID] = count
	}

	return counts, nil
}

func orderByClause(sort string) string {
	switch sort {
	case "oldest":
		return "ORDER BY c.created_at ASC, c.id ASC"
	case "most_messages":
		return "ORDER BY message_count DESC, c.id DESC"
	default:
		return "ORDER BY COALESCE(lm.created_at, c.created_at) DESC, c.id DESC"
	}
}

func buildConversationListQuery(cursorStr string, sort string, filter string, limit int, page int, useOffset bool) (string, []any) {
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

	where := filterWhereClause(filter)
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

	orderBy := orderByClause(sort)

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

func filterWhereClause(filter string) string {
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

// SQL queries for admin conversation endpoints.
const queryAdminCountReportedConversations = `
	SELECT COUNT(*) FROM conversations c
	WHERE EXISTS (
		SELECT 1 FROM reports r
		WHERE r.status = 'pending'
		AND (r.conversation_id = c.id
			OR (r.target_type = 'user' AND r.target_id IN (
				SELECT user_id FROM conversation_participants WHERE conversation_id = c.id)))
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

const queryAdminReportedMessage = `
	SELECT m.content FROM messages m
	JOIN reports r ON r.target_type = 'message' AND r.target_id = m.id AND r.status = 'pending'
	WHERE m.conversation_id = $1
	ORDER BY r.created_at DESC LIMIT 1`
