package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/cursor"
)

type ConversationRepository struct {
	db *sql.DB
}

func NewConversationRepository(db *sql.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

func (r *ConversationRepository) FindOrCreateConversation(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	// Use SERIALIZABLE transaction to prevent race conditions where two concurrent
	// requests both see "no conversation" and create duplicates.
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return uuid.UUID{}, false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Check for existing conversation inside the transaction
	var convID uuid.UUID
	err = tx.QueryRowContext(ctx, queryFindExistingConversation, userA, userB).Scan(&convID)
	if err == nil {
		_ = tx.Commit()
		return convID, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.UUID{}, false, fmt.Errorf("find conversation: %w", err)
	}

	// Create new conversation
	conv := message.NewConversation()
	if _, err := tx.ExecContext(ctx, queryInsertConversation, conv.ID, conv.CreatedAt, conv.UpdatedAt); err != nil {
		return uuid.UUID{}, false, fmt.Errorf("insert conversation: %w", err)
	}

	now := time.Now()
	if _, err := tx.ExecContext(ctx, queryInsertParticipant, conv.ID, userA, now); err != nil {
		return uuid.UUID{}, false, fmt.Errorf("insert participant A: %w", err)
	}
	if _, err := tx.ExecContext(ctx, queryInsertParticipant, conv.ID, userB, now); err != nil {
		return uuid.UUID{}, false, fmt.Errorf("insert participant B: %w", err)
	}

	if err := tx.Commit(); err != nil {
		// On serialization failure, retry once: the other transaction likely created it
		if isSerializationError(err) {
			return r.retryFindConversation(ctx, userA, userB)
		}
		return uuid.UUID{}, false, fmt.Errorf("commit: %w", err)
	}

	return conv.ID, true, nil
}

// retryFindConversation is called after a serialization error to retrieve the
// conversation that was created by the concurrent transaction.
func (r *ConversationRepository) retryFindConversation(ctx context.Context, userA, userB uuid.UUID) (uuid.UUID, bool, error) {
	var convID uuid.UUID
	err := r.db.QueryRowContext(ctx, queryFindExistingConversation, userA, userB).Scan(&convID)
	if err == nil {
		return convID, false, nil
	}
	return uuid.UUID{}, false, fmt.Errorf("retry find conversation after serialization error: %w", err)
}

// isSerializationError checks if the error is a PostgreSQL serialization failure (40001).
func isSerializationError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "pq: could not serialize") ||
		strings.Contains(err.Error(), "40001")
}

func (r *ConversationRepository) GetConversation(ctx context.Context, id uuid.UUID) (*message.Conversation, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	conv := &message.Conversation{}
	err := r.db.QueryRowContext(ctx, queryGetConversation, id).Scan(
		&conv.ID, &conv.CreatedAt, &conv.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, message.ErrConversationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get conversation: %w", err)
	}

	return conv, nil
}

func (r *ConversationRepository) ListConversations(ctx context.Context, params repository.ListConversationsParams) ([]repository.ConversationSummary, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error

	if params.Cursor == "" {
		rows, err = r.db.QueryContext(ctx, queryListConversationsFirst, params.UserID, limit+1)
	} else {
		c, cErr := cursor.Decode(params.Cursor)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListConversationsWithCursor,
			params.UserID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	results, nextCursor, err := scanConversationSummaries(rows, limit)
	if err != nil {
		return nil, "", err
	}

	return results, nextCursor, nil
}

func scanConversationSummaries(rows *sql.Rows, limit int) ([]repository.ConversationSummary, string, error) {
	var results []repository.ConversationSummary

	for rows.Next() {
		var s repository.ConversationSummary
		var lastMsgAt sql.NullTime
		var lastMsg sql.NullString

		if err := rows.Scan(
			&s.ConversationID,
			&s.OtherUserID,
			&s.OtherUserName,
			&s.OtherUserRole,
			&s.OtherPhotoURL,
			&lastMsg,
			&lastMsgAt,
			&s.LastMessageSeq,
			&s.UnreadCount,
		); err != nil {
			return nil, "", fmt.Errorf("scan conversation: %w", err)
		}

		if lastMsg.Valid {
			s.LastMessage = &lastMsg.String
		}
		if lastMsgAt.Valid {
			t := lastMsgAt.Time
			s.LastMessageAt = &t
		}

		results = append(results, s)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration: %w", err)
	}

	if results == nil {
		results = []repository.ConversationSummary{}
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		if last.LastMessageAt != nil {
			nextCursor = cursor.Encode(*last.LastMessageAt, last.ConversationID)
		}
		results = results[:limit]
	}

	return results, nextCursor, nil
}

func (r *ConversationRepository) IsParticipant(ctx context.Context, conversationID, userID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var exists bool
	err := r.db.QueryRowContext(ctx, queryIsParticipant, conversationID, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check participant: %w", err)
	}

	return exists, nil
}

func (r *ConversationRepository) CreateMessage(ctx context.Context, msg *message.Message) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// Lock conversation row to prevent concurrent seq conflicts
	if _, err := tx.ExecContext(ctx, queryLockConversation, msg.ConversationID); err != nil {
		return fmt.Errorf("lock conversation: %w", err)
	}

	// Get next sequence number
	var seq int
	if err := tx.QueryRowContext(ctx, queryNextSeq, msg.ConversationID).Scan(&seq); err != nil {
		return fmt.Errorf("get next seq: %w", err)
	}
	msg.Seq = seq

	if _, err := tx.ExecContext(ctx, queryInsertMessage,
		msg.ID, msg.ConversationID, msg.SenderID, msg.Content,
		string(msg.Type), msg.Metadata, msg.ReplyToID, msg.Seq, string(msg.Status),
		msg.CreatedAt, msg.UpdatedAt,
	); err != nil {
		return fmt.Errorf("insert message: %w", err)
	}

	if _, err := tx.ExecContext(ctx, queryUpdateConversationTimestamp,
		msg.ConversationID, msg.CreatedAt,
	); err != nil {
		return fmt.Errorf("update conversation: %w", err)
	}

	return tx.Commit()
}

