"use client"

import {
  CheckCircle2,
  XCircle,
  DollarSign,
  CreditCard,
  Clock,
  RotateCcw,
  Pencil,
  Star,
  Trophy,
  ArrowRight,
} from "lucide-react"
import { useRouter } from "@i18n/navigation"
import { useTranslations } from "next-intl"
import { cn, formatCurrency } from "@/shared/lib/utils"
import type { ProposalMessageMetadata } from "../types"

type SystemMessageConfig = {
  icon: React.ElementType
  iconColor: string
  iconBg: string
  cardBg: string
  cardBorder: string
}

const SYSTEM_MESSAGE_STYLES: Record<string, SystemMessageConfig> = {
  proposal_accepted: {
    icon: CheckCircle2,
    iconColor: "text-green-600 dark:text-green-400",
    iconBg: "bg-green-100 dark:bg-green-500/20",
    cardBg: "bg-green-50 dark:bg-green-900/20",
    cardBorder: "border-green-200 dark:border-green-800",
  },
  proposal_declined: {
    icon: XCircle,
    iconColor: "text-red-600 dark:text-red-400",
    iconBg: "bg-red-100 dark:bg-red-500/20",
    cardBg: "bg-red-50 dark:bg-red-900/20",
    cardBorder: "border-red-200 dark:border-red-800",
  },
  proposal_paid: {
    icon: DollarSign,
    iconColor: "text-blue-600 dark:text-blue-400",
    iconBg: "bg-blue-100 dark:bg-blue-500/20",
    cardBg: "bg-blue-50 dark:bg-blue-900/20",
    cardBorder: "border-blue-200 dark:border-blue-800",
  },
  proposal_payment_requested: {
    icon: CreditCard,
    iconColor: "text-blue-600 dark:text-blue-400",
    iconBg: "bg-blue-100 dark:bg-blue-500/20",
    cardBg: "bg-blue-50 dark:bg-blue-900/20",
    cardBorder: "border-blue-200 dark:border-blue-800",
  },
  proposal_completion_requested: {
    icon: Clock,
    iconColor: "text-amber-600 dark:text-amber-400",
    iconBg: "bg-amber-100 dark:bg-amber-500/20",
    cardBg: "bg-amber-50 dark:bg-amber-900/20",
    cardBorder: "border-amber-200 dark:border-amber-800",
  },
  proposal_completed: {
    icon: Trophy,
    iconColor: "text-emerald-600 dark:text-emerald-400",
    iconBg: "bg-emerald-100 dark:bg-emerald-500/20",
    cardBg: "bg-emerald-50 dark:bg-emerald-900/20",
    cardBorder: "border-emerald-200 dark:border-emerald-800",
  },
  proposal_completion_rejected: {
    icon: RotateCcw,
    iconColor: "text-slate-600 dark:text-slate-400",
    iconBg: "bg-slate-100 dark:bg-slate-500/20",
    cardBg: "bg-slate-50 dark:bg-slate-800/50",
    cardBorder: "border-slate-200 dark:border-slate-700",
  },
  proposal_modified: {
    icon: Pencil,
    iconColor: "text-purple-600 dark:text-purple-400",
    iconBg: "bg-purple-100 dark:bg-purple-500/20",
    cardBg: "bg-purple-50 dark:bg-purple-900/20",
    cardBorder: "border-purple-200 dark:border-purple-800",
  },
  evaluation_request: {
    icon: Star,
    iconColor: "text-amber-600 dark:text-amber-400",
    iconBg: "bg-amber-100 dark:bg-amber-500/20",
    cardBg: "bg-amber-50 dark:bg-amber-900/20",
    cardBorder: "border-amber-200 dark:border-amber-800",
  },
}

function getSystemMessageTitle(type: string, t: ReturnType<typeof useTranslations<"proposal">>) {
  const titles: Record<string, string> = {
    proposal_accepted: t("systemAccepted"),
    proposal_declined: t("systemDeclined"),
    proposal_paid: t("systemPaid"),
    proposal_completed: t("systemCompleted"),
    proposal_completion_requested: t("systemCompletionRequested"),
    proposal_completion_rejected: t("systemCompletionRejected"),
    proposal_modified: t("systemModified"),
    proposal_payment_requested: t("systemPaymentRequested"),
    evaluation_request: t("systemEvaluationRequest"),
  }
  return titles[type] ?? ""
}

