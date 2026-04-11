export type MessageType =
  | "text"
  | "file"
  | "voice"
  | "proposal_sent"
  | "proposal_accepted"
  | "proposal_declined"
  | "proposal_modified"
  | "proposal_paid"
  | "proposal_payment_requested"
  | "proposal_completion_requested"
  | "proposal_completed"
  | "proposal_completion_rejected"
  | "evaluation_request"
  | "call_ended"
  | "call_missed"
  | "dispute_opened"
  | "dispute_counter_proposal"
  | "dispute_counter_accepted"
  | "dispute_counter_rejected"
  | "dispute_escalated"
  | "dispute_resolved"
  | "dispute_cancelled"
  | "dispute_auto_resolved"
  | "dispute_cancellation_requested"
  | "dispute_cancellation_refused"

export type MessageStatus = "sending" | "sent" | "delivered" | "read"

export type FileMetadata = {
  url: string
  filename: string
  size: number
  mime_type: string
}

export type VoiceMetadata = {
  url: string
  duration: number
  size: number
  mime_type: string
}

export type ProposalMessageMetadata = {
  proposal_id: string
  proposal_title: string
  proposal_amount: number
  proposal_status: "pending" | "accepted" | "declined" | "withdrawn" | "paid" | "active" | "completion_requested" | "completed"
  proposal_deadline: string | null
  proposal_sender_name: string
  proposal_documents_count: number
  proposal_version: number
  proposal_parent_id: string | null
  proposal_client_id: string
  proposal_provider_id: string
  target_user_id?: string
}

export type ReplyToInfo = {
  id: string
  sender_id: string
  content: string
  type: string
}

export type Message = {
  id: string
  conversation_id: string
  sender_id: string
  content: string
  type: MessageType
  metadata: FileMetadata | VoiceMetadata | ProposalMessageMetadata | null
  reply_to?: ReplyToInfo | null
  seq: number
  status: MessageStatus
  edited_at: string | null
  deleted_at: string | null
  created_at: string
}

// A conversation is now identified by the "other organization" on the
// thread, not a specific user. Every operator of the sender's org
// sees the same thread, and it targets whichever operator of the
// recipient org is on call — the Stripe Dashboard inbox model.
export type Conversation = {
  id: string
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