func (r *ConversationRepository) GetMessage(ctx context.Context, id uuid.UUID) (*message.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	msg, err := scanMessage(r.db.QueryRowContext(ctx, queryGetMessage, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, message.ErrMessageNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}

	return msg, nil
}

func (r *ConversationRepository) ListMessages(ctx context.Context, params repository.ListMessagesParams) ([]*message.Message, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var rows *sql.Rows
	var err error

	if params.Cursor == "" {
		rows, err = r.db.QueryContext(ctx, queryListMessagesFirst, params.ConversationID, limit+1)
	} else {
		c, cErr := cursor.Decode(params.Cursor)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListMessagesWithCursor,
			params.ConversationID, c.CreatedAt, c.ID, limit+1)
	}
	if err != nil {
		return nil, "", fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	return scanMessageList(rows, limit)
}

func scanMessageList(rows *sql.Rows, limit int) ([]*message.Message, string, error) {
	var results []*message.Message

	for rows.Next() {
		msg, err := scanMessageFromRows(rows)
		if err != nil {
			return nil, "", fmt.Errorf("scan message: %w", err)
		}
		results = append(results, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, "", fmt.Errorf("rows iteration: %w", err)
	}

	if results == nil {
		results = []*message.Message{}
	}

	var nextCursor string
	if len(results) > limit {
		last := results[limit-1]
		nextCursor = cursor.Encode(last.CreatedAt, last.ID)
		results = results[:limit]
	}

	return results, nextCursor, nil
}

func (r *ConversationRepository) GetMessagesSinceSeq(ctx context.Context, conversationID uuid.UUID, sinceSeq int, limit int) ([]*message.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, queryMessagesSinceSeq, conversationID, sinceSeq, limit)
	if err != nil {
		return nil, fmt.Errorf("get messages since seq: %w", err)
	}
	defer rows.Close()

	var results []*message.Message
	for rows.Next() {
		msg, err := scanMessageFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		results = append(results, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	if results == nil {
		results = []*message.Message{}
	}

	return results, nil
}

func (r *ConversationRepository) ListMessagesSinceTime(ctx context.Context, conversationID uuid.UUID, since time.Time, limit int) ([]*message.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if limit <= 0 || limit > 500 {
		limit = 200
	}

	rows, err := r.db.QueryContext(ctx, queryListMessagesSinceTime, conversationID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("list messages since time: %w", err)
	}
	defer rows.Close()

	var results []*message.Message
	for rows.Next() {
		msg, err := scanMessageFromRows(rows)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		results = append(results, msg)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}
	if results == nil {
		results = []*message.Message{}
	}
	return results, nil
}

func (r *ConversationRepository) UpdateMessage(ctx context.Context, msg *message.Message) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	result, err := r.db.ExecContext(ctx, queryUpdateMessage,
		msg.ID, msg.Content, msg.EditedAt, msg.DeletedAt, msg.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update message: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check rows affected: %w", err)
	}
	if rows == 0 {
		return message.ErrMessageNotFound
	}

	return nil
}

func (r *ConversationRepository) IncrementUnread(ctx context.Context, conversationID, senderID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryIncrementUnread, conversationID, senderID)
	if err != nil {
		return fmt.Errorf("increment unread: %w", err)
	}

	return nil
}

func (r *ConversationRepository) MarkAsRead(ctx context.Context, conversationID, userID uuid.UUID, seq int) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryMarkAsRead, conversationID, userID, seq)
	if err != nil {
		return fmt.Errorf("mark as read: %w", err)
	}

	return nil
}

func (r *ConversationRepository) GetTotalUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var total int
	err := r.db.QueryRowContext(ctx, queryGetTotalUnread, userID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("get total unread: %w", err)
	}

	return total, nil
}

