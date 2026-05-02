/**
 * Shared messaging query keys.
 *
 * These keys are used cross-feature: the `proposal` feature
 * invalidates `conversationsQueryKey` and `messagesQueryKey` after
 * proposal-related mutations push new system messages into a
 * conversation. Lifted out of `features/messaging/hooks/...` so the
 * proposal feature does not have to import from messaging directly.
 *
 * The messaging feature also imports from here — single source of
 * truth for the key shape.
 */

export function conversationsQueryKey(uid: string | undefined) {
  return ["user", uid, "messaging", "conversations"] as const
}

export const MESSAGES_KEY_BASE = "messaging-messages"

export function messagesQueryKey(
  uid: string | undefined,
  conversationId: string | null,
) {
  return ["user", uid, MESSAGES_KEY_BASE, conversationId] as const
}
