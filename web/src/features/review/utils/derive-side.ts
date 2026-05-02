// `deriveReviewSide` is shared with the `messaging` feature (P9). Lives
// in `@/shared/lib/review/derive-side` and is re-exported here so
// existing intra-feature imports keep working without churn.
export {
  deriveReviewSide,
  type ReviewSideSource,
  type ReviewSideMessageSource,
} from "@/shared/lib/review/derive-side"