func (r *ConversationRepository) GetTotalUnreadBatch(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryGetTotalUnreadBatch, pq.Array(userIDs))
	if err != nil {
		return nil, fmt.Errorf("get total unread batch: %w", err)
	}
	defer rows.Close()

	result := make(map[uuid.UUID]int, len(userIDs))
	for rows.Next() {
		var uid uuid.UUID
		var count int
		if err := rows.Scan(&uid, &count); err != nil {
			return nil, fmt.Errorf("scan unread batch: %w", err)
		}
		result[uid] = count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	// Ensure all requested userIDs are in the map (default 0)
	for _, uid := range userIDs {
		if _, ok := result[uid]; !ok {
			result[uid] = 0
		}
	}

	return result, nil
}

func (r *ConversationRepository) GetParticipantIDs(ctx context.Context, conversationID uuid.UUID) ([]uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryGetParticipantIDs, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get participant ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan participant id: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return ids, nil
}

func (r *ConversationRepository) UpdateMessageStatus(ctx context.Context, messageID uuid.UUID, status message.MessageStatus) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryUpdateMessageStatus, messageID, string(status))
	if err != nil {
		return fmt.Errorf("update message status: %w", err)
	}

	return nil
}

func (r *ConversationRepository) MarkMessagesAsRead(ctx context.Context, conversationID, readerID uuid.UUID, upToSeq int) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryMarkMessagesAsRead, conversationID, readerID, upToSeq)
	if err != nil {
		return fmt.Errorf("mark messages as read: %w", err)
	}

	return nil
}

func (r *ConversationRepository) GetContactIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryGetContactIDs, userID)
	if err != nil {
		return nil, fmt.Errorf("get contact ids: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan contact id: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration: %w", err)
	}

	return ids, nil
}

// scanMessage scans a single message from a QueryRow result (with reply JOIN).
func scanMessage(row *sql.Row) (*message.Message, error) {
	msg := &message.Message{}
	var msgType, status string
	var metadata []byte
	var replyID, replySenderID *uuid.UUID
	var replyContent, replyType *string

	err := row.Scan(
		&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content,
		&msgType, &metadata, &msg.ReplyToID, &msg.Seq, &status,
		&msg.EditedAt, &msg.DeletedAt, &msg.CreatedAt, &msg.UpdatedAt,
		&replyID, &replySenderID, &replyContent, &replyType,
	)
	if err != nil {
		return nil, err
	}

	if len(metadata) > 0 {
		msg.Metadata = metadata
	}
	msg.Type = message.MessageType(msgType)
	msg.Status = message.MessageStatus(status)
	msg.ReplyPreview = buildReplyPreview(replyID, replySenderID, replyContent, replyType)
	return msg, nil
}

// scanMessageFromRows scans a single message from a Rows iterator (with reply JOIN).
func scanMessageFromRows(rows *sql.Rows) (*message.Message, error) {
	msg := &message.Message{}
	var msgType, status string
	var metadata []byte
	var replyID, replySenderID *uuid.UUID
	var replyContent, replyType *string

	err := rows.Scan(
		&msg.ID, &msg.ConversationID, &msg.SenderID, &msg.Content,
		&msgType, &metadata, &msg.ReplyToID, &msg.Seq, &status,
		&msg.EditedAt, &msg.DeletedAt, &msg.CreatedAt, &msg.UpdatedAt,
		&replyID, &replySenderID, &replyContent, &replyType,
	)
	if err != nil {
		return nil, err
	}

	if len(metadata) > 0 {
		msg.Metadata = metadata
	}
	msg.Type = message.MessageType(msgType)
	msg.Status = message.MessageStatus(status)
	msg.ReplyPreview = buildReplyPreview(replyID, replySenderID, replyContent, replyType)
	return msg, nil
}

// buildReplyPreview constructs a ReplyPreview from nullable JOIN columns.
func buildReplyPreview(id, senderID *uuid.UUID, content, msgType *string) *message.ReplyPreview {
	if id == nil {
		return nil
	}
	rp := &message.ReplyPreview{ID: *id}
	if senderID != nil {
		rp.SenderID = *senderID
	}
	if content != nil {
		rp.Content = message.TruncateContent(*content, 100)
	}
	if msgType != nil {
		rp.Type = message.MessageType(*msgType)
	}
	return rp
}

func (r *ConversationRepository) SaveMessageHistory(ctx context.Context, messageID, performedBy uuid.UUID, content, action string) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, querySaveMessageHistory, messageID, content, action, performedBy)
	if err != nil {
		return fmt.Errorf("save message history: %w", err)
	}
	return nil
}

func (r *ConversationRepository) UpdateMessageModeration(ctx context.Context, messageID uuid.UUID, status string, score float64, labelsJSON []byte) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryUpdateMessageModeration, messageID, status, score, labelsJSON)
	if err != nil {
		return fmt.Errorf("update message moderation: %w", err)
	}
	return nil
}
