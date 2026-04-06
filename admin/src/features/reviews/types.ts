export type AdminReviewUserBrief = {
  id: string
  display_name: string
  email: string
  role: string
}

export type AdminReview = {
  id: string
  proposal_id: string
  global_rating: number
  timeliness?: number
  communication?: number
  quality?: number
  comment: string
  video_url?: string
  created_at: string
  updated_at: string
  pending_report_count: number
  reviewer: AdminReviewUserBrief
  reviewed: AdminReviewUserBrief
}

export type AdminReviewListResponse = {
  data: AdminReview[]
  total: number
  page: number
  total_pages: number
}

export type AdminReviewDetailResponse = {
  data: AdminReview
}

export type ReviewFilters = {
  search: string
  rating: string
  sort: string
  filter: string
  page: number
}
