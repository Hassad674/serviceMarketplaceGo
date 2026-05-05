"use client"

import { AlertTriangle, ShieldAlert, ArrowRight } from "lucide-react"
import Link from "next/link"
import { useTranslations } from "next-intl"

import type { CurrentUser } from "@/shared/hooks/use-user"
import { cn } from "@/shared/lib/utils"

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
      className={cn(
        "mb-5 flex items-start gap-3 rounded-2xl border p-4 animate-slide-up",
        isRestricted
          ? "border-destructive/40 bg-primary-soft/60"
          : "border-warning/30 bg-amber-soft",
      )}
    >
      {isRestricted ? (
        <ShieldAlert
          className="mt-0.5 h-5 w-5 shrink-0 text-destructive"
          aria-hidden
        />
      ) : (
        <AlertTriangle
          className="mt-0.5 h-5 w-5 shrink-0 text-warning"
          aria-hidden
        />
      )}
      <div className="flex-1 min-w-0">
        <p className="font-serif text-[16px] font-medium tracking-[-0.01em] text-foreground">
          {isRestricted ? t("restrictedTitle") : t("pendingTitle")}
        </p>
        <p className="mt-1 text-[13px] leading-relaxed text-muted-foreground">
          {isRestricted
            ? t("restrictedBody")
            : t("pendingBody", { days: daysLeft ?? "?" })}
        </p>
      </div>
      <Link
        href="/payment-info"
        className={cn(
          "inline-flex shrink-0 items-center gap-1.5 rounded-full px-4 py-2 text-[12px] font-bold transition-all duration-200 ease-out",
          isRestricted
            ? "bg-destructive text-primary-foreground hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(196,58,38,0.28)]"
            : "bg-primary text-primary-foreground hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)]",
        )}
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
