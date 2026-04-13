/**
 * Shared Review type used across multiple features.
 *
 * The single source of truth for the review data shape. Features like
 * `review` and `provider` (for project history) both import from here
 * to avoid cross-feature imports.
 */
export type ReviewSide = "client_to_provider" | "provider_to_client"

export type Review = {
  id: string
  proposal_id: string
  reviewer_id: string
  reviewed_id: string
  global_rating: number
  timeliness: number | null
  communication: number | null
  quality: number | null
  comment: string
  video_url: string | null
  title_visible: boolean
  // Which direction of the double-blind review this row represents.
  // "client_to_provider" carries the optional detailed sub-criteria
  // (timeliness, communication, quality); "provider_to_client" never does.
  side: ReviewSide
  // ISO timestamp when the review becomes visible (after both sides have
  // submitted, or when the 14-day window expires). Null while the review
  // is still hidden pending its counterpart.
  published_at: string | null
  created_at: string
}

export type AverageRating = {
  average: number
  count: number
}
