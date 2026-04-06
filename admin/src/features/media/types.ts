export type ModerationLabel = {
  name: string
  confidence: number
  parent_name?: string
}

export type AdminMedia = {
  id: string
  uploader_id: string
  file_url: string
  file_name: string
  file_type: string
  file_size: number
  context: string
  context_id?: string
  moderation_status: "pending" | "approved" | "flagged" | "rejected"
  moderation_labels: ModerationLabel[]
  moderation_score: number
  reviewed_at?: string
  reviewed_by?: string
  created_at: string
  updated_at: string
  uploader_display_name: string
  uploader_email: string
  uploader_role: string
}

export type AdminMediaListResponse = {
  data: AdminMedia[]
  total: number
  page: number
  total_pages: number
}

export type AdminMediaDetailResponse = {
  data: AdminMedia
}

export type MediaFilters = {
  status: string
  type: string
  context: string
  search: string
  sort: string
  page: number
}
