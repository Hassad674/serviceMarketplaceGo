/**
 * rating.ts — `aggregateRating` + `review` JSON-LD payload helpers.
 *
 * Google rejects schema.org rating payloads that have `reviewCount: 0`
 * or that lack a `ratingValue` — the rich result is silently dropped
 * from the SERP and Search Console flags the page as "invalid".
 *
 * The helpers below return `undefined` when the entity has zero
 * reviews so the caller can omit the field entirely from the JSON-LD
 * payload (rather than emitting a hollow `aggregateRating: { count: 0 }`
 * block that triggers the rejection).
 *
 * Reference:
 *   - https://schema.org/AggregateRating
 *   - https://developers.google.com/search/docs/appearance/structured-data/review-snippet
 */

import type { Review, AverageRating } from "@/shared/types/review"

export interface AggregateRatingInput {
  rating: AverageRating | null
}

/**
 * buildAggregateRating returns a schema.org AggregateRating payload
 * when the entity has at least one review, or `undefined` otherwise.
 *
 * `bestRating` and `worstRating` are explicitly set to 5/1 because
 * Google's parser warns when the scale is implicit (it defaults to
 * 5/1 anyway, but the explicit form removes ambiguity).
 */
export function buildAggregateRating(
  input: AggregateRatingInput,
): Record<string, unknown> | undefined {
  if (!input.rating || input.rating.count <= 0) return undefined
  return {
    "@type": "AggregateRating",
    ratingValue: roundRating(input.rating.average),
    reviewCount: input.rating.count,
    bestRating: 5,
    worstRating: 1,
  }
}

export interface ReviewItemsInput {
  reviews: Review[] | null
  /** Max number of reviews to embed. Default 5 — Google ignores more. */
  limit?: number
  /** Localized label for anonymized review authors. */
  anonymousAuthorLabel: string
}

/**
 * buildReviewItems returns an array of schema.org Review payloads
 * suitable for embedding inside the parent entity's `review` field.
 *
 * Returns `undefined` when no reviews are available so the caller can
 * omit the field entirely. Google rejects an empty `review: []` array.
 *
 * Each review carries:
 *   - `author` — anonymized; we never expose reviewer identities
 *     publicly because reviews are double-blind on this marketplace.
 *   - `datePublished` — ISO date from `published_at` if available,
 *     fallback to `created_at` for older rows where `published_at`
 *     was not yet populated.
 *   - `reviewBody` — the comment text.
 *   - `reviewRating` — global rating mapped to schema.org Rating.
 */
export function buildReviewItems(
  input: ReviewItemsInput,
): Record<string, unknown>[] | undefined {
  if (!input.reviews || input.reviews.length === 0) return undefined
  const limit = input.limit ?? 5
  const items = input.reviews
    .slice(0, limit)
    .filter((review) => review.global_rating > 0)
    .map((review) => ({
      "@type": "Review",
      datePublished: review.published_at ?? review.created_at,
      reviewBody: review.comment || undefined,
      reviewRating: {
        "@type": "Rating",
        ratingValue: review.global_rating,
        bestRating: 5,
        worstRating: 1,
      },
      author: {
        "@type": "Organization",
        name: input.anonymousAuthorLabel,
      },
    }))
  return items.length > 0 ? items : undefined
}

function roundRating(value: number): number {
  // Google accepts up to 1 decimal place; more precision triggers a
  // "value too precise" warning in Search Console.
  return Math.round(value * 10) / 10
}
