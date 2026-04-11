package postgres

const queryFindExistingConversation = `
	SELECT cp1.conversation_id
	FROM conversation_participants cp1
	JOIN conversation_participants cp2 ON cp1.conversation_id = cp2.conversation_id
	WHERE cp1.user_id = $1 AND cp2.user_id = $2
	LIMIT 1`

const queryInsertConversation = `
	INSERT INTO conversations (id, created_at, updated_at)
	VALUES ($1, $2, $3)`

const queryInsertParticipant = `
	INSERT INTO conversation_participants (conversation_id, user_id, joined_at)
	VALUES ($1, $2, $3)`

// queryBackfillConversationOrg denormalizes the first org-owning participant
// onto the conversation row. Runs inside the same tx as the participant
// inserts, so the subquery always sees both participants.
//
// We resolve the org via organization_members (the source of truth) rather
// than users.organization_id, so this works regardless of whether the
// denormalized user column has been backfilled.
const queryBackfillConversationOrg = `
	UPDATE conversations
	SET organization_id = (
		SELECT om.organization_id
		FROM conversation_participants cp
		JOIN organization_members om ON om.user_id = cp.user_id
		WHERE cp.conversation_id = $1
		ORDER BY cp.user_id
		LIMIT 1
	)
	WHERE id = $1`

const queryGetConversation = `
	SELECT id, created_at, updated_at
	FROM conversations WHERE id = $1`

// Phase R4 — List conversations visible to an organization. Returns
// one row per conversation where at least one user belonging to the
// calling org participates; the "other side" is resolved as any
// participant whose current org differs from the caller's org.
//
// $1 = caller organization_id
// $2 = caller user_id (for the operator's personal unread count)
// $3 = limit
const queryListConversationsFirst = `
	SELECT
		c.id,
		COALESCE(other_org.id, '00000000-0000-0000-0000-000000000000'::uuid),
		COALESCE(other_org.name, ''),
		COALESCE(other_org.type, ''),
		COALESCE(p.photo_url, ''),
		lm.content,
		lm.created_at,
		COALESCE(lm.seq, 0),
		COALESCE(cp_me.unread_count, 0)
	FROM conversations c
	LEFT JOIN LATERAL (
		SELECT u.organization_id AS org_id
		FROM conversation_participants cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.conversation_id = c.id AND u.organization_id <> $1
		LIMIT 1
	) og ON TRUE
	LEFT JOIN organizations other_org ON other_org.id = og.org_id
	LEFT JOIN profiles p ON p.organization_id = other_org.id
	LEFT JOIN conversation_participants cp_me
		ON cp_me.conversation_id = c.id AND cp_me.user_id = $2
	LEFT JOIN LATERAL (
		SELECT content, created_at, seq
		FROM messages
		WHERE conversation_id = c.id
		ORDER BY seq DESC
		LIMIT 1
	) lm ON TRUE
	WHERE EXISTS (
		SELECT 1
		FROM conversation_participants cp_my
		JOIN users u_my ON u_my.id = cp_my.user_id
		WHERE cp_my.conversation_id = c.id AND u_my.organization_id = $1
	)
	ORDER BY c.updated_at DESC, c.id DESC
	LIMIT $3`

// $1 = caller organization_id, $2 = caller user_id,
// $3/$4 = cursor (updated_at, id), $5 = limit
const queryListConversationsWithCursor = `
	SELECT
		c.id,
		COALESCE(other_org.id, '00000000-0000-0000-0000-000000000000'::uuid),
		COALESCE(other_org.name, ''),
		COALESCE(other_org.type, ''),
		COALESCE(p.photo_url, ''),
		lm.content,
		lm.created_at,
		COALESCE(lm.seq, 0),
		COALESCE(cp_me.unread_count, 0)
	FROM conversations c
	LEFT JOIN LATERAL (
		SELECT u.organization_id AS org_id
		FROM conversation_participants cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.conversation_id = c.id AND u.organization_id <> $1
		LIMIT 1
	) og ON TRUE
	LEFT JOIN organizations other_org ON other_org.id = og.org_id
	LEFT JOIN profiles p ON p.organization_id = other_org.id
	LEFT JOIN conversation_participants cp_me
		ON cp_me.conversation_id = c.id AND cp_me.user_id = $2
	LEFT JOIN LATERAL (
		SELECT content, created_at, seq
		FROM messages
		WHERE conversation_id = c.id
		ORDER BY seq DESC
		LIMIT 1
	) lm ON TRUE
	WHERE EXISTS (
		SELECT 1
		FROM conversation_participants cp_my
		JOIN users u_my ON u_my.id = cp_my.user_id
		WHERE cp_my.conversation_id = c.id AND u_my.organization_id = $1
	)
		AND (c.updated_at, c.id) < ($3, $4)
	ORDER BY c.updated_at DESC, c.id DESC
	LIMIT $5`

const queryIsParticipant = `
	SELECT EXISTS(
		SELECT 1 FROM conversation_participants
		WHERE conversation_id = $1 AND user_id = $2
	)`

const queryLockConversation = `
	SELECT id FROM conversations WHERE id = $1 FOR UPDATE`

