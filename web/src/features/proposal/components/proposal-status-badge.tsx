"use client"

import {
  CheckCircle2,
  Clock,
  DollarSign,
  Star,
  XCircle,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { ProposalStatus } from "../types"

export function StatusBadge({ status }: { status: ProposalStatus }) {
  const t = useTranslations("proposal")

  const config: Record<string, { label: string; icon: React.ElementType; className: string }> = {
    pending: {
      label: t("pending"),
      icon: Clock,
      className: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
    },
    accepted: {
      label: t("accepted"),
      icon: CheckCircle2,
      className: "bg-green-50 text-green-700 dark:bg-green-500/10 dark:text-green-400",
    },
    declined: {
      label: t("declined"),
      icon: XCircle,
      className: "bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-400",
    },
    withdrawn: {
      label: t("withdrawn"),
      icon: XCircle,
      className: "bg-gray-50 text-gray-600 dark:bg-gray-500/10 dark:text-gray-400",
    },
    paid: {
      label: t("paid"),
      icon: DollarSign,
      className: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-400",
    },
    active: {
      label: t("active"),
      icon: Star,
      className: "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-400",
    },
    completion_requested: {
      label: t("completionRequested"),
      icon: Clock,
      className: "bg-amber-50 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400",
    },
    completed: {
      label: t("completed"),
      icon: CheckCircle2,
      className: "bg-gray-50 text-gray-600 dark:bg-gray-500/10 dark:text-gray-400",
    },
  }

  const entry = config[status] ?? config.pending
  const { label, icon: StatusIcon, className } = entry

  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-1 rounded-full px-3 py-1 text-xs font-medium",
        className,
      )}
    >
      <StatusIcon className="h-3.5 w-3.5" strokeWidth={2} />
      {label}
    </span>
  )
}

export function DetailSkeleton() {
  return (
    <div className="mx-auto max-w-2xl px-4 py-8">
      <div className="h-5 w-32 animate-pulse rounded bg-gray-200 dark:bg-gray-700 mb-6" />
      <div className="rounded-2xl border border-gray-200 bg-white dark:border-gray-700 dark:bg-gray-800/80 overflow-hidden">
        <div className="h-1.5 animate-pulse bg-gray-200 dark:bg-gray-700" />
        <div className="px-6 pt-6 pb-8 space-y-6">
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-3">
              <div className="h-12 w-12 animate-pulse rounded-xl bg-gray-200 dark:bg-gray-700" />
              <div className="space-y-2">
                <div className="h-5 w-48 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
                <div className="h-3 w-24 animate-pulse rounded bg-gray-100 dark:bg-gray-700" />
              </div>
            </div>
            <div className="h-6 w-20 animate-pulse rounded-full bg-gray-200 dark:bg-gray-700" />
          </div>
          <div className="border-t border-gray-100 dark:border-gray-700" />
          <div className="h-8 w-32 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
          <div className="border-t border-gray-100 dark:border-gray-700" />
          <div className="space-y-2">
            <div className="h-3 w-full animate-pulse rounded bg-gray-100 dark:bg-gray-700" />
            <div className="h-3 w-3/4 animate-pulse rounded bg-gray-100 dark:bg-gray-700" />
            <div className="h-3 w-1/2 animate-pulse rounded bg-gray-100 dark:bg-gray-700" />
          </div>
        </div>
      </div>
    </div>
  )
}
