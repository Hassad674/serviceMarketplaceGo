import { adminApi } from "@/shared/lib/api-client"
import type {
  AdminConversation,
  ConversationListResponse,
  MessageListResponse,
} from "../types"

type ListConversationsParams = {
  page?: number
  sort?: string
  filter?: string
}

export function listConversations(params: ListConversationsParams): Promise<ConversationListResponse> {
  const qs = new URLSearchParams()
  if (params.page && params.page > 0) qs.set("page", String(params.page))
  if (params.sort) qs.set("sort", params.sort)
  if (params.filter) qs.set("filter", params.filter)
  qs.set("limit", "20")
  const str = qs.toString()
  return adminApi<ConversationListResponse>(`/api/v1/admin/conversations${str ? `?${str}` : ""}`)
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
