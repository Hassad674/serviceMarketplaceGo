export type ConversationRole = "freelancer" | "agency" | "enterprise"

export type Conversation = {
  id: string
  name: string
  role: ConversationRole
  lastMessage: string | null
  lastMessageAt: string | null
  avatar: string | null
  unread: number
  online: boolean
}

export type Message = {
  id: string
  conversationId: string
  senderId: string
  content: string
  sentAt: string
  isOwn: boolean
}
