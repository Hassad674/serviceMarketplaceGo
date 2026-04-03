export type ConversationParticipant = {
  id: string
  display_name: string
  email: string
  role: string
}

export type AdminConversation = {
  id: string
  participants: ConversationParticipant[]
  message_count: number
  last_message: string | null
  last_message_at: string | null
  created_at: string
}

export type AdminMessage = {
  id: string
  conversation_id: string
  sender_id: string
  sender_name: string
  sender_role: string
  content: string
  type: string
  metadata?: Record<string, unknown>
  reply_to_id?: string
  created_at: string
}

export type ConversationListResponse = {
  data: AdminConversation[]
  next_cursor: string
  has_more: boolean
  total: number
}

export type MessageListResponse = {
  data: AdminMessage[]
  next_cursor: string
  has_more: boolean
}
