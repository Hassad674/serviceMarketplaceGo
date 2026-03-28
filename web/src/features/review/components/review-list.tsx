"use client"

import { Star } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useReviewsByUser, useAverageRating } from "../hooks/use-reviews"
import { StarRating } from "./star-rating"
import type { Review } from "../api/review-api"

interface ReviewListProps {
  userId: string
}

export function ReviewList({ userId }: ReviewListProps) {
  const t = useTranslations("review")

  const { data: reviewsData, isLoading: reviewsLoading } = useReviewsByUser(userId)
  const { data: avgData, isLoading: avgLoading } = useAverageRating(userId)

  if (reviewsLoading || avgLoading) return <ReviewListSkeleton />

  const reviews = reviewsData?.data ?? []
  const average = avgData?.data?.average ?? 0
  const count = avgData?.data?.count ?? 0

  if (count === 0) return null

  return (
    <div className="rounded-2xl border border-border bg-white p-6 shadow-sm dark:bg-gray-900">
      <ReviewListHeader
        title={t("reviewsTitle")}
        average={average}
        count={count}
        singular={t("reviewSingular")}
        plural={t("reviewPlural")}
      />

      <div className="space-y-4">
        {reviews.map((review) => (
          <ReviewCard key={review.id} review={review} labels={{
            timeliness: t("timeliness"),
            communication: t("communication"),
            quality: t("quality"),
            videoReview: t("videoReview"),
          }} />
        ))}
      </div>
    </div>
  )
}

function ReviewListHeader({
  title,
  average,
  count,
  singular,
  plural,
}: {
  title: string
  average: number
  count: number
  singular: string
  plural: string
}) {
  return (
    <div className="mb-6 flex items-center justify-between">
      <h3 className="text-lg font-semibold text-foreground">{title}</h3>
      <div className="flex items-center gap-2">
        <Star className="h-5 w-5 fill-amber-400 text-amber-400" strokeWidth={1.5} />
        <span className="text-xl font-bold text-foreground">
          {average.toFixed(1)}
        </span>
        <span className="text-sm text-muted-foreground">
          ({count} {count === 1 ? singular : plural})
        </span>
      </div>
    </div>
  )
}

function ReviewCard({
  review,
  labels,
}: {
  review: Review
  labels: { timeliness: string; communication: string; quality: string; videoReview: string }
}) {
  return (
    <div
      className={cn(
        "rounded-xl border border-border/50 p-4",
        "hover:border-border transition-colors duration-200",
      )}
    >
      <div className="flex items-center justify-between">
        <StarRating rating={review.global_rating} size="sm" readOnly />
        <time
          className="text-xs text-muted-foreground"
          dateTime={review.created_at}
        >
          {new Date(review.created_at).toLocaleDateString()}
        </time>
      </div>

      {(review.timeliness || review.communication || review.quality) && (
        <div className="mt-2 flex flex-wrap gap-3 text-xs text-muted-foreground">
          {review.timeliness && (
            <span>{labels.timeliness}: {review.timeliness}/5</span>
          )}
          {review.communication && (
            <span>{labels.communication}: {review.communication}/5</span>
          )}
          {review.quality && (
            <span>{labels.quality}: {review.quality}/5</span>
          )}
        </div>
      )}

      {review.comment && (
        <p className="mt-3 text-sm leading-relaxed text-foreground">
          {review.comment}
        </p>
      )}

      {review.video_url && (
        <div className="mt-3">
          <video
            src={review.video_url}
            controls
            className="w-full rounded-lg border border-border"
            style={{ maxHeight: 300 }}
            preload="metadata"
          >
            {labels.videoReview}
          </video>
        </div>
      )}
    </div>
  )
}

function ReviewListSkeleton() {
  return (
    <div className="rounded-2xl border border-border bg-white p-6 shadow-sm dark:bg-gray-900">
      <div className="mb-6 flex items-center justify-between">
        <div className="h-6 w-32 animate-shimmer rounded bg-muted" />
        <div className="h-6 w-24 animate-shimmer rounded bg-muted" />
      </div>
      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="rounded-xl border border-border/50 p-4">
            <div className="h-5 w-28 animate-shimmer rounded bg-muted" />
            <div className="mt-3 h-12 w-full animate-shimmer rounded bg-muted" />
          </div>
        ))}
      </div>
    </div>
  )
}
