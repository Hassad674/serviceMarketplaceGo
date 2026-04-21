"use client"

import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { ReviewCard } from "@/shared/components/ui/review-card"
import type { Review } from "@/shared/types/review"

interface ClientReviewsListProps {
  reviews: Review[]
}

// ClientReviewsList renders reviews left by providers about this
// client (side === "provider_to_client"). We do not filter here: the
// backend guarantees the /clients endpoint only returns provider-side
// reviews, so any filtering would double-enforce and hide legit data.
export function ClientReviewsList(props: ClientReviewsListProps) {
  const { reviews } = props
  const t = useTranslations("clientProfile")

  return (
    <section
      className="bg-card border border-border rounded-2xl p-6 shadow-sm"
      aria-labelledby="client-reviews-heading"
    >
      <div className="flex items-center gap-3">
        <h2
          id="client-reviews-heading"
          className="text-lg font-semibold text-foreground"
        >
          {t("reviewsTitle")}
        </h2>
        {reviews.length > 0 ? (
          <span className="rounded-full bg-muted text-muted-foreground px-2.5 py-0.5 text-xs font-medium">
            {reviews.length}
          </span>
        ) : null}
      </div>

      {reviews.length === 0 ? (
        <div className="mt-4 flex flex-col items-center gap-2 rounded-xl border border-dashed border-border bg-muted/30 px-6 py-8 text-center">
          <MessageSquare
            className="h-8 w-8 text-muted-foreground"
            aria-hidden="true"
          />
          <p className="text-sm text-muted-foreground">{t("reviewsEmpty")}</p>
        </div>
      ) : (
        <ul className="mt-4 space-y-3">
          {reviews.map((review) => (
            <li key={review.id}>
              <ReviewCard review={review} />
            </li>
          ))}
        </ul>
      )}
    </section>
  )
}
