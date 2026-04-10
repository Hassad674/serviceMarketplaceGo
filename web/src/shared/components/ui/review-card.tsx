"use client"

import { useTranslations } from "next-intl"
import { Star } from "lucide-react"
import { cn } from "@/shared/lib/utils"
import type { Review } from "@/shared/types/review"

interface ReviewCardProps {
  review: Review
  className?: string
}

/**
 * Standalone card that renders one review with stars, optional sub-criteria,
 * comment and video. Used by both the standalone reviews list and the
 * embedded project history cards.
 */
export function ReviewCard({ review, className }: ReviewCardProps) {
  const t = useTranslations("review")

  return (
    <div
      className={cn(
        "rounded-xl border border-border/50 p-4 transition-colors duration-200 hover:border-border",
        className,
      )}
    >
      <div className="flex items-center justify-between">
        <InlineStars rating={review.global_rating} />
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
            <span>
              {t("timeliness")}: {review.timeliness}/5
            </span>
          )}
          {review.communication && (
            <span>
              {t("communication")}: {review.communication}/5
            </span>
          )}
          {review.quality && (
            <span>
              {t("quality")}: {review.quality}/5
            </span>
          )}
        </div>
      )}

      {review.comment && (
        <p className="mt-3 whitespace-pre-wrap break-words text-sm leading-relaxed text-foreground">
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
            {t("videoReview")}
          </video>
        </div>
      )}
    </div>
  )
}

function InlineStars({ rating }: { rating: number }) {
  return (
    <div className="flex items-center gap-0.5" aria-label={`${rating}/5`}>
      {[1, 2, 3, 4, 5].map((i) => (
        <Star
          key={i}
          className={cn(
            "h-4 w-4",
            i <= rating ? "fill-amber-400 text-amber-400" : "text-muted-foreground/30",
          )}
          strokeWidth={1.5}
        />
      ))}
    </div>
  )
}
