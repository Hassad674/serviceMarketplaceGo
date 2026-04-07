"use client"

import { AlertTriangle, ShieldAlert, ArrowRight } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import type { CurrentUser } from "@/shared/hooks/use-user"

interface KYCBannerProps {
  user: CurrentUser
}

export function KYCBanner({ user }: KYCBannerProps) {
  const t = useTranslations("kycBanner")

  if (user.role === "enterprise") return null
  if (user.kyc_status === "none" || user.kyc_status === "completed") return null

  const isRestricted = user.kyc_status === "restricted"
  const daysLeft = user.kyc_deadline ? daysUntil(user.kyc_deadline) : null

  return (
    <div
      role="alert"
      className={
        isRestricted
          ? "mb-5 flex items-start gap-3 rounded-xl border border-red-200 bg-red-50 p-4 dark:border-red-500/20 dark:bg-red-500/10 animate-slide-up"
          : "mb-5 flex items-start gap-3 rounded-xl border border-amber-200 bg-amber-50 p-4 dark:border-amber-500/20 dark:bg-amber-500/10 animate-slide-up"
      }
    >
      {isRestricted ? (
        <ShieldAlert className="mt-0.5 h-5 w-5 shrink-0 text-red-500" aria-hidden />
      ) : (
        <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-500" aria-hidden />
      )}
      <div className="flex-1 min-w-0">
        <p
          className={
            isRestricted
              ? "text-sm font-semibold text-red-900 dark:text-red-300"
              : "text-sm font-semibold text-amber-900 dark:text-amber-300"
          }
        >
          {isRestricted ? t("restrictedTitle") : t("pendingTitle")}
        </p>
        <p
          className={
            isRestricted
              ? "mt-1 text-sm text-red-700 dark:text-red-400"
              : "mt-1 text-sm text-amber-700 dark:text-amber-400"
          }
        >
          {isRestricted
            ? t("restrictedBody")
            : t("pendingBody", { days: daysLeft ?? "?" })}
        </p>
      </div>
      <Link
        href="/payment-info"
        className={
          isRestricted
            ? "inline-flex shrink-0 items-center gap-1 rounded-lg bg-red-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-red-700 transition-colors"
            : "inline-flex shrink-0 items-center gap-1 rounded-lg bg-amber-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-amber-700 transition-colors"
        }
      >
        {t("cta")}
        <ArrowRight className="h-3.5 w-3.5" aria-hidden />
      </Link>
    </div>
  )
}

function daysUntil(deadline: string): number {
  const diff = new Date(deadline).getTime() - Date.now()
  return Math.max(0, Math.ceil(diff / (1000 * 60 * 60 * 24)))
}
