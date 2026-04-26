export type ModerationSource = "human_report" | "auto_media" | "auto_text"

export type ModerationContentType =
  | "report"
  | "message"
  | "review"
  | "media"
  // Phase 2 — content types added by the moderation extension. The
  // backend emits these strings verbatim from moderation_results
  // (see backend domain/moderation.ContentType*). The admin UI uses
  // them to pick a label, badge, and the click-through "Voir" URL.
  | "profile_about"
  | "profile_title"
  | "job_title"
  | "job_description"
  | "proposal_description"
  | "job_application_message"
  | "user_display_name"

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
