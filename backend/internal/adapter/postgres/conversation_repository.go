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
	// txRunner is the tenant-aware transaction wrapper used by the
	// RLS-protected write paths (FindOrCreateConversation, CreateMessage).
	// It is OPTIONAL: when nil the repository falls back to plain
	// db.BeginTx — useful for unit tests that build the repo with only
	// a *sql.DB. In production main.go always wires a non-nil runner so
	// the SET LOCAL app.current_org_id / app.current_user_id calls fire
	// before any insert, satisfying the policies installed by mig 125.
	txRunner *TxRunner
}

func NewConversationRepository(db *sql.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// WithTxRunner attaches the tenant-aware transaction wrapper. Wired
// from cmd/api/main.go alongside the rest of the repository graph.
// Returning the same pointer lets the wiring chain stay terse:
//
//	postgres.NewConversationRepository(db).WithTxRunner(txRunner)
func (r *ConversationRepository) WithTxRunner(runner *TxRunner) *ConversationRepository {
	r.txRunner = runner
	return r
}

// FindOrCreateConversation finds the 1:1 conversation between userA and
// userB or creates it. senderOrgID + senderUserID are the tenant context
// of the CALLER (not necessarily either of userA / userB — system paths
// pass uuid.Nil for both). When a TxRunner is wired, the function runs
// inside RunInTxWithTenant so app.current_org_id / app.current_user_id
// are set before the INSERT into `conversations` (RLS-protected by mig
// 125). Without this, production deploys with a non-superuser DB role
// reject the INSERT with "new row violates row-level security policy".
func (r *ConversationRepository) FindOrCreateConversation(ctx context.Context, userA, userB, senderOrgID, senderUserID uuid.UUID) (uuid.UUID, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if r.txRunner != nil {
		return r.findOrCreateConversationWithRunner(ctx, userA, userB, senderOrgID, senderUserID)
	}
	return r.findOrCreateConversationLegacy(ctx, userA, userB, senderOrgID)
}

// findOrCreateConversationWithRunner is the production path: opens a
// SERIALIZABLE transaction with the tenant context already set, then
// runs the find / insert pipeline against RLS-active tables.
func (r *ConversationRepository) findOrCreateConversationWithRunner(ctx context.Context, userA, userB, senderOrgID, senderUserID uuid.UUID) (uuid.UUID, bool, error) {
	var convID uuid.UUID
	var created bool

	err := r.txRunner.RunInTxWithTenantSerializable(ctx, senderOrgID, senderUserID, func(tx *sql.Tx) error {
		id, isNew, err := findOrCreateConversationInTx(ctx, tx, userA, userB, senderOrgID)
		if err != nil {
			return err
		}
		convID = id
		created = isNew
		return nil
	})
	if err != nil {
		if isSerializationError(err) {
			return r.retryFindConversation(ctx, userA, userB)
		}
		return uuid.UUID{}, false, err
	}
	return convID, created, nil
}

// findOrCreateConversationLegacy preserves the pre-RLS behavior for
// unit tests that build the repository without a TxRunner. Production
// callers MUST go through WithTxRunner so the tenant context is set.
func (r *ConversationRepository) findOrCreateConversationLegacy(ctx context.Context, userA, userB, senderOrgID uuid.UUID) (uuid.UUID, bool, error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return uuid.UUID{}, false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	convID, created, err := findOrCreateConversationInTx(ctx, tx, userA, userB, senderOrgID)
	if err != nil {
		return uuid.UUID{}, false, err
	}

	if err := tx.Commit(); err != nil {
		if isSerializationError(err) {
			return r.retryFindConversation(ctx, userA, userB)
		}
		return uuid.UUID{}, false, fmt.Errorf("commit: %w", err)
	}

	return convID, created, nil
}

// findOrCreateConversationInTx runs the find / insert pipeline inside an
// already-open transaction. Shared between the legacy and tenant-aware
// entry points so the SQL logic lives in a single place.
//
// The conversation row is inserted with organization_id ALREADY SET to
// the sender's org. This avoids the chicken-and-egg of the previous
// design where the row was inserted with NULL and backfilled in the
// same tx — under RLS the NULL row is rejected by the policy because
// `NULL = current_setting(...)` is NULL, not true.
func findOrCreateConversationInTx(ctx context.Context, tx *sql.Tx, userA, userB, senderOrgID uuid.UUID) (uuid.UUID, bool, error) {
	var convID uuid.UUID
	err := tx.QueryRowContext(ctx, queryFindExistingConversation, userA, userB).Scan(&convID)
	if err == nil {
		return convID, false, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return uuid.UUID{}, false, fmt.Errorf("find conversation: %w", err)
	}

	conv := message.NewConversation()

	// Insert the conversation with its organization_id already set
	// when the sender belongs to one. NULL is preserved for solo-
	// provider senders so the participant escape hatch on the RLS
	// policy still admits the row through app.current_user_id.
	var orgArg interface{}
	if senderOrgID != uuid.Nil {
		orgArg = senderOrgID
	} else {
		orgArg = nil
	}

	if _, err := tx.ExecContext(ctx, queryInsertConversationWithOrg, conv.ID, orgArg, conv.CreatedAt, conv.UpdatedAt); err != nil {
		return uuid.UUID{}, false, fmt.Errorf("insert conversation: %w", err)
	}

	now := time.Now()
	if _, err := tx.ExecContext(ctx, queryInsertParticipant, conv.ID, userA, now); err != nil {
		return uuid.UUID{}, false, fmt.Errorf("insert participant A: %w", err)
	}
	if _, err := tx.ExecContext(ctx, queryInsertParticipant, conv.ID, userB, now); err != nil {
		return uuid.UUID{}, false, fmt.Errorf("insert participant B: %w", err)
	}

	// Backfill organization_id from the participant set when the
	// caller did not supply one (system actor / solo provider). The
	// UPDATE is admitted by the participant escape hatch because we
	// just inserted both participants, and we set app.current_user_id
	// on the tx.
	if senderOrgID == uuid.Nil {
		if _, err := tx.ExecContext(ctx, queryBackfillConversationOrg, conv.ID); err != nil {
			return uuid.UUID{}, false, fmt.Errorf("backfill conversation org: %w", err)
		}
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
		rows, err = r.db.QueryContext(ctx, queryListConversationsFirst,
			params.OrganizationID, params.UserID, limit+1)
	} else {
		c, cErr := cursor.Decode(params.Cursor)
		if cErr != nil {
			return nil, "", fmt.Errorf("decode cursor: %w", cErr)
		}
		rows, err = r.db.QueryContext(ctx, queryListConversationsWithCursor,
			params.OrganizationID, params.UserID, c.CreatedAt, c.ID, limit+1)
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
			&s.OtherOrgID,
			&s.OtherOrgName,
			&s.OtherOrgType,
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

func (r *ConversationRepository) IsOrgAuthorizedForConversation(ctx context.Context, conversationID, orgID uuid.UUID) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var exists bool
	err := r.db.QueryRowContext(ctx, queryIsOrgAuthorizedForConversation, conversationID, orgID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check org authorized for conversation: %w", err)
	}
	return exists, nil
}

// CreateMessage inserts a message row, advances the conversation's
// sequence under FOR UPDATE, and bumps the conversation's updated_at.
//
// senderOrgID + senderUserID are the tenant context of the caller —
// not of the message itself. They are used to install
// app.current_org_id / app.current_user_id on the transaction so the
// FOR UPDATE on `conversations`, the MAX(seq) read on `messages`, the
// INSERT into `messages`, and the UPDATE on `conversations` all pass
// the RLS isolation policies installed by mig 125.
//
// uuid.Nil is acceptable for both arguments — system-actor paths
// (scheduler, end-of-project effects) have no caller. In that case
// the txRunner falls through to the legacy non-tenant path so
// nothing breaks in deployments where RLS is not yet active or the
// DB role still bypasses RLS.
func (r *ConversationRepository) CreateMessage(ctx context.Context, msg *message.Message, senderOrgID, senderUserID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	if r.txRunner != nil {
		return r.txRunner.RunInTxWithTenant(ctx, senderOrgID, senderUserID, func(tx *sql.Tx) error {
			return createMessageInTx(ctx, tx, msg)
		})
	}
	return r.createMessageLegacy(ctx, msg)
}

// createMessageLegacy preserves the pre-RLS code path so unit tests
// that build the repo without a TxRunner keep working unchanged.
func (r *ConversationRepository) createMessageLegacy(ctx context.Context, msg *message.Message) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := createMessageInTx(ctx, tx, msg); err != nil {
		return err
	}
	return tx.Commit()
}

// createMessageInTx is the SQL-only core of CreateMessage. Shared
// between the tenant-aware and legacy entry points.
func createMessageInTx(ctx context.Context, tx *sql.Tx, msg *message.Message) error {
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

	// System messages emitted by background paths (end-of-project
	// effects, dispute resolution, scheduler) carry uuid.Nil as the
	// sender. The messages.sender_id column is FK-constrained on
	// users(id), so binding the zero UUID would trip the foreign
	// key check and silently drop the row (the proposal service
	// ignores SendSystemMessage errors). Convert uuid.Nil → SQL NULL
	// so system-actor sends persist correctly.
	senderArg := senderForInsert(msg.SenderID)
	if _, err := tx.ExecContext(ctx, queryInsertMessage,
		msg.ID, msg.ConversationID, senderArg, msg.Content,
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

	return nil
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

func (r *ConversationRepository) IncrementUnreadForRecipients(ctx context.Context, conversationID, senderUserID, senderOrgID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryIncrementUnreadForRecipients, conversationID, senderUserID, senderOrgID)
	if err != nil {
		return fmt.Errorf("increment unread for recipients: %w", err)
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

func (r *ConversationRepository) GetOrgMemberRecipients(ctx context.Context, conversationID, excludeUserID uuid.UUID) ([]uuid.UUID, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	rows, err := r.db.QueryContext(ctx, queryGetOrgMemberRecipients, conversationID, excludeUserID)
	if err != nil {
		return nil, fmt.Errorf("get org member recipients: %w", err)
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan org member recipient: %w", err)
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
	var senderID *uuid.UUID
	var replyID, replySenderID *uuid.UUID
	var replyContent, replyType *string

	err := row.Scan(
		&msg.ID, &msg.ConversationID, &senderID, &msg.Content,
		&msgType, &metadata, &msg.ReplyToID, &msg.Seq, &status,
		&msg.EditedAt, &msg.DeletedAt, &msg.CreatedAt, &msg.UpdatedAt,
		&replyID, &replySenderID, &replyContent, &replyType,
	)
	if err != nil {
		return nil, err
	}

	msg.SenderID = senderForRead(senderID)
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
	var senderID *uuid.UUID
	var replyID, replySenderID *uuid.UUID
	var replyContent, replyType *string

	err := rows.Scan(
		&msg.ID, &msg.ConversationID, &senderID, &msg.Content,
		&msgType, &metadata, &msg.ReplyToID, &msg.Seq, &status,
		&msg.EditedAt, &msg.DeletedAt, &msg.CreatedAt, &msg.UpdatedAt,
		&replyID, &replySenderID, &replyContent, &replyType,
	)
	if err != nil {
		return nil, err
	}

	msg.SenderID = senderForRead(senderID)
	if len(metadata) > 0 {
		msg.Metadata = metadata
	}
	msg.Type = message.MessageType(msgType)
	msg.Status = message.MessageStatus(status)
	msg.ReplyPreview = buildReplyPreview(replyID, replySenderID, replyContent, replyType)
	return msg, nil
}

// senderForInsert converts a uuid.UUID into the value to bind for the
// messages.sender_id parameter. The system-actor sentinel uuid.Nil is
// rewritten to nil so the database stores SQL NULL instead — required
// because messages.sender_id has a foreign key into users(id) and the
// zero UUID has no matching row.
func senderForInsert(senderID uuid.UUID) any {
	if senderID == uuid.Nil {
		return nil
	}
	return senderID
}

// senderForRead converts a nullable sender_id read from the database
// back into a uuid.UUID value. NULL becomes uuid.Nil — the same
// system-actor sentinel the rest of the app uses.
func senderForRead(senderID *uuid.UUID) uuid.UUID {
	if senderID == nil {
		return uuid.Nil
	}
	return *senderID
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

// UpdateMessageModeration removed in Phase 7 — see moderation_results
// repository's Upsert / MarkReviewed instead.
