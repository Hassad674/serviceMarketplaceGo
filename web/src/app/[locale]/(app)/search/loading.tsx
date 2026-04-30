import { getTranslations } from "next-intl/server"
import { SkeletonBlock } from "@/shared/components/ui/skeleton-block"

// loading.tsx for /search (PERF-W-03). Mirrors the search page's
// sidebar+grid layout so the cards drop in without shifting the
// filter rail.
export default async function SearchLoading() {
  const t = await getTranslations("boundary")
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={t("loadingSearch")}
      className="flex flex-col gap-4 lg:flex-row"
    >
      <span className="sr-only">{t("loadingSearch")}</span>
      <aside className="hidden w-64 shrink-0 space-y-3 lg:block">
        <SkeletonBlock className="h-10 w-full" />
        <SkeletonBlock className="h-32 w-full" />
        <SkeletonBlock className="h-32 w-full" />
      </aside>
      <main className="flex-1 space-y-4">
        <div className="flex gap-3">
          <SkeletonBlock className="h-10 flex-1" />
          <SkeletonBlock className="h-10 w-28" />
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <SkeletonBlock key={i} className="h-72 rounded-2xl" />
          ))}
        </div>
      </main>
    </div>
  )
}
