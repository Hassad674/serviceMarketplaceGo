import { getTranslations } from "next-intl/server"
import { SkeletonBlock } from "@/shared/components/ui/skeleton-block"

// loading.tsx for the public listing route group (PERF-W-03).
// Renders a top header strip and a 3x4 card grid that mirrors the
// SearchPageLayout's results grid, keeping the layout reserved so
// the eventual cards drop in with no shift.
export default async function PublicListingLoading() {
  const t = await getTranslations("boundary")
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={t("loadingListing")}
      className="space-y-6"
    >
      <span className="sr-only">{t("loadingListing")}</span>
      <div className="flex flex-col gap-3">
        <SkeletonBlock className="h-9 w-1/2" />
        <SkeletonBlock className="h-5 w-3/4" />
      </div>
      <div className="flex gap-3">
        <SkeletonBlock className="h-10 flex-1" />
        <SkeletonBlock className="h-10 w-28" />
        <SkeletonBlock className="h-10 w-32" />
      </div>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {Array.from({ length: 9 }).map((_, i) => (
          <SkeletonBlock key={i} className="h-72 rounded-2xl" />
        ))}
      </div>
    </div>
  )
}
