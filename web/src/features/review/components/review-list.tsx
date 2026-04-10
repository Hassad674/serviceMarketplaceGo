"use client"

import { Star } from "lucide-react"
import { useTranslations } from "next-intl"
import { useReviewsByUser, useAverageRating } from "../hooks/use-reviews"
import { ReviewCard } from "@/shared/components/ui/review-card"

interface ReviewListProps {
  userId: string
}

/**
 * Legacy standalone reviews section. Kept for potential admin/moderation
 * use; no longer mounted on the public profile pages — project history is
 * now the unified entry point there.
 */
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
          <ReviewCard key={review.id} review={review} />
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
