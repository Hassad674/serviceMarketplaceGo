"use client"

import { useEffect, useState, useCallback, Suspense } from "react"
import { useSearchParams } from "next/navigation"
import { useTranslations } from "next-intl"
import { ShieldAlert, ArrowLeft, ArrowRight } from "lucide-react"
import { Link, useRouter } from "@i18n/navigation"

const ROLE_DASHBOARDS: Record<string, string> = {
  agency: "/dashboard/agency",
  provider: "/dashboard/provider",
  enterprise: "/dashboard/enterprise",
  referrer: "/dashboard/referrer",
}

const COUNTDOWN_SECONDS = 10

function AccessDeniedContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const t = useTranslations("accessDenied")

  const role = searchParams.get("role") || "provider"
  const attempted = searchParams.get("attempted") || "unknown"

  const dashboardPath = ROLE_DASHBOARDS[role] || "/dashboard/provider"
  const roleLabel = t(`roles.${role}` as Parameters<typeof t>[0])
  const attemptedLabel = t(`roles.${attempted}` as Parameters<typeof t>[0])

  const [secondsLeft, setSecondsLeft] = useState(COUNTDOWN_SECONDS)

  const handleRedirect = useCallback(() => {
    router.push(dashboardPath)
  }, [router, dashboardPath])

  useEffect(() => {
    if (secondsLeft <= 0) {
      handleRedirect()
      return
    }

    const timer = setInterval(() => {
      setSecondsLeft((prev) => prev - 1)
    }, 1000)

    return () => clearInterval(timer)
  }, [secondsLeft, handleRedirect])

  return (
    <div className="flex min-h-screen flex-col items-center justify-center bg-gradient-to-b from-gray-50 to-white px-4 dark:from-gray-950 dark:to-gray-900">
      <div className="w-full max-w-md">
        <div className="rounded-2xl border border-gray-200 bg-white p-8 shadow-lg dark:border-gray-800 dark:bg-gray-900">
          {/* Shield icon */}
          <div className="flex justify-center">
            <div className="flex h-16 w-16 items-center justify-center rounded-full bg-red-50 dark:bg-red-500/10">
              <ShieldAlert className="h-8 w-8 text-red-500" aria-hidden="true" />
            </div>
          </div>

          {/* Title */}
          <h1 className="mt-6 text-center text-xl font-bold text-red-500">
            {t("title")}
          </h1>

          {/* Description */}
          <p className="mt-3 text-center text-sm leading-relaxed text-gray-600 dark:text-gray-400">
            {t("description", {
              attempted: attemptedLabel,
              role: roleLabel,
            })}
          </p>

          {/* Warning box */}
          <div className="mt-6 rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-500/20 dark:bg-amber-500/10">
            <p className="text-center text-sm text-amber-700 dark:text-amber-400">
              {t("warning")}
            </p>
          </div>

          {/* Countdown */}
          <div className="mt-4 rounded-lg bg-gray-50 p-4 dark:bg-gray-800">
            <p className="text-center text-sm font-medium text-primary">
              {t("countdown", { seconds: secondsLeft })}
            </p>
          </div>

          {/* Actions */}
          <div className="mt-6 space-y-3">
            <Link
              href={dashboardPath}
              className="gradient-primary flex w-full items-center justify-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium text-white shadow-sm transition-all duration-150 hover:shadow-md"
            >
              {t("goToDashboard", { role: roleLabel })}
              <ArrowRight className="h-4 w-4" aria-hidden="true" />
            </Link>

            <button
              type="button"
              onClick={() => router.back()}
              className="flex w-full items-center justify-center gap-2 rounded-lg border border-gray-200 bg-white px-4 py-2.5 text-sm font-medium text-gray-700 shadow-sm transition-all duration-150 hover:bg-gray-50 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"
            >
              <ArrowLeft className="h-4 w-4" aria-hidden="true" />
              {t("goBack")}
            </button>
          </div>

          {/* Support text */}
          <p className="mt-6 text-center text-xs text-gray-400 dark:text-gray-500">
            {t("contactSupport")}
          </p>
        </div>
      </div>
    </div>
  )
}

export default function AccessDeniedPage() {
  return (
    <Suspense>
      <AccessDeniedContent />
    </Suspense>
  )
}
