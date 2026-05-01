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

// queryInsertConversationWithOrg inserts a conversation row with its
// organization_id already populated when known. Used by the
// tenant-aware FindOrCreateConversation path so the row satisfies the
// RLS USING expression `organization_id = current_setting('app.current_org_id')`
// at INSERT time — the previous NULL-then-backfill design was rejected
// by RLS because `NULL = orgID` is NULL (not true) and the participant
// escape hatch had nothing to match yet.
//
// $1 = conversation id
// $2 = organization id (NULL when sender is a solo provider)
// $3 = created_at
// $4 = updated_at
const queryInsertConversationWithOrg = `
	INSERT INTO conversations (id, organization_id, created_at, updated_at)
	VALUES ($1, $2, $3, $4)`

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
// Since phase R11 the caller's personal unread count is read from
// conversation_read_state (per-user, lazily created) instead of the
// old conversation_participants columns — so an operator that joined
// the org AFTER the conversation was opened still sees 0 unread
// until they actually receive a message and the fan-out creates a
// row for them.
//
// We surface BOTH the other participant's user id (still needed by
// proposal + call flows that anchor on user ids) and the other org's
// metadata (used for display in the list).
//
// $1 = caller organization_id
// $2 = caller user_id (for the operator's personal unread count)
// $3 = limit
const queryListConversationsFirst = `
	SELECT
		c.id,
		COALESCE(og.user_id, '00000000-0000-0000-0000-000000000000'::uuid),
		COALESCE(other_org.id, '00000000-0000-0000-0000-000000000000'::uuid),
		COALESCE(other_org.name, ''),
		COALESCE(other_org.type, ''),
		COALESCE(p.photo_url, ''),
		lm.content,
		lm.created_at,
		COALESCE(lm.seq, 0),
		COALESCE(crs.unread_count, 0)
	FROM conversations c
	LEFT JOIN LATERAL (
		SELECT u.id AS user_id, u.organization_id AS org_id
		FROM conversation_participants cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.conversation_id = c.id AND u.organization_id <> $1
		LIMIT 1
	) og ON TRUE
	LEFT JOIN organizations other_org ON other_org.id = og.org_id
	LEFT JOIN profiles p ON p.organization_id = other_org.id
	LEFT JOIN conversation_read_state crs
		ON crs.conversation_id = c.id AND crs.user_id = $2
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
		COALESCE(og.user_id, '00000000-0000-0000-0000-000000000000'::uuid),
		COALESCE(other_org.id, '00000000-0000-0000-0000-000000000000'::uuid),
		COALESCE(other_org.name, ''),
		COALESCE(other_org.type, ''),
		COALESCE(p.photo_url, ''),
		lm.content,
		lm.created_at,
		COALESCE(lm.seq, 0),
		COALESCE(crs.unread_count, 0)
	FROM conversations c
	LEFT JOIN LATERAL (
		SELECT u.id AS user_id, u.organization_id AS org_id
		FROM conversation_participants cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.conversation_id = c.id AND u.organization_id <> $1
		LIMIT 1
	) og ON TRUE
	LEFT JOIN organizations other_org ON other_org.id = og.org_id
	LEFT JOIN profiles p ON p.organization_id = other_org.id
	LEFT JOIN conversation_read_state crs
		ON crs.conversation_id = c.id AND crs.user_id = $2
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

// queryIsOrgAuthorizedForConversation — phase R11 authorization guard.
// Returns true when the caller's organization has at least one user in
// the direct-participant set of the conversation. This is what allows
// an operator who joined the team after the conversation was opened to
// read/write in it.
//
// $1 = conversation_id, $2 = caller organization_id
const queryIsOrgAuthorizedForConversation = `
	SELECT EXISTS(
		SELECT 1
		FROM conversation_participants cp
		JOIN users u ON u.id = cp.user_id
		WHERE cp.conversation_id = $1 AND u.organization_id = $2
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

// queryIncrementUnreadForRecipients — phase R11 fan-out.
//
// Fan out a +1 unread bump to every user belonging to any organization
// that has a direct participant in the conversation, EXCEPT users in
// the sender's own org. The row is lazily inserted if missing, so an
// operator who joined after the conversation was opened still gets a
// live bump on their next list call.
//
// $1 = conversation_id
// $2 = sender user_id (belt-and-braces self-exclude)
// $3 = sender organization_id (team-wide self-exclude)
//
// The explicit ::uuid cast on $1 is required because Postgres cannot
// deduce the parameter type when the same placeholder appears both in
// the SELECT list (as a literal column value) and in the JOIN/WHERE
// clause (as a foreign key comparison).
const queryIncrementUnreadForRecipients = `
	INSERT INTO conversation_read_state (user_id, conversation_id, last_read_seq, unread_count)
	SELECT DISTINCT u.id, $1::uuid, 0, 1
	FROM conversation_participants cp
	JOIN users u_part ON u_part.id = cp.user_id
	JOIN users u ON u.organization_id = u_part.organization_id
	WHERE cp.conversation_id = $1::uuid
	  AND u_part.organization_id <> $3::uuid
	  AND u.id <> $2::uuid
	ON CONFLICT (user_id, conversation_id) DO UPDATE
	  SET unread_count = conversation_read_state.unread_count + 1,
	      updated_at   = now()`

// queryMarkAsRead — phase R11 upsert.
//
// Upsert the caller's read-state row to unread_count = 0 and bump
// last_read_seq forward only (never backward — a caller that races
// with their own earlier MarkAsRead must not regress the seq).
//
// $1 = conversation_id, $2 = user_id, $3 = seq
const queryMarkAsRead = `
	INSERT INTO conversation_read_state (user_id, conversation_id, last_read_seq, unread_count, created_at, updated_at)
	VALUES ($2, $1, $3, 0, now(), now())
	ON CONFLICT (user_id, conversation_id) DO UPDATE
	  SET last_read_seq = GREATEST(conversation_read_state.last_read_seq, EXCLUDED.last_read_seq),
	      unread_count  = 0,
	      updated_at    = now()`

const queryGetTotalUnread = `
	SELECT COALESCE(SUM(unread_count), 0)
	FROM conversation_read_state
	WHERE user_id = $1`

const queryGetTotalUnreadBatch = `
	SELECT user_id, COALESCE(SUM(unread_count), 0)
	FROM conversation_read_state
	WHERE user_id = ANY($1)
	GROUP BY user_id`

const queryGetParticipantIDs = `
	SELECT user_id
	FROM conversation_participants
	WHERE conversation_id = $1`

// queryGetOrgMemberRecipients — phase R11 broadcast fan-out.
//
// Returns every user id belonging to any organization that has a
// direct participant in the given conversation, excluding the given
// user (typically the sender). Used by broadcasters (WS, push) so
// every operator on both sides of the conversation sees the event.
//
// $1 = conversation_id, $2 = user_id to exclude
const queryGetOrgMemberRecipients = `
	SELECT DISTINCT u.id
	FROM conversation_participants cp
	JOIN users u_part ON u_part.id = cp.user_id
	JOIN users u ON u.organization_id = u_part.organization_id
	WHERE cp.conversation_id = $1 AND u.id <> $2`

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

// queryUpdateMessageModeration removed in Phase 7. The moderation
// pipeline writes to moderation_results exclusively now.
