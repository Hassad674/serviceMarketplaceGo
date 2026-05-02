// Review API lives in `@/shared/lib/review/review-api` (P9 — review
// hooks are shared). Re-exported here for back-compat.
export {
  fetchReviewsByUser,
  fetchAverageRating,
  fetchCanReview,
  createReview,
  uploadReviewVideo,
} from "@/shared/lib/review/review-api"
export type {
  Review,
  AverageRating,
  ReviewListResponse,
  CanReviewResponse,
  CreateReviewPayload,
} from "@/shared/lib/review/review-api"
