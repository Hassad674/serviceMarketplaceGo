/**
 * Shared messaging wire types used across multiple features. The
 * `Conversation` type is the cross-feature surface — both the
 * `messaging` and `referral` features render conversation lists, so
 * the type lives here rather than inside `features/messaging/types`.
 *
 * Internal messaging-only types (Message, ProposalMessageMetadata,
 * etc.) stay scoped to the messaging feature.
 */

export type Conversation = {
  id: string
  other_user_id: string
  other_org_id: string
  other_org_name: string
  other_org_type: string
  other_photo_url: string
  last_message: string | null
  last_message_at: string | null
  unread_count: number
  last_message_seq: number
  online: boolean
}

export type ConversationListResponse = {
  data: Conversation[]
  next_cursor?: string
  has_more: boolean
}
