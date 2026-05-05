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

// Soleil v2 — Milestone tracker. Vertical timeline of milestones with
// Soleil status pills (sapin / amber / corail / sable) and a
// progress bar showing % completion of released milestones.

type MilestoneTrackerProps = {
  milestones: MilestoneResponse[]
  paymentMode: PaymentMode
  currentSequence?: number
}

export function MilestoneTracker({
  milestones,
  paymentMode,
  currentSequence,
}: MilestoneTrackerProps) {
  const t = useTranslations("proposal.milestoneTracker")
  const tFlow = useTranslations("proposal")

  if (milestones.length === 0) {
    return null
  }

  // One-time mode collapses to a single cleaner card.
  if (paymentMode === "one_time" && milestones.length === 1) {
    return <CompactSingleMilestone milestone={milestones[0]} />
  }

  // Progress = released / total milestones
  const released = milestones.filter((m) => m.status === "released").length
  const progress = Math.round((released / milestones.length) * 100)

  return (
    <section
      aria-label={t("ariaLabel")}
      className={cn(
        "rounded-2xl border border-border bg-card p-6",
      )}
      style={{ boxShadow: "var(--shadow-card)" }}
    >
      <header className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
        <h2 className="font-serif text-[20px] font-medium tracking-[-0.015em] text-foreground">
          {t("title")}
        </h2>
        <span className="font-mono text-[11px] font-bold uppercase tracking-[0.08em] text-subtle-foreground">
          {t("count", { total: milestones.length })}
        </span>
      </header>

      {/* Progress bar */}
      <div className="mb-6">
        <div className="mb-1.5 flex items-center justify-between">
          <span className="font-mono text-[10.5px] font-bold uppercase tracking-[0.08em] text-primary">
            {tFlow("proposalFlow_list_progress")}
          </span>
          <span className="font-mono text-[12px] font-semibold text-foreground">
            {progress}%
          </span>
        </div>
        <div className="h-1.5 w-full overflow-hidden rounded-full bg-border">
          <div
            className="h-full rounded-full bg-primary transition-all duration-300 ease-out"
            style={{ width: `${progress}%` }}
            aria-hidden="true"
          />
        </div>
      </div>

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
          className="absolute left-[19px] top-12 -bottom-3 w-px bg-border"
        />
      )}

      <div
        className={cn(
          "relative flex gap-4 rounded-2xl border p-4 transition-all duration-200 ease-out",
          isCurrent
            ? "border-primary bg-primary-soft/40"
            : "border-border bg-card",
        )}
      >
        {/* Status icon with circular background */}
        <div
          className={cn(
            "relative z-10 flex h-10 w-10 shrink-0 items-center justify-center rounded-full",
            cfg.iconBg,
          )}
        >
          <cfg.icon className={cn("h-5 w-5", cfg.iconColor)} strokeWidth={1.7} />
        </div>

        <div className="min-w-0 flex-1">
          <div className="mb-1 flex items-baseline justify-between gap-3">
            <h3 className="truncate font-serif text-[15px] font-medium text-foreground">
              {t("milestone")} {milestone.sequence} — {milestone.title}
            </h3>
            <span className="shrink-0 font-mono text-[14px] font-bold text-foreground">
              {amountEuros}&nbsp;&euro;
            </span>
          </div>

          {milestone.description && (
            <p className="mb-2 line-clamp-2 text-[13px] leading-relaxed text-muted-foreground">
              {milestone.description}
            </p>
          )}

          <div className="flex flex-wrap items-center gap-2 text-[11.5px]">
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
              <span className="inline-flex items-center gap-1 rounded-full bg-primary px-2.5 py-0.5 font-bold text-primary-foreground">
                {t("dueNow")}
              </span>
            )}
            {milestone.deadline && (
              <span className="font-mono text-[11px] text-subtle-foreground">
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
        "rounded-2xl border border-border bg-card p-6",
      )}
      style={{ boxShadow: "var(--shadow-card)" }}
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
            <cfg.icon className={cn("h-6 w-6", cfg.iconColor)} strokeWidth={1.7} />
          </div>
          <div>
            <p className="font-mono text-[10.5px] font-bold uppercase tracking-[0.1em] text-primary">
              {t("oneTimePayment")}
            </p>
            <p className="text-[13.5px] font-semibold text-foreground">
              {cfg.label}
            </p>
          </div>
        </div>
        <span className="font-mono text-[26px] font-bold text-foreground">
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
        iconBg: "bg-amber-soft",
        iconColor: "text-warning",
        badgeBg: "bg-amber-soft",
        badgeText: "text-warning",
      }
    case "funded":
      return {
        icon: CircleDot,
        label: t("statusFunded"),
        iconBg: "bg-primary-soft",
        iconColor: "text-primary",
        badgeBg: "bg-primary-soft",
        badgeText: "text-primary-deep",
      }
    case "submitted":
      return {
        icon: Loader2,
        label: t("statusSubmitted"),
        iconBg: "bg-amber-soft",
        iconColor: "text-warning",
        badgeBg: "bg-amber-soft",
        badgeText: "text-warning",
      }
    case "approved":
      return {
        icon: CheckCircle2,
        label: t("statusApproved"),
        iconBg: "bg-success-soft",
        iconColor: "text-success",
        badgeBg: "bg-success-soft",
        badgeText: "text-success",
      }
    case "released":
      return {
        icon: CheckCircle2,
        label: t("statusReleased"),
        iconBg: "bg-success-soft",
        iconColor: "text-success",
        badgeBg: "bg-success-soft",
        badgeText: "text-success",
      }
    case "disputed":
      return {
        icon: AlertTriangle,
        label: t("statusDisputed"),
        iconBg: "bg-amber-soft",
        iconColor: "text-warning",
        badgeBg: "bg-amber-soft",
        badgeText: "text-warning",
      }
    case "cancelled":
      return {
        icon: XCircle,
        label: t("statusCancelled"),
        iconBg: "bg-border",
        iconColor: "text-muted-foreground",
        badgeBg: "bg-border",
        badgeText: "text-muted-foreground",
      }
    case "refunded":
      return {
        icon: XCircle,
        label: t("statusRefunded"),
        iconBg: "bg-primary-soft",
        iconColor: "text-primary-deep",
        badgeBg: "bg-primary-soft",
        badgeText: "text-primary-deep",
      }
    default:
      return {
        icon: Circle,
        label: status,
        iconBg: "bg-border",
        iconColor: "text-muted-foreground",
        badgeBg: "bg-border",
        badgeText: "text-muted-foreground",
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
