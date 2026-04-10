/**
 * Shared Review type used across multiple features.
 *
 * The single source of truth for the review data shape. Features like
 * `review` and `provider` (for project history) both import from here
 * to avoid cross-feature imports.
 */
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
  created_at: string
}

export type AverageRating = {
  average: number
  count: number
}
