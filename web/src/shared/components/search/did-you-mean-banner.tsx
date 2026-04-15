"use client"

import { Sparkles } from "lucide-react"
import { useTranslations } from "next-intl"

/**
 * DidYouMeanBanner is the small inline banner shown above the
 * results grid when Typesense returns a corrected query string.
 *
 * Click to apply the correction. The component is purely
 * presentational — the parent owns the search query state and
 * decides what to do with the click.
 */

export interface DidYouMeanBannerProps {
  /** The Typesense-corrected query string (must be non-empty). */
  correctedQuery: string
  /** Fires when the user clicks "run corrected search". */
  onApply: (correctedQuery: string) => void
}

export function DidYouMeanBanner({ correctedQuery, onApply }: DidYouMeanBannerProps) {
  const t = useTranslations("search")
  const trimmed = correctedQuery.trim()
  if (trimmed.length === 0) return null
  return (
    <div
      role="status"
      aria-live="polite"
      className="flex items-center gap-3 rounded-xl border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-foreground dark:border-rose-500/30 dark:bg-rose-500/10"
    >
      <Sparkles className="h-4 w-4 text-rose-600 dark:text-rose-400" aria-hidden />
      <span className="flex-1">
        {t("didYouMean")}{" "}
        <button
          type="button"
          onClick={() => onApply(trimmed)}
          aria-label={t("didYouMeanCta")}
          className="font-semibold text-rose-700 underline-offset-4 transition-colors hover:underline focus:outline-none focus-visible:underline dark:text-rose-300"
        >
          {trimmed}
        </button>
        {" ?"}
      </span>
    </div>
  )
}
