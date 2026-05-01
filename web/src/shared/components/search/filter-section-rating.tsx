"use client"

import { Star } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { SectionShell } from "./filter-primitives"

import { Button } from "@/shared/components/ui/button"
// Star-based "minimum rating" filter. Click a filled star to set the
// floor; click the SAME star to clear back to zero. The brief
// reserves space here for future verified / top_rated / negotiable
// toggles — they are not in the V1 scope, so this section currently
// renders just the rating row.

interface FilterSectionRatingProps {
  value: number
  onChange: (next: number) => void
}

export function FilterSectionRating({
  value,
  onChange,
}: FilterSectionRatingProps) {
  const t = useTranslations("search.filters")
  return (
    <SectionShell title={t("rating")}>
      <div className="flex items-center gap-1" role="radiogroup" aria-label={t("rating")}>
        {[1, 2, 3, 4, 5].map((star) => {
          const selected = star <= value
          return (
            <Button variant="ghost" size="auto"
              key={star}
              type="button"
              role="radio"
              aria-checked={selected}
              aria-label={`${star}`}
              onClick={() => onChange(value === star ? 0 : star)}
              className={cn(
                "rounded-sm p-0.5 transition-colors",
                selected ? "text-amber-400" : "text-muted-foreground/40",
                "hover:text-amber-400 focus-visible:outline-none focus-visible:ring-4 focus-visible:ring-rose-500/20",
              )}
            >
              <Star
                className={cn("h-5 w-5", selected && "fill-amber-400")}
                strokeWidth={1.75}
              />
            </Button>
          )
        })}
      </div>
    </SectionShell>
  )
}
