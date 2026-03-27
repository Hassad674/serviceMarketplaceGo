"use client"

import { useState, useCallback } from "react"
import { X, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useCreateReview } from "../hooks/use-reviews"
import { StarRating } from "./star-rating"

interface ReviewModalProps {
  proposalId: string
  proposalTitle: string
  isOpen: boolean
  onClose: () => void
}

export function ReviewModal({
  proposalId,
  proposalTitle,
  isOpen,
  onClose,
}: ReviewModalProps) {
  const t = useTranslations("review")
  const [globalRating, setGlobalRating] = useState(0)
  const [timeliness, setTimeliness] = useState(0)
  const [communication, setCommunication] = useState(0)
  const [quality, setQuality] = useState(0)
  const [comment, setComment] = useState("")

  const { mutate: submitReview, isPending } = useCreateReview()

  const resetForm = useCallback(() => {
    setGlobalRating(0)
    setTimeliness(0)
    setCommunication(0)
    setQuality(0)
    setComment("")
  }, [])

  const handleSubmit = useCallback(() => {
    if (globalRating === 0) return

    submitReview(
      {
        proposal_id: proposalId,
        global_rating: globalRating,
        timeliness: timeliness > 0 ? timeliness : undefined,
        communication: communication > 0 ? communication : undefined,
        quality: quality > 0 ? quality : undefined,
        comment: comment.trim() || undefined,
      },
      {
        onSuccess: () => {
          resetForm()
          onClose()
        },
      },
    )
  }, [
    proposalId, globalRating, timeliness, communication,
    quality, comment, submitReview, resetForm, onClose,
  ])

  if (!isOpen) return null

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm"
      onClick={(e) => e.target === e.currentTarget && onClose()}
      role="dialog"
      aria-modal="true"
      aria-label={t("title")}
    >
      <div className="mx-4 w-full max-w-lg animate-scale-in rounded-2xl bg-white p-6 shadow-xl dark:bg-gray-900">
        {/* Header */}
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-foreground">
              {t("title")}
            </h2>
            <p className="mt-0.5 text-sm text-muted-foreground">
              {proposalTitle}
            </p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="rounded-lg p-1.5 text-muted-foreground hover:bg-muted transition-colors"
            aria-label={t("close")}
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Global rating */}
        <div className="space-y-5">
          <StarRating
            rating={globalRating}
            onRatingChange={setGlobalRating}
            size="lg"
            label={`${t("globalRating")} *`}
          />

          {/* Optional criteria */}
          <div className="space-y-3 border-t border-border pt-4">
            <p className="text-sm font-medium text-muted-foreground">
              {t("detailedCriteria")}
            </p>

            <StarRating
              rating={timeliness}
              onRatingChange={setTimeliness}
              size="md"
              label={t("timeliness")}
            />

            <StarRating
              rating={communication}
              onRatingChange={setCommunication}
              size="md"
              label={t("communication")}
            />

            <StarRating
              rating={quality}
              onRatingChange={setQuality}
              size="md"
              label={t("quality")}
            />
          </div>

          {/* Comment */}
          <div className="space-y-1.5">
            <label
              htmlFor="review-comment"
              className="text-sm font-medium text-foreground"
            >
              {t("comment")}
            </label>
            <textarea
              id="review-comment"
              value={comment}
              onChange={(e) => setComment(e.target.value)}
              placeholder={t("commentPlaceholder")}
              rows={4}
              maxLength={2000}
              className={cn(
                "w-full resize-none rounded-lg border border-border bg-transparent",
                "px-3 py-2 text-sm text-foreground placeholder:text-muted-foreground",
                "focus:border-rose-500 focus:outline-none focus:ring-4 focus:ring-rose-500/10",
                "transition-all duration-200",
              )}
              disabled={isPending}
            />
            <p className="text-right text-xs text-muted-foreground">
              {comment.length}/2000
            </p>
          </div>

          {/* Actions */}
          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={handleSubmit}
              disabled={isPending || globalRating === 0}
              className={cn(
                "flex-1 rounded-lg px-4 py-2.5 text-sm font-semibold text-white",
                "gradient-primary hover:shadow-glow active:scale-[0.98]",
                "transition-all duration-200",
                "disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-none",
              )}
            >
              {isPending && <Loader2 className="mr-2 inline h-4 w-4 animate-spin" />}
              {t("submit")}
            </button>
            <button
              type="button"
              onClick={onClose}
              disabled={isPending}
              className={cn(
                "rounded-lg border border-border px-4 py-2.5 text-sm font-medium",
                "text-foreground hover:bg-muted transition-all duration-200",
              )}
            >
              {t("cancel")}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}
