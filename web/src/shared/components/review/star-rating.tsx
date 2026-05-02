"use client"

import { useState } from "react"
import { Star } from "lucide-react"
import { cn } from "@/shared/lib/utils"

import { Button } from "@/shared/components/ui/button"
interface StarRatingProps {
  rating: number
  onRatingChange?: (rating: number) => void
  size?: "sm" | "md" | "lg"
  label?: string
  readOnly?: boolean
}

const SIZES = {
  sm: "h-4 w-4",
  md: "h-5 w-5",
  lg: "h-6 w-6",
}

export function StarRating({
  rating,
  onRatingChange,
  size = "md",
  label,
  readOnly = false,
}: StarRatingProps) {
  const [hovered, setHovered] = useState(0)

  const displayRating = hovered || rating

  return (
    <div className="space-y-1">
      {label && (
        <span className="text-sm font-medium text-foreground">{label}</span>
      )}
      <div
        className="flex items-center gap-0.5"
        onMouseLeave={() => !readOnly && setHovered(0)}
        role={readOnly ? undefined : "radiogroup"}
        aria-label={label}
      >
        {[1, 2, 3, 4, 5].map((star) => (
          <Button variant="ghost" size="auto"
            key={star}
            type="button"
            disabled={readOnly}
            onClick={() => onRatingChange?.(star)}
            onMouseEnter={() => !readOnly && setHovered(star)}
            className={cn(
              "transition-all duration-150",
              readOnly ? "cursor-default" : "cursor-pointer hover:scale-110",
            )}
            aria-label={`${star} star${star > 1 ? "s" : ""}`}
            aria-checked={rating === star}
            role={readOnly ? undefined : "radio"}
          >
            <Star
              className={cn(
                SIZES[size],
                displayRating >= star
                  ? "fill-amber-400 text-amber-400"
                  : "fill-none text-gray-300 dark:text-gray-600",
              )}
              strokeWidth={1.5}
            />
          </Button>
        ))}
      </div>
    </div>
  )
}
