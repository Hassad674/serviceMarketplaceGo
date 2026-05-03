package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"marketplace-backend/internal/domain/message"
	"marketplace-backend/internal/port/repository"
	"marketplace-backend/pkg/cursor"
)

// CreateMessage inserts a message INSIDE a tenant-aware transaction so
// the messages policy admits the row under prod NOSUPERUSER NOBYPASSRLS.
//
// The senderOrgID + senderUserID are installed via SET LOCAL before the
// INSERT so the policy admits the row through either the org-side OR
// the participant escape hatch.
//
// Some callers (system-actor sends fired by background paths) pass
// uuid.Nil for both ids — in that case the txRunner falls through to
// the legacy non-tenant path so nothing breaks in deployments where
// RLS is not yet active or the DB role still bypasses RLS.
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

	// P6: maintain conversations.last_message_* in the SAME tx as the
	// INSERT above so /api/v1/messaging/conversations can read the
	// preview without a per-row LATERAL subquery on `messages`.
	// senderArg is reused — system messages bind NULL exactly like
	// the messages.sender_id column does (mig 130).
	if _, err := tx.ExecContext(ctx, queryUpdateConversationLastMessage,
		msg.ConversationID, msg.CreatedAt, msg.Seq, msg.Content, senderArg,
	); err != nil {
		return fmt.Errorf("update conversation last_message: %w", err)
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

// ListMessages is wrapped in RunInTxWithTenant when a TxRunner is wired
// AND params carries non-zero caller ids (BUG-NEW-04 path 8/8). The
// messages policy (migration 125) admits the row when its parent
// conversation matches either app.current_org_id (organization side)
// or app.current_user_id (participant escape hatch). Setting both
// covers solo-provider conversations AND org-side reads from a single
// call site.
//
// When caller ids are uuid.Nil OR no runner is wired, the call falls
// back to the legacy direct-db path — preserved for unit tests that
// build the repo with only a *sql.DB.
func (r *ConversationRepository) ListMessages(ctx context.Context, params repository.ListMessagesParams) ([]*message.Message, string, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	limit := params.Limit
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	var results []*message.Message
	var nextCursor string

	doQuery := func(runner sqlQuerier) error {
		var rows *sql.Rows
		var err error
		if params.Cursor == "" {
			rows, err = runner.QueryContext(ctx, queryListMessagesFirst, params.ConversationID, limit+1)
		} else {
			c, cErr := cursor.Decode(params.Cursor)
			if cErr != nil {
				return fmt.Errorf("decode cursor: %w", cErr)
			}
			rows, err = runner.QueryContext(ctx, queryListMessagesWithCursor,
				params.ConversationID, c.CreatedAt, c.ID, limit+1)
		}
		if err != nil {
			return fmt.Errorf("list messages: %w", err)
		}
		defer rows.Close()
		out, nc, err := scanMessageList(rows, limit)
		if err != nil {
			return err
		}
		results = out
		nextCursor = nc
		return nil
	}

	useTenantTx := r.txRunner != nil && (params.CallerOrgID != uuid.Nil || params.CallerUserID != uuid.Nil)
	if useTenantTx {
		err := r.txRunner.RunInTxWithTenant(ctx, params.CallerOrgID, params.CallerUserID, func(tx *sql.Tx) error {
			return doQuery(tx)
		})
		if err != nil {
			return nil, "", err
		}
		return results, nextCursor, nil
	}

	if err := doQuery(r.db); err != nil {
		return nil, "", err
	}
	return results, nextCursor, nil
}

// GetMessageForCaller returns a single message under the caller's
// tenant context. The caller's orgID + userID are installed before
// the SELECT so the messages policy admits the row.
func (r *ConversationRepository) GetMessageForCaller(ctx context.Context, id, callerOrgID, callerUserID uuid.UUID) (*message.Message, error) {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	var msg *message.Message
	doRead := func(runner sqlQuerier) error {
		got, err := scanMessage(runner.QueryRowContext(ctx, queryGetMessage, id))
		if errors.Is(err, sql.ErrNoRows) {
			return message.ErrMessageNotFound
		}
		if err != nil {
			return fmt.Errorf("get message for caller: %w", err)
		}
		msg = got
		return nil
	}

	if r.txRunner != nil {
		err := r.txRunner.RunInTxWithTenant(ctx, callerOrgID, callerUserID, func(tx *sql.Tx) error {
			return doRead(tx)
		})
		if err != nil {
			return nil, err
		}
		return msg, nil
	}

	if err := doRead(r.db); err != nil {
		return nil, err
	}
	return msg, nil
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

func (r *ConversationRepository) UpdateMessageStatus(ctx context.Context, messageID uuid.UUID, status message.MessageStatus) error {
	ctx, cancel := context.WithTimeout(ctx, queryTimeout)
	defer cancel()

	_, err := r.db.ExecContext(ctx, queryUpdateMessageStatus, messageID, string(status))
	if err != nil {
		return fmt.Errorf("update message status: %w", err)
	}

	return nil
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
