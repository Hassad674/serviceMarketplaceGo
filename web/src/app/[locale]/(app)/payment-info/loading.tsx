import { getTranslations } from "next-intl/server"
import { SkeletonBlock } from "@/shared/components/ui/skeleton-block"

// loading.tsx for /payment-info (PERF-W-03). The payment-info page
// pulls a Stripe Connect account snapshot + KYC status before it can
// render — the skeleton mirrors the resulting card stack so the page
// stays usable during slow Stripe API responses.
export default async function PaymentInfoLoading() {
  const t = await getTranslations("boundary")
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={t("loadingPaymentInfo")}
      className="space-y-6"
    >
      <span className="sr-only">{t("loadingPaymentInfo")}</span>
      <SkeletonBlock className="h-9 w-1/3" />
      <SkeletonBlock className="h-5 w-2/3" />
      <SkeletonBlock className="h-40 w-full rounded-2xl" />
      <SkeletonBlock className="h-24 w-full rounded-2xl" />
      <SkeletonBlock className="h-24 w-full rounded-2xl" />
    </div>
  )
}
