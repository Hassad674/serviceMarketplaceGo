import { getTranslations } from "next-intl/server"
import { SkeletonBlock } from "@/shared/components/ui/skeleton-block"

// loading.tsx for /messages (PERF-W-03). Mirrors the messaging
// layout's two-pane shell: conversation list on the left, message
// area on the right. Reserves the dimensions so when the WS-driven
// data lands, the panes don't shift.
export default async function MessagesLoading() {
  const t = await getTranslations("boundary")
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={t("loadingMessages")}
      className="flex h-[calc(100vh-12rem)] gap-4"
    >
      <span className="sr-only">{t("loadingMessages")}</span>
      <aside className="hidden w-72 shrink-0 space-y-3 sm:block">
        <SkeletonBlock className="h-10 w-full" />
        <SkeletonBlock className="h-16 w-full" />
        <SkeletonBlock className="h-16 w-full" />
        <SkeletonBlock className="h-16 w-full" />
        <SkeletonBlock className="h-16 w-full" />
        <SkeletonBlock className="h-16 w-full" />
      </aside>
      <main className="flex flex-1 flex-col gap-3">
        <SkeletonBlock className="h-12" />
        <SkeletonBlock className="flex-1" />
        <SkeletonBlock className="h-12" />
      </main>
    </div>
  )
}
