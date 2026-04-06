package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/cursor"
)

// AdminConversationRepository implements repository.AdminConversationRepository using PostgreSQL.
type AdminConversationRepository struct {
	db *sql.DB
}

// NewAdminConversationRepository creates a new PostgreSQL-backed admin conversation repository.
func NewAdminConversationRepository(db *sql.DB) *AdminConversationRepository {
	return &AdminConversationRepository{db: db}
}

// List returns conversations with pagination, sorting, and filtering.
func (r *AdminConversationRepository) List(ctx context.Context, filters repository.AdminConversationFilters) ([]repository.AdminConversation, string, int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	total, err := r.countConversations(ctx, filters.Filter)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list conversations: %w", err)
	}

	conversations, nextCursor, err := r.queryConversations(ctx, filters)
	if err != nil {
		return nil, "", 0, fmt.Errorf("list conversations: %w", err)
	}

	if err := r.loadPendingReportCounts(ctx, conversations); err != nil {
		return nil, "", 0, fmt.Errorf("list conversations: %w", err)
	}

	if filters.Filter == "reported_messages" || filters.Filter == "reported" {
		if err := r.loadReportedMessages(ctx, conversations); err != nil {
			return nil, "", 0, fmt.Errorf("list conversations: %w", err)
		}
	}

	return conversations, nextCursor, total, nil
}

// GetByID returns a single conversation with participants and stats.
func (r *AdminConversationRepository) GetByID(ctx context.Context, conversationID uuid.UUID) (*repository.AdminConversation, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var c repository.AdminConversation
	var lastMsg sql.NullString
	var lastMsgAt sql.NullTime

	err := r.db.QueryRowContext(ctx, queryAdminGetConversation, conversationID).Scan(
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

	participants, err := r.loadParticipants(ctx, c.ID)
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}
	c.Participants = participants

	return &c, nil
}

// ListMessages returns messages for a conversation with cursor pagination.
func (r *AdminConversationRepository) ListMessages(ctx context.Context, conversationID uuid.UUID, cursorStr string, limit int) ([]repository.AdminMessage, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	if cursorStr == "" {
		rows, err = r.db.QueryContext(ctx, queryAdminListMessagesFirst, conversationID, limit+1)
	} else {
		c, cErr := cursor.Decode(cursorStr)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryAdminListMessagesWithCursor,
			conversationID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	return scanAdminMessages(rows, limit)
}

func (r *AdminConversationRepository) countConversations(ctx context.Context, filter string) (int, error) {
	var total int
	var err error
	switch filter {
	case "reported":
		err = r.db.QueryRowContext(ctx, queryAdminCountReportedConversations).Scan(&total)
	case "reported_conversations":
		err = r.db.QueryRowContext(ctx, queryAdminCountReportedConvOnly).Scan(&total)
	case "reported_messages":
		err = r.db.QueryRowContext(ctx, queryAdminCountReportedMsgOnly).Scan(&total)
	default:
		err = r.db.QueryRowContext(ctx, queryAdminCountAllConversations).Scan(&total)
	}
	if err != nil {
		return 0, fmt.Errorf("count conversations: %w", err)
	}
	return total, nil
}

func (r *AdminConversationRepository) queryConversations(ctx context.Context, filters repository.AdminConversationFilters) ([]repository.AdminConversation, string, error) {
	useOffset := filters.Page > 0 && filters.Cursor == ""
	query, args := buildAdminConversationListQuery(filters.Cursor, filters.Sort, filters.Filter, filters.Limit, filters.Page, useOffset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, "", fmt.Errorf("query conversations: %w", err)
	}
	defer rows.Close()

	conversations, nextCursor, err := scanAdminConversations(rows, filters.Limit)
	if err != nil {
		return nil, "", err
	}

	for i := range conversations {
		participants, pErr := r.loadParticipants(ctx, conversations[i].ID)
		if pErr != nil {
			return nil, "", fmt.Errorf("load participants: %w", pErr)
		}
		conversations[i].Participants = participants
	}

	return conversations, nextCursor, nil
}

func scanAdminConversations(rows *sql.Rows, limit int) ([]repository.AdminConversation, string, error) {
	var results []repository.AdminConversation

	for rows.Next() {
		var c repository.AdminConversation
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
		results = []repository.AdminConversation{}
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}

	return results, nextCursor, nil
}

func (r *AdminConversationRepository) loadParticipants(ctx context.Context, conversationID uuid.UUID) ([]repository.ConversationParticipant, error) {
	rows, err := r.db.QueryContext(ctx, queryAdminConversationParticipants, conversationID)
	if err != nil {
		return nil, fmt.Errorf("query participants: %w", err)
	}
	defer rows.Close()

	var participants []repository.ConversationParticipant
	for rows.Next() {
		var p repository.ConversationParticipant
		if err := rows.Scan(&p.ID, &p.DisplayName, &p.Email, &p.Role); err != nil {
			return nil, fmt.Errorf("scan participant: %w", err)
		}
		participants = append(participants, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if participants == nil {
		participants = []repository.ConversationParticipant{}
	}

	return participants, nil
}

func (r *AdminConversationRepository) loadReportedMessages(ctx context.Context, conversations []repository.AdminConversation) error {
	for i := range conversations {
		var content sql.NullString
		err := r.db.QueryRowContext(ctx, queryAdminReportedMessage, conversations[i].ID).Scan(&content)
		if err != nil && err != sql.ErrNoRows {
			return fmt.Errorf("load reported message for %s: %w", conversations[i].ID, err)
		}
		if content.Valid {
			conversations[i].ReportedMessage = &content.String
		}
	}
	return nil
}

func (r *AdminConversationRepository) loadPendingReportCounts(ctx context.Context, conversations []repository.AdminConversation) error {
	if len(conversations) == 0 {
		return nil
	}

	ids := make([]uuid.UUID, len(conversations))
	for i, c := range conversations {
		ids[i] = c.ID
	}

	rows, err := r.db.QueryContext(ctx, queryAdminPendingReportCounts, pq.Array(ids))
	if err != nil {
		return fmt.Errorf("load pending report counts: %w", err)
	}
	defer rows.Close()

	counts := make(map[uuid.UUID]int, len(conversations))
	for rows.Next() {
		var convID uuid.UUID
		var count int
		if err := rows.Scan(&convID, &count); err != nil {
			return fmt.Errorf("scan report count: %w", err)
		}
		counts[convID] = count
	}

	for i := range conversations {
		conversations[i].PendingReportCount = counts[conversations[i].ID]
	}
	return nil
}

func scanAdminMessages(rows *sql.Rows, limit int) ([]repository.AdminMessage, string, error) {
	var results []repository.AdminMessage

	for rows.Next() {
		var m repository.AdminMessage
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
		results = []repository.AdminMessage{}
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}

	return results, nextCursor, nil
}
