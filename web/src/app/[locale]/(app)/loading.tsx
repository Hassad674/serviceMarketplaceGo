import { getTranslations } from "next-intl/server"
import { SkeletonBlock } from "@/shared/components/ui/skeleton-block"

// loading.tsx for the authenticated dashboard route group
// (PERF-W-03). Next.js wraps the route in a Suspense boundary that
// renders this file while the page's RSC data is in flight, so the
// dashboard shell never shows a blank canvas during navigation.
//
// Markup intentionally mirrors the dashboard's layout — a centered
// content column with a header strip, a few KPI cards, and a list —
// so the perceived layout shift is near zero when real data lands.
export default async function DashboardLoading() {
  const t = await getTranslations("boundary")
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={t("loadingDashboard")}
      className="space-y-6"
    >
      <span className="sr-only">{t("loadingDashboard")}</span>
      <SkeletonBlock className="h-9 w-1/3" />
      <SkeletonBlock className="h-5 w-2/3" />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <SkeletonBlock className="h-32" />
        <SkeletonBlock className="h-32" />
        <SkeletonBlock className="h-32" />
      </div>
      <div className="space-y-3">
        <SkeletonBlock className="h-14 w-full" />
        <SkeletonBlock className="h-14 w-full" />
        <SkeletonBlock className="h-14 w-full" />
      </div>
    </div>
  )
}
