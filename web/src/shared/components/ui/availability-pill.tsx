"use client"

import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

// AvailabilityStatus mirrors the backend enum at the boundary so both
// persona features (freelance + referrer) can share the pill without
// re-declaring the union in shared. Keeping it here means the atom
// stays self-contained and features import both the type and the
// component from one place.
export type AvailabilityStatus =
  | "available_now"
  | "available_soon"
  | "not_available"

const STATUS_STYLES: Record<AvailabilityStatus, string> = {
  available_now:
    "bg-emerald-50 text-emerald-700 border-emerald-200 dark:bg-emerald-500/10 dark:text-emerald-300 dark:border-emerald-500/30",
  available_soon:
    "bg-amber-50 text-amber-700 border-amber-200 dark:bg-amber-500/10 dark:text-amber-300 dark:border-amber-500/30",
  not_available:
    "bg-rose-50 text-rose-700 border-rose-200 dark:bg-rose-500/10 dark:text-rose-300 dark:border-rose-500/30",
}

const DOT_STYLES: Record<AvailabilityStatus, string> = {
  available_now: "bg-emerald-500",
  available_soon: "bg-amber-500",
  not_available: "bg-rose-500",
}

interface AvailabilityPillProps {
  status: AvailabilityStatus
  className?: string
}

// AvailabilityPill is the small colored dot + label badge used on
// profile headers, hero strips, and listing cards. Pure presentational
// — reads from the shared profile.availability i18n namespace so the
// legacy labels stay authoritative.
export function AvailabilityPill({ status, className }: AvailabilityPillProps) {
  const t = useTranslations("profile.availability")
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium",
        STATUS_STYLES[status],
        className,
      )}
      data-testid={`availability-pill-${status}`}
    >
      <span
        aria-hidden="true"
        className={cn("h-1.5 w-1.5 rounded-full", DOT_STYLES[status])}
      />
      {t(statusLabelKey(status))}
    </span>
  )
}

function statusLabelKey(status: AvailabilityStatus): string {
  switch (status) {
    case "available_now":
      return "statusAvailableNow"
    case "available_soon":
      return "statusAvailableSoon"
    case "not_available":
      return "statusNotAvailable"
  }
}
