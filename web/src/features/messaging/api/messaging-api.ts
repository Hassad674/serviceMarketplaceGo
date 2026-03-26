import { apiClient } from "@/shared/lib/api-client"
import type {
  ConversationListResponse,
  MessageListResponse,
  StartConversationResponse,
  Message,
  PresignedURLResponse,
  UnreadCountResponse,
} from "../types"

export function listConversations(cursor?: string): Promise<ConversationListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<ConversationListResponse>(`/api/v1/messaging/conversations${params}`)
}

export function listMessages(conversationId: string, cursor?: string): Promise<MessageListResponse> {
  const params = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ""
  return apiClient<MessageListResponse>(
    `/api/v1/messaging/conversations/${conversationId}/messages${params}`,
  )
}

export function sendMessage(conversationId: string, content: string, type: "text" | "file" = "text", metadata?: { file_key: string; filename: string; size: number; mime_type: string }): Promise<Message> {
  return apiClient<Message>(
    `/api/v1/messaging/conversations/${conversationId}/messages`,
    {
      method: "POST",
      body: { content, type, metadata },
    },
  )
}

export function startConversation(otherUserId: string, content: string): Promise<StartConversationResponse> {
  return apiClient<StartConversationResponse>("/api/v1/messaging/conversations", {
    method: "POST",
    body: { recipient_id: otherUserId, content },
  })
}

export function markAsRead(conversationId: string, seq?: number): Promise<void> {
  return apiClient<void>(
    `/api/v1/messaging/conversations/${conversationId}/read`,
    { method: "POST", body: { seq: seq ?? 0 } },
  )
}

export function editMessage(messageId: string, content: string): Promise<Message> {
  return apiClient<Message>(`/api/v1/messaging/messages/${messageId}`, {
    method: "PUT",
    body: { content },
  })
}

export function deleteMessage(messageId: string): Promise<void> {
  return apiClient<void>(`/api/v1/messaging/messages/${messageId}`, {
    method: "DELETE",
  })
}

export function getPresignedURL(filename: string, mimeType: string): Promise<PresignedURLResponse> {
  return apiClient<PresignedURLResponse>("/api/v1/messaging/upload-url", {
    method: "POST",
    body: { filename, mime_type: mimeType },
  })
}

export function getUnreadCount(): Promise<UnreadCountResponse> {
  return apiClient<UnreadCountResponse>("/api/v1/messaging/unread-count")
}
