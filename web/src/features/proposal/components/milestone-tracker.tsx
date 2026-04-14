"use client"

import {
  CheckCircle2,
  Circle,
  CircleDot,
  Loader2,
  AlertTriangle,
  XCircle,
  CreditCard,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { MilestoneResponse, MilestoneStatus, PaymentMode } from "../types"

type MilestoneTrackerProps = {
  milestones: MilestoneResponse[]
  paymentMode: PaymentMode
  currentSequence?: number
}

/**
 * MilestoneTracker renders the project's milestone list as a vertical
 * timeline. Each milestone shows its status icon, title, amount, and
 * (when present) deadline. The current active milestone is highlighted
 * with a rose accent border and a "Due now" badge.
 *
 * For one-time mode (single synthetic milestone) the tracker collapses
 * to a compact single-card layout that mirrors the legacy proposal
 * detail view — users coming from pre-phase-4 proposals see no
 * regression in their UX.
 */
export function MilestoneTracker({
  milestones,
  paymentMode,
  currentSequence,
}: MilestoneTrackerProps) {
  const t = useTranslations("proposal.milestoneTracker")

  if (milestones.length === 0) {
    return null
  }

  // One-time mode collapses to a single cleaner card.
  if (paymentMode === "one_time" && milestones.length === 1) {
    return <CompactSingleMilestone milestone={milestones[0]} />
  }

  return (
    <section
      aria-label={t("ariaLabel")}
      className={cn(
        "rounded-2xl border border-gray-200 bg-white p-6 shadow-sm",
        "dark:border-gray-700 dark:bg-gray-900",
      )}
    >
      <header className="mb-5 flex items-baseline justify-between">
        <h2 className="text-base font-semibold text-gray-900 dark:text-white">
          {t("title")}
        </h2>
        <span className="text-xs text-gray-500 dark:text-gray-400">
          {t("count", { total: milestones.length })}
        </span>
      </header>
      <ol className="space-y-3" role="list">
        {milestones.map((m, index) => (
          <MilestoneCard
            key={m.id}
            milestone={m}
            isCurrent={m.sequence === currentSequence}
            isLast={index === milestones.length - 1}
          />
        ))}
      </ol>
    </section>
  )
}

type MilestoneCardProps = {
  milestone: MilestoneResponse
  isCurrent: boolean
  isLast: boolean
}

function MilestoneCard({ milestone, isCurrent, isLast }: MilestoneCardProps) {
  const t = useTranslations("proposal.milestoneTracker")
  const amountEuros = (milestone.amount / 100).toFixed(2)
  const cfg = milestoneStatusConfig(milestone.status, t)

  return (
    <li className="relative" role="listitem">
      {/* Vertical connector to the next card (skipped on the last) */}
      {!isLast && (
        <span
          aria-hidden="true"
          className="absolute left-[19px] top-12 -bottom-3 w-px bg-gray-200 dark:bg-gray-700"
        />
      )}

      <div
        className={cn(
          "relative flex gap-4 rounded-xl border p-4 transition-all duration-200",
          isCurrent
            ? "border-rose-300 bg-rose-50/30 dark:border-rose-700 dark:bg-rose-900/10"
            : "border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800",
        )}
      >
        {/* Status icon with circular background */}
        <div
          className={cn(
            "relative z-10 flex h-10 w-10 shrink-0 items-center justify-center rounded-full",
            cfg.iconBg,
          )}
        >
          <cfg.icon className={cn("h-5 w-5", cfg.iconColor)} strokeWidth={1.5} />
        </div>

        <div className="min-w-0 flex-1">
          <div className="mb-1 flex items-baseline justify-between gap-3">
            <h3 className="truncate text-sm font-semibold text-gray-900 dark:text-white">
              {t("milestone")} {milestone.sequence} — {milestone.title}
            </h3>
            <span className="shrink-0 text-sm font-bold text-gray-900 dark:text-white">
              {amountEuros}&nbsp;&euro;
            </span>
          </div>

          {milestone.description && (
            <p className="mb-2 line-clamp-2 text-xs text-gray-600 dark:text-gray-400">
              {milestone.description}
            </p>
          )}

          <div className="flex flex-wrap items-center gap-2 text-xs">
            <span
              className={cn(
                "inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 font-medium",
                cfg.badgeBg,
                cfg.badgeText,
              )}
            >
              {cfg.label}
            </span>
            {isCurrent && milestone.status === "pending_funding" && (
              <span className="inline-flex items-center gap-1 rounded-full bg-rose-100 px-2.5 py-0.5 font-semibold text-rose-700 dark:bg-rose-900/40 dark:text-rose-300">
                {t("dueNow")}
              </span>
            )}
            {milestone.deadline && (
              <span className="text-gray-500 dark:text-gray-400">
                {t("dueBy", { date: formatDate(milestone.deadline) })}
              </span>
            )}
          </div>
        </div>
      </div>
    </li>
  )
}

function CompactSingleMilestone({ milestone }: { milestone: MilestoneResponse }) {
  const t = useTranslations("proposal.milestoneTracker")
  const amountEuros = (milestone.amount / 100).toFixed(2)
  const cfg = milestoneStatusConfig(milestone.status, t)

  return (
    <section
      className={cn(
        "rounded-2xl border border-gray-200 bg-white p-6 shadow-sm",
        "dark:border-gray-700 dark:bg-gray-900",
      )}
      aria-label={t("ariaLabel")}
    >
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div
            className={cn(
              "flex h-12 w-12 items-center justify-center rounded-full",
              cfg.iconBg,
            )}
          >
            <cfg.icon className={cn("h-6 w-6", cfg.iconColor)} strokeWidth={1.5} />
          </div>
          <div>
            <p className="text-xs font-medium uppercase tracking-wider text-gray-500 dark:text-gray-400">
              {t("oneTimePayment")}
            </p>
            <p className="text-sm font-semibold text-gray-900 dark:text-white">
              {cfg.label}
            </p>
          </div>
        </div>
        <span className="text-2xl font-bold text-gray-900 dark:text-white">
          {amountEuros}&nbsp;&euro;
        </span>
      </div>
    </section>
  )
}

type StatusConfig = {
  icon: React.ElementType
  label: string
  iconBg: string
  iconColor: string
  badgeBg: string
  badgeText: string
}

function milestoneStatusConfig(
  status: MilestoneStatus,
  t: ReturnType<typeof useTranslations<"proposal.milestoneTracker">>,
): StatusConfig {
  switch (status) {
    case "pending_funding":
      return {
        icon: CreditCard,
        label: t("statusPendingFunding"),
        iconBg: "bg-amber-100 dark:bg-amber-900/30",
        iconColor: "text-amber-600 dark:text-amber-400",
        badgeBg: "bg-amber-50 dark:bg-amber-900/20",
        badgeText: "text-amber-700 dark:text-amber-300",
      }
    case "funded":
      return {
        icon: CircleDot,
        label: t("statusFunded"),
        iconBg: "bg-blue-100 dark:bg-blue-900/30",
        iconColor: "text-blue-600 dark:text-blue-400",
        badgeBg: "bg-blue-50 dark:bg-blue-900/20",
        badgeText: "text-blue-700 dark:text-blue-300",
      }
    case "submitted":
      return {
        icon: Loader2,
        label: t("statusSubmitted"),
        iconBg: "bg-indigo-100 dark:bg-indigo-900/30",
        iconColor: "text-indigo-600 dark:text-indigo-400",
        badgeBg: "bg-indigo-50 dark:bg-indigo-900/20",
        badgeText: "text-indigo-700 dark:text-indigo-300",
      }
    case "approved":
      return {
        icon: Loader2,
        label: t("statusApproved"),
        iconBg: "bg-emerald-100 dark:bg-emerald-900/30",
        iconColor: "text-emerald-600 dark:text-emerald-400",
        badgeBg: "bg-emerald-50 dark:bg-emerald-900/20",
        badgeText: "text-emerald-700 dark:text-emerald-300",
      }
    case "released":
      return {
        icon: CheckCircle2,
        label: t("statusReleased"),
        iconBg: "bg-emerald-100 dark:bg-emerald-900/30",
        iconColor: "text-emerald-600 dark:text-emerald-400",
        badgeBg: "bg-emerald-50 dark:bg-emerald-900/20",
        badgeText: "text-emerald-700 dark:text-emerald-300",
      }
    case "disputed":
      return {
        icon: AlertTriangle,
        label: t("statusDisputed"),
        iconBg: "bg-orange-100 dark:bg-orange-900/30",
        iconColor: "text-orange-600 dark:text-orange-400",
        badgeBg: "bg-orange-50 dark:bg-orange-900/20",
        badgeText: "text-orange-700 dark:text-orange-300",
      }
    case "cancelled":
      return {
        icon: XCircle,
        label: t("statusCancelled"),
        iconBg: "bg-gray-100 dark:bg-gray-800",
        iconColor: "text-gray-500 dark:text-gray-400",
        badgeBg: "bg-gray-100 dark:bg-gray-800",
        badgeText: "text-gray-600 dark:text-gray-400",
      }
    case "refunded":
      return {
        icon: XCircle,
        label: t("statusRefunded"),
        iconBg: "bg-rose-100 dark:bg-rose-900/30",
        iconColor: "text-rose-600 dark:text-rose-400",
        badgeBg: "bg-rose-50 dark:bg-rose-900/20",
        badgeText: "text-rose-700 dark:text-rose-300",
      }
    default:
      return {
        icon: Circle,
        label: status,
        iconBg: "bg-gray-100 dark:bg-gray-800",
        iconColor: "text-gray-500 dark:text-gray-400",
        badgeBg: "bg-gray-100 dark:bg-gray-800",
        badgeText: "text-gray-600 dark:text-gray-400",
      }
  }
}

function formatDate(iso: string): string {
  try {
    return new Date(iso).toLocaleDateString("fr-FR", {
      day: "numeric",
      month: "short",
      year: "numeric",
    })
  } catch {
    return iso
  }
}
