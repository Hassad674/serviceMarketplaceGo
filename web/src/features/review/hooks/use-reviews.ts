// Review hooks live in `@/shared/hooks/review/use-reviews` (P9 — the
// review modal is consumed cross-feature). Re-exported here for
// back-compat.
export {
  useReviewsByUser,
  useAverageRating,
  useCanReview,
  useCreateReview,
  useUploadReviewVideo,
} from "@/shared/hooks/review/use-reviews"
