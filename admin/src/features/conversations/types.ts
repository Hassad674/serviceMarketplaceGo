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
  pending_report_count: number
  reported_message?: string
  created_at: string
}

export type ModerationLabel = {
  Name: string
  Score: number
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
  moderation_status: string
  moderation_score: number
  moderation_labels?: ModerationLabel[]
  created_at: string
}

export type ConversationListResponse = {
  data: AdminConversation[]
  next_cursor: string
  has_more: boolean
  total: number
  page: number
  total_pages: number
}

export type MessageListResponse = {
  data: AdminMessage[]
  next_cursor: string
  has_more: boolean
}
