export type MessageType = "text" | "file"

export type MessageStatus = "sending" | "sent" | "delivered" | "read"

export type FileMetadata = {
  url: string
  filename: string
  size: number
  mime_type: string
}

export type Message = {
  id: string
  conversation_id: string
  sender_id: string
  content: string
  type: MessageType
  metadata: FileMetadata | null
  seq: number
  status: MessageStatus
  edited_at: string | null
  deleted_at: string | null
  created_at: string
}

export type Conversation = {
  id: string
  other_user_id: string
  other_user_name: string
  other_user_role: string
  other_photo_url: string
  last_message: string | null
  last_message_at: string | null
  unread_count: number
  last_seq: number
  online: boolean
}

export type ConversationListResponse = {
  data: Conversation[]
  next_cursor?: string
  has_more: boolean
}

export type MessageListResponse = {
  data: Message[]
  next_cursor?: string
  has_more: boolean
}

export type StartConversationResponse = {
  conversation_id: string
  message: Message
}

export type PresignedURLResponse = {
  upload_url: string
  file_key: string
}

export type UnreadCountResponse = {
  count: number
}

// WebSocket frame types
export type WSClientFrame =
  | { type: "heartbeat" }
  | { type: "typing"; conversation_id: string }
  | { type: "ack"; message_id: string }
  | { type: "sync"; conversations: Record<string, number> }

export type WSServerFrame =
  | { type: "new_message"; payload: Message }
  | { type: "typing"; conversation_id: string; user_id: string }
  | { type: "status_update"; message_id: string; status: "delivered" | "read" }
  | { type: "unread_count"; count: number }
  | { type: "message_edited"; payload: Message }
  | { type: "message_deleted"; message_id: string; conversation_id: string }
