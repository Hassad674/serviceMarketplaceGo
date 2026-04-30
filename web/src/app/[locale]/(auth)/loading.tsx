import { getTranslations } from "next-intl/server"
import { SkeletonBlock } from "@/shared/components/ui/skeleton-block"

// loading.tsx for the (auth) route group (PERF-W-03). The auth flow
// is server-rendered already; this loader only ever shows during
// initial JS hydration of the form + during navigation between
// /login, /register and password-reset variants. Renders a centered
// card-shaped skeleton matching the AuthLayout's wrapper.
export default async function AuthLoading() {
  const t = await getTranslations("boundary")
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={t("loadingAuth")}
      className="w-full max-w-md space-y-4"
    >
      <span className="sr-only">{t("loadingAuth")}</span>
      <SkeletonBlock className="h-9 w-1/2" />
      <SkeletonBlock className="h-4 w-3/4" />
      <SkeletonBlock className="h-12 w-full" />
      <SkeletonBlock className="h-12 w-full" />
      <SkeletonBlock className="h-10 w-full rounded-lg" />
    </div>
  )
}
