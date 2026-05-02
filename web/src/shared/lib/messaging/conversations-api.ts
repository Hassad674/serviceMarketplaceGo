import { apiClient } from "@/shared/lib/api-client"
import type { ConversationListResponse } from "@/shared/types/messaging"

/**
 * `GET /api/v1/messaging/conversations` — paginated.
 *
 * Lifted out of `features/messaging/api/messaging-api` so the
 * `referral` feature's pickers can list conversations without
 * importing from the messaging feature directly. The messaging
 * feature also imports this function from here.
 */
export function listConversations(cursor?: string): Promise<ConversationListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<ConversationListResponse>(`/api/v1/messaging/conversations${params}`)
}
