export type NotificationType =
  | "proposal_received"
  | "proposal_accepted"
  | "proposal_declined"
  | "proposal_modified"
  | "proposal_paid"
  | "completion_requested"
  | "proposal_completed"
  | "review_received"
  | "new_message"
  | "system_announcement"

export type Notification = {
  id: string
  user_id: string
  type: NotificationType
  title: string
  body: string
  data: Record<string, unknown>
  read_at: string | null
  created_at: string
}

export type NotificationPreference = {
  type: NotificationType
  in_app: boolean
  push: boolean
  email: boolean
}

export type NotificationListResponse = {
  data: Notification[]
  next_cursor: string
  has_more: boolean
}

export type UnreadCountResponse = {
  data: { count: number }
}