export function ProposalSystemMessage({
  type,
  metadata,
}: {
  type: string
  metadata: ProposalMessageMetadata
}) {
  const t = useTranslations("proposal")
  const config = SYSTEM_MESSAGE_STYLES[type]
  if (!config) return null

  const Icon = config.icon
  const title = getSystemMessageTitle(type, t)
  const subtitle = `${metadata.proposal_title} — ${formatCurrency(metadata.proposal_amount / 100)}`

  return (
    <div className="flex justify-center py-2">
      <div
        className={cn(
          "w-full max-w-[400px] rounded-xl border p-4 animate-scale-in",
          config.cardBg,
          config.cardBorder,
        )}
      >
        <div className="flex items-start gap-3">
          <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full", config.iconBg)}>
            <Icon className={cn("h-4.5 w-4.5", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>
              {title}
            </p>
            <p className="mt-0.5 text-xs text-slate-600 dark:text-slate-400 truncate">
              {subtitle}
            </p>
          </div>
        </div>
      </div>
    </div>
  )
}

export function PaymentRequestedMessage({
  metadata,
  currentUserId,
}: {
  metadata: ProposalMessageMetadata
  currentUserId: string
}) {
  const t = useTranslations("proposal")
  const router = useRouter()
  const config = SYSTEM_MESSAGE_STYLES.proposal_payment_requested
  const Icon = config.icon
  const title = getSystemMessageTitle("proposal_payment_requested", t)
  const subtitle = `${metadata.proposal_title} — ${formatCurrency(metadata.proposal_amount / 100)}`

  return (
    <div className="flex justify-center py-2">
      <div
        className={cn(
          "w-full max-w-[400px] rounded-xl border p-4 animate-scale-in",
          config.cardBg,
          config.cardBorder,
        )}
      >
        <div className="flex items-start gap-3">
          <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full", config.iconBg)}>
            <Icon className={cn("h-4.5 w-4.5", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>
              {title}
            </p>
            <p className="mt-0.5 text-xs text-slate-600 dark:text-slate-400 truncate">
              {subtitle}
            </p>
          </div>
        </div>
        {metadata.proposal_client_id === currentUserId && (
          <>
            <div className="mt-3 border-t border-inherit" />
            <button
              type="button"
              onClick={() => router.push(`/projects/pay?proposal=${metadata.proposal_id}`)}
              className={cn(
                "mt-3 w-full flex items-center justify-center gap-2 rounded-lg px-4 py-2",
                "text-sm font-semibold text-white transition-all duration-200",
                "bg-gradient-to-r from-rose-500 to-rose-600",
                "hover:shadow-glow active:scale-[0.98]",
              )}
            >
              {t("payNow")}
              <ArrowRight className="h-4 w-4" strokeWidth={1.5} />
            </button>
          </>
        )}
      </div>
    </div>
  )
}

export function CompletionRequestedMessage({
  metadata,
  currentUserId,
}: {
  metadata: ProposalMessageMetadata
  currentUserId: string
}) {
  const t = useTranslations("proposal")
  const router = useRouter()
  const config = SYSTEM_MESSAGE_STYLES.proposal_completion_requested
  const Icon = config.icon
  const title = getSystemMessageTitle("proposal_completion_requested", t)
  const subtitle = `${metadata.proposal_title} — ${formatCurrency(metadata.proposal_amount / 100)}`

  return (
    <div className="flex justify-center py-2">
      <div
        className={cn(
          "w-full max-w-[400px] rounded-xl border p-4 animate-scale-in",
          config.cardBg,
          config.cardBorder,
        )}
      >
        <div className="flex items-start gap-3">
          <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full", config.iconBg)}>
            <Icon className={cn("h-4.5 w-4.5", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>
              {title}
            </p>
            <p className="mt-0.5 text-xs text-slate-600 dark:text-slate-400 truncate">
              {subtitle}
            </p>
          </div>
        </div>
        {metadata.proposal_client_id === currentUserId && (
          <>
            <div className="mt-3 border-t border-inherit" />
            <button
              type="button"
              onClick={() => router.push(`/projects/${metadata.proposal_id}`)}
              className={cn(
                "mt-3 w-full flex items-center justify-center gap-2 rounded-lg px-4 py-2",
                "text-sm font-semibold text-white transition-all duration-200",
                "bg-gradient-to-r from-rose-500 to-rose-600",
                "hover:shadow-glow active:scale-[0.98]",
              )}
            >
              {t("viewDetails")}
              <ArrowRight className="h-4 w-4" strokeWidth={1.5} />
            </button>
          </>
        )}
      </div>
    </div>
  )
}

export function EvaluationRequestMessage({
  metadata,
  onReview,
}: {
  metadata: ProposalMessageMetadata
  onReview?: (proposalId: string, proposalTitle: string) => void
}) {
  const t = useTranslations("review")
  const tp = useTranslations("proposal")
  const config = SYSTEM_MESSAGE_STYLES.evaluation_request
  const Icon = config.icon
  const title = tp("systemEvaluationRequest")

  return (
    <div className="flex justify-center py-2">
      <div
        className={cn(
          "w-full max-w-[400px] rounded-xl border p-4 animate-scale-in",
          config.cardBg,
          config.cardBorder,
        )}
      >
        <div className="flex items-start gap-3">
          <div className={cn("flex h-9 w-9 shrink-0 items-center justify-center rounded-full", config.iconBg)}>
            <Icon className={cn("h-4.5 w-4.5", config.iconColor)} strokeWidth={1.5} />
          </div>
          <div className="min-w-0 flex-1">
            <p className={cn("text-sm font-semibold", config.iconColor)}>
              {t("evaluationRequest")}
            </p>
            <p className="mt-0.5 text-xs text-slate-600 dark:text-slate-400 truncate">
              {title}
            </p>
          </div>
        </div>
        <div className="mt-3 border-t border-inherit" />
        <button
          type="button"
          onClick={() => onReview?.(metadata.proposal_id, metadata.proposal_title)}
          className={cn(
            "mt-3 w-full flex items-center justify-center gap-2 rounded-lg px-4 py-2",
            "text-sm font-semibold text-white transition-all duration-200",
            "bg-gradient-to-r from-rose-500 to-rose-600",
            "hover:shadow-glow active:scale-[0.98]",
          )}
        >
          {t("leaveReview")}
          <ArrowRight className="h-4 w-4" strokeWidth={1.5} />
        </button>
      </div>
    </div>
  )
}
