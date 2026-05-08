import { describe, it, expect } from "vitest"

import { buildAggregateRating, buildReviewItems } from "../rating"
import type { Review } from "@/shared/types/review"

function makeReview(overrides: Partial<Review> = {}): Review {
  return {
    id: "rev-1",
    proposal_id: "p-1",
    reviewer_id: "u-1",
    reviewed_id: "u-2",
    global_rating: 5,
    timeliness: 5,
    communication: 5,
    quality: 5,
    comment: "Great work",
    video_url: null,
    title_visible: false,
    side: "client_to_provider",
    published_at: "2026-04-01T00:00:00Z",
    created_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

describe("buildAggregateRating", () => {
  it("returns undefined when there are no reviews (Google rejects empty)", () => {
    expect(buildAggregateRating({ rating: null })).toBeUndefined()
    expect(
      buildAggregateRating({ rating: { average: 0, count: 0 } }),
    ).toBeUndefined()
  })

  it("returns a complete AggregateRating when count > 0", () => {
    const out = buildAggregateRating({ rating: { average: 4.755, count: 12 } })
    expect(out).toMatchObject({
      "@type": "AggregateRating",
      reviewCount: 12,
      bestRating: 5,
      worstRating: 1,
    })
    // Rating is rounded to 1 decimal place — schema.org best practice.
    expect(out!.ratingValue).toBe(4.8)
  })

  it("rounds with one decimal precision", () => {
    expect(
      buildAggregateRating({ rating: { average: 4.49, count: 1 } })!
        .ratingValue,
    ).toBe(4.5)
    expect(
      buildAggregateRating({ rating: { average: 4.0, count: 1 } })!.ratingValue,
    ).toBe(4)
  })
})

describe("buildReviewItems", () => {
  it("returns undefined when no reviews are available", () => {
    expect(
      buildReviewItems({ reviews: null, anonymousAuthorLabel: "x" }),
    ).toBeUndefined()
    expect(
      buildReviewItems({ reviews: [], anonymousAuthorLabel: "x" }),
    ).toBeUndefined()
  })

  it("emits up to 5 review items by default", () => {
    const reviews = Array.from({ length: 8 }, (_, i) =>
      makeReview({ id: `r-${i}` }),
    )
    const items = buildReviewItems({
      reviews,
      anonymousAuthorLabel: "Verified client",
    })
    expect(items).toHaveLength(5)
  })

  it("skips reviews with global_rating == 0 (no rating signal)", () => {
    const items = buildReviewItems({
      reviews: [makeReview({ global_rating: 0 }), makeReview({ id: "r-ok" })],
      anonymousAuthorLabel: "Verified client",
    })
    expect(items).toHaveLength(1)
    const rating = items![0].reviewRating as Record<string, unknown>
    expect(rating.ratingValue).toBe(5)
  })

  it("uses published_at when available, falls back to created_at", () => {
    const items = buildReviewItems({
      reviews: [
        makeReview({ published_at: "2026-04-10T00:00:00Z" }),
        makeReview({
          id: "r-2",
          published_at: null,
          created_at: "2026-04-12T00:00:00Z",
        }),
      ],
      anonymousAuthorLabel: "Verified client",
    })
    expect(items).toHaveLength(2)
    expect(items![0].datePublished).toBe("2026-04-10T00:00:00Z")
    expect(items![1].datePublished).toBe("2026-04-12T00:00:00Z")
  })

  it("anonymises the author with the provided label", () => {
    const items = buildReviewItems({
      reviews: [makeReview()],
      anonymousAuthorLabel: "Verified client",
    })
    const author = items![0].author as Record<string, unknown>
    expect(author.name).toBe("Verified client")
  })
})
