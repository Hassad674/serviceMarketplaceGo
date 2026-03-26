export type MessageType = "text" | "file" | "proposal_sent"

export type MessageStatus = "sending" | "sent" | "delivered" | "read"

export type FileMetadata = {
  url: string
  filename: string
  size: number
  mime_type: string
}

export type ProposalMessageMetadata = {
  proposal_id: string
  proposal_title: string
  proposal_total_amount: number
  proposal_payment_type: "escrow" | "invoice"
  proposal_milestones_count: number
  proposal_status: "pending" | "accepted" | "declined" | "withdrawn"
  proposal_sender_name: string
}

export type Message = {
  id: string
  conversation_id: string
  sender_id: string
  content: string
  type: MessageType
  metadata: FileMetadata | ProposalMessageMetadata | null
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
  last_message_seq: number
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
  public_url: string
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

// Server sends Envelope: { type: string, payload: any }
export type WSServerFrame =
  | { type: "new_message"; payload: Message }
  | { type: "typing"; payload: { conversation_id: string; user_id: string } }
  | { type: "status_update"; payload: { conversation_id: string; reader_id: string; up_to_seq: number; status: "delivered" | "read" } }
  | { type: "unread_count"; payload: { count: number } }
  | { type: "message_edited"; payload: Message }
  | { type: "message_deleted"; payload: { message_id: string; conversation_id: string } }
  | { type: "presence"; payload: { user_id: string; online: boolean } }
