import { adminApi } from "@/shared/lib/api-client"
import type {
  AdminConversation,
  ConversationListResponse,
  MessageListResponse,
} from "../types"

export function listConversations(cursor?: string): Promise<ConversationListResponse> {
  const params = new URLSearchParams()
  if (cursor) params.set("cursor", cursor)
  params.set("limit", "20")
  const qs = params.toString()
  return adminApi<ConversationListResponse>(`/api/v1/admin/conversations${qs ? `?${qs}` : ""}`)
}

export type ConversationDetailResponse = {
  data: AdminConversation
}

export function getConversation(id: string): Promise<ConversationDetailResponse> {
  return adminApi<ConversationDetailResponse>(`/api/v1/admin/conversations/${id}`)
}

export function getConversationMessages(
  id: string,
  cursor?: string,
): Promise<MessageListResponse> {
  const params = new URLSearchParams()
  if (cursor) params.set("cursor", cursor)
  params.set("limit", "50")
  const qs = params.toString()
  return adminApi<MessageListResponse>(
    `/api/v1/admin/conversations/${id}/messages${qs ? `?${qs}` : ""}`,
  )
}