const queryNextSeq = `
	SELECT COALESCE(MAX(seq), 0) + 1
	FROM messages
	WHERE conversation_id = $1`

const queryInsertMessage = `
	INSERT INTO messages (id, conversation_id, sender_id, content, msg_type, metadata, reply_to_id, seq, status, created_at, updated_at)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

const queryUpdateConversationTimestamp = `
	UPDATE conversations SET updated_at = $2 WHERE id = $1`

const queryGetMessage = `
	SELECT m.id, m.conversation_id, m.sender_id, m.content, m.msg_type, m.metadata,
		m.reply_to_id, m.seq, m.status, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
		r.id, r.sender_id, r.content, r.msg_type
	FROM messages m
	LEFT JOIN messages r ON r.id = m.reply_to_id
	WHERE m.id = $1`

const queryListMessagesFirst = `
	SELECT m.id, m.conversation_id, m.sender_id, m.content, m.msg_type, m.metadata,
		m.reply_to_id, m.seq, m.status, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
		r.id, r.sender_id, r.content, r.msg_type
	FROM messages m
	LEFT JOIN messages r ON r.id = m.reply_to_id
	WHERE m.conversation_id = $1
	ORDER BY m.created_at DESC, m.id DESC
	LIMIT $2`

const queryListMessagesWithCursor = `
	SELECT m.id, m.conversation_id, m.sender_id, m.content, m.msg_type, m.metadata,
		m.reply_to_id, m.seq, m.status, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
		r.id, r.sender_id, r.content, r.msg_type
	FROM messages m
	LEFT JOIN messages r ON r.id = m.reply_to_id
	WHERE m.conversation_id = $1
		AND (m.created_at, m.id) < ($2, $3)
	ORDER BY m.created_at DESC, m.id DESC
	LIMIT $4`

const queryMessagesSinceSeq = `
	SELECT m.id, m.conversation_id, m.sender_id, m.content, m.msg_type, m.metadata,
		m.reply_to_id, m.seq, m.status, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
		r.id, r.sender_id, r.content, r.msg_type
	FROM messages m
	LEFT JOIN messages r ON r.id = m.reply_to_id
	WHERE m.conversation_id = $1 AND m.seq > $2
	ORDER BY m.seq ASC
	LIMIT $3`

// queryListMessagesSinceTime returns messages of a conversation in
// chronological order, starting from the given timestamp. Used by the
// dispute AI summary which only feeds Claude messages exchanged after
// the mission actually started (i.e. after payment was held in escrow).
const queryListMessagesSinceTime = `
	SELECT m.id, m.conversation_id, m.sender_id, m.content, m.msg_type, m.metadata,
		m.reply_to_id, m.seq, m.status, m.edited_at, m.deleted_at, m.created_at, m.updated_at,
		r.id, r.sender_id, r.content, r.msg_type
	FROM messages m
	LEFT JOIN messages r ON r.id = m.reply_to_id
	WHERE m.conversation_id = $1 AND m.created_at >= $2
	ORDER BY m.created_at ASC, m.id ASC
	LIMIT $3`

const queryUpdateMessage = `
	UPDATE messages
	SET content = $2, edited_at = $3, deleted_at = $4, updated_at = $5
	WHERE id = $1`

const queryIncrementUnread = `
	UPDATE conversation_participants
	SET unread_count = unread_count + 1
	WHERE conversation_id = $1 AND user_id != $2`

const queryMarkAsRead = `
	UPDATE conversation_participants
	SET unread_count = 0, last_read_seq = $3
	WHERE conversation_id = $1 AND user_id = $2`

const queryGetTotalUnread = `
	SELECT COALESCE(SUM(unread_count), 0)
	FROM conversation_participants
	WHERE user_id = $1`

const queryGetTotalUnreadBatch = `
	SELECT user_id, COALESCE(SUM(unread_count), 0)
	FROM conversation_participants
	WHERE user_id = ANY($1)
	GROUP BY user_id`

const queryGetParticipantIDs = `
	SELECT user_id
	FROM conversation_participants
	WHERE conversation_id = $1`

const queryGetContactIDs = `
	SELECT DISTINCT cp_other.user_id
	FROM conversation_participants cp_me
	JOIN conversation_participants cp_other
		ON cp_other.conversation_id = cp_me.conversation_id
		AND cp_other.user_id != $1
	WHERE cp_me.user_id = $1`

const queryUpdateMessageStatus = `
	UPDATE messages SET status = $2, updated_at = now() WHERE id = $1`

const queryMarkMessagesAsRead = `
	UPDATE messages
	SET status = 'read', updated_at = now()
	WHERE conversation_id = $1
		AND sender_id != $2
		AND seq <= $3
		AND status != 'read'`

const querySaveMessageHistory = `
	INSERT INTO message_history (id, message_id, content, action, performed_by, created_at)
	VALUES (gen_random_uuid(), $1, $2, $3, $4, now())`

const queryUpdateMessageModeration = `
	UPDATE messages
	SET moderation_status = $2, moderation_score = $3, moderation_labels = $4, updated_at = now()
	WHERE id = $1`
