import { apiClient } from "@/shared/lib/api-client"
import type { Get, Post, Put, Void } from "@/shared/lib/api-paths"
import type {
  MessageListResponse,
  StartConversationResponse,
  Message,
  PresignedURLResponse,
  UnreadCountResponse,
} from "../types"

// `listConversations` is shared with the `referral` feature (P9). Lives
// in `@/shared/lib/messaging/conversations-api` and is re-exported here
// so existing intra-feature imports keep working.
export { listConversations } from "@/shared/lib/messaging/conversations-api"

export function listMessages(conversationId: string, cursor?: string): Promise<MessageListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<Get<"/api/v1/messaging/conversations/{id}/messages"> & MessageListResponse>(
    `/api/v1/messaging/conversations/${conversationId}/messages${params}`,
  )
}

export type FileMessageMetadata = {
  url: string
  filename: string
  size: number
  mime_type: string
}

export type VoiceMessageMetadata = {
  url: string
  duration: number
  size: number
  mime_type: string
}

export function sendMessage(
  conversationId: string,
  content: string,
  type: "text" | "file" | "voice" = "text",
  metadata?: FileMessageMetadata | VoiceMessageMetadata,
  replyToId?: string,
): Promise<Message> {
  return apiClient<Post<"/api/v1/messaging/conversations/{id}/messages"> & Message>(
    `/api/v1/messaging/conversations/${conversationId}/messages`,
    {
      method: "POST",
      body: { content, type, metadata, reply_to_id: replyToId },
    },
  )
}

export function startConversation(otherOrgId: string, content: string): Promise<StartConversationResponse> {
  return apiClient<Post<"/api/v1/messaging/conversations"> & StartConversationResponse>("/api/v1/messaging/conversations", {
    method: "POST",
    body: { recipient_org_id: otherOrgId, content },
  })
}

export function markAsRead(conversationId: string, seq?: number): Promise<void> {
  return apiClient<Void<"/api/v1/messaging/conversations/{id}/read">>(
    `/api/v1/messaging/conversations/${conversationId}/read`,
    { method: "POST", body: { seq: seq ?? 0 } },
  )
}

export function editMessage(messageId: string, content: string): Promise<Message> {
  return apiClient<Put<"/api/v1/messaging/messages/{id}"> & Message>(`/api/v1/messaging/messages/${messageId}`, {
    method: "PUT",
    body: { content },
  })
}

export function deleteMessage(messageId: string): Promise<void> {
  return apiClient<Void<"/api/v1/messaging/messages/{id}">>(`/api/v1/messaging/messages/${messageId}`, {
    method: "DELETE",
  })
}

export function getPresignedURL(filename: string, contentType: string): Promise<PresignedURLResponse> {
  return apiClient<Post<"/api/v1/messaging/upload-url"> & PresignedURLResponse>("/api/v1/messaging/upload-url", {
    method: "POST",
    body: { filename, content_type: contentType },
  })
}

export function getUnreadCount(): Promise<UnreadCountResponse> {
  return apiClient<Get<"/api/v1/messaging/unread-count"> & UnreadCountResponse>("/api/v1/messaging/unread-count")
}
