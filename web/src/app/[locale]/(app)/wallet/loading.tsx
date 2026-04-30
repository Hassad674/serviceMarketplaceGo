import { getTranslations } from "next-intl/server"
import { SkeletonBlock } from "@/shared/components/ui/skeleton-block"

// loading.tsx for /wallet (PERF-W-03). Reserves layout for the
// balance card, transaction filters and the (paginated) transaction
// list. The wallet page issues several queries on mount (balance,
// payouts, recent transactions) so a skeleton is critical for
// perceived performance.
export default async function WalletLoading() {
  const t = await getTranslations("boundary")
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={t("loadingWallet")}
      className="space-y-6"
    >
      <span className="sr-only">{t("loadingWallet")}</span>
      <SkeletonBlock className="h-9 w-1/3" />
      <SkeletonBlock className="h-32 w-full rounded-2xl" />
      <div className="flex gap-3">
        <SkeletonBlock className="h-10 flex-1" />
        <SkeletonBlock className="h-10 w-28" />
      </div>
      <div className="space-y-2">
        {Array.from({ length: 8 }).map((_, i) => (
          <SkeletonBlock key={i} className="h-14 w-full" />
        ))}
      </div>
    </div>
  )
}
