export type ModerationSource = "human_report" | "auto_media" | "auto_text"

export type ModerationContentType = "report" | "message" | "review" | "media"

export type ModerationItem = {
  id: string
  source: ModerationSource
  content_type: ModerationContentType
  content_id: string
  content_preview: string
  content_url: string
  status: string
  moderation_score: number
  reason: string
  user_involved: {
    id: string
    display_name: string
    role: string
  }
  conversation_id?: string
  created_at: string
}

export type ModerationListResponse = {
  data: ModerationItem[]
  total: number
  page: number
  total_pages: number
}

export type ModerationCountResponse = {
  count: number
}

export type ModerationFilters = {
  source: string
  type: string
  status: string
  sort: string
  page: number
}
