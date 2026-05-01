"use client"

import {
  Handshake,
  CheckCircle2,
  XCircle,
  Clock,
  Calendar,
  Paperclip,
  CreditCard,
  Pencil,
  Loader2,
  DollarSign,
  Star,
  ExternalLink,
} from "lucide-react"
import { useRouter } from "@i18n/navigation"
import { useTranslations } from "next-intl"
import { cn, formatCurrency } from "@/shared/lib/utils"
import type { ProposalMessageMetadata } from "../types"
// Cross-feature import — ProposalCard renders proposal actions within messaging context.
// This is an accepted exception documented in the task spec.
import {
  useAcceptProposal,
  useDeclineProposal,
} from "@/features/proposal/hooks/use-proposals"

type ProposalStatus = ProposalMessageMetadata["proposal_status"]

type ProposalCardProps = {
  metadata: ProposalMessageMetadata
  isOwn: boolean
  currentUserId: string
  conversationId: string
}

const STATUS_BORDER_COLOR: Record<ProposalStatus, string> = {
  pending: "border-l-rose-500",
  accepted: "border-l-green-500",
  declined: "border-l-red-500",
  withdrawn: "border-l-slate-400",
  paid: "border-l-emerald-500",
  active: "border-l-emerald-500",
  completion_requested: "border-l-amber-500",
  completed: "border-l-blue-500",
}

export function ProposalCard({
  metadata,
  isOwn,
  currentUserId,
  conversationId,
}: ProposalCardProps) {
  const t = useTranslations("proposal")
  const router = useRouter()
  const acceptMutation = useAcceptProposal()
  const declineMutation = useDeclineProposal()

  const isMutating = acceptMutation.isPending || declineMutation.isPending
  const isRecipient = !isOwn
  const showPendingActions = isRecipient && metadata.proposal_status === "pending"
  const showPayButton =
    metadata.proposal_status === "accepted" &&
    metadata.proposal_client_id === currentUserId
  const showModifyButton = isOwn && metadata.proposal_status === "pending"

  const isCounterProposal = metadata.proposal_version > 1

  function handleAccept(e: React.MouseEvent) {
    e.stopPropagation()
    acceptMutation.mutate(metadata.proposal_id)
  }

  function handleDecline(e: React.MouseEvent) {
    e.stopPropagation()
    declineMutation.mutate(metadata.proposal_id)
  }

  function handleViewDetail() {
    router.push(`/projects/${metadata.proposal_id}`)
  }

  function handleModify(e: React.MouseEvent) {
    e.stopPropagation()
    const params = new URLSearchParams({
      modify: metadata.proposal_id,
      conversation: conversationId,
      to: isOwn ? metadata.proposal_provider_id : metadata.proposal_client_id,
    })
    router.push(`/projects/new?${params.toString()}`)
  }

  function handlePay(e: React.MouseEvent) {
    e.stopPropagation()
    router.push(`/projects/pay?proposal=${metadata.proposal_id}`)
  }

  const borderColor = STATUS_BORDER_COLOR[metadata.proposal_status] ?? "border-l-rose-500"

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={handleViewDetail}
      onKeyDown={(e) => { if (e.key === "Enter") handleViewDetail() }}
      className={cn(
        "w-full max-w-[420px] rounded-xl border-l-[3px] border overflow-hidden cursor-pointer",
        "transition-all duration-200 animate-fade-in",
        "bg-white dark:bg-slate-800/50",
        "border-slate-200 dark:border-slate-700",
        "hover:shadow-md",
        borderColor,
      )}
    >
      <div className="px-4 pt-3 pb-4 space-y-3">
        <ProposalCardHeader
          metadata={metadata}
          isCounterProposal={isCounterProposal}
        />

        <ProposalCardStats metadata={metadata} />

        <ProposalCardActions
          showPendingActions={showPendingActions}
          showModifyButton={showModifyButton}
          showPayButton={showPayButton}
          isMutating={isMutating}
          acceptPending={acceptMutation.isPending}
          declinePending={declineMutation.isPending}
          onAccept={handleAccept}
          onDecline={handleDecline}
          onModify={handleModify}
          onPay={handlePay}
        />

        <div className="flex items-center justify-center gap-1.5 pt-1">
          <ExternalLink className="h-3 w-3 text-slate-400 dark:text-slate-500" strokeWidth={1.5} />
          <span className="text-[10px] font-medium text-slate-400 dark:text-slate-500">
            {t("viewDetails")}
          </span>
        </div>
      </div>
    </div>
  )
}

function ProposalCardHeader({
  metadata,
  isCounterProposal,
}: {
  metadata: ProposalMessageMetadata
  isCounterProposal: boolean
}) {
  const t = useTranslations("proposal")

  return (
    <div className="flex items-start justify-between gap-3">
      <div className="flex items-center gap-2.5 min-w-0">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-rose-50 dark:bg-rose-500/15">
          <Handshake className="h-4 w-4 text-rose-600 dark:text-rose-400" strokeWidth={1.5} />
        </div>
        <div className="min-w-0">
          <p className="text-xs font-medium text-slate-500 dark:text-slate-400">
            {isCounterProposal
              ? t("counterProposal", { version: metadata.proposal_version })
              : t("proposalFrom", { name: metadata.proposal_sender_name })}
          </p>
          <h3 className="truncate text-sm font-bold text-slate-900 dark:text-white">
            {metadata.proposal_title}
          </h3>
        </div>
      </div>
      <StatusBadge status={metadata.proposal_status} />
    </div>
  )
}

function ProposalCardStats({ metadata }: { metadata: ProposalMessageMetadata }) {
  const t = useTranslations("proposal")

  const deadlineLabel = metadata.proposal_deadline
    ? new Intl.DateTimeFormat("fr-FR", {
        day: "numeric",
        month: "short",
        year: "numeric",
      }).format(new Date(metadata.proposal_deadline))
    : t("noDeadline")

  const docsLabel = metadata.proposal_documents_count > 0
    ? `${metadata.proposal_documents_count}`
    : t("noDocuments")

  return (
    <div className="grid grid-cols-3 gap-2">
      <StatCell
        icon={<DollarSign className="h-3.5 w-3.5" strokeWidth={1.5} />}
        label={t("totalAmount")}
        value={formatCurrency(metadata.proposal_amount / 100)}
      />
      <StatCell
        icon={<Calendar className="h-3.5 w-3.5" strokeWidth={1.5} />}
        label={t("deadline")}
        value={deadlineLabel}
      />
      <StatCell
        icon={<Paperclip className="h-3.5 w-3.5" strokeWidth={1.5} />}
        label={t("documents")}
        value={docsLabel}
      />
    </div>
  )
}

function StatCell({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode
  label: string
  value: string
}) {
  return (
    <div className="flex flex-col items-center gap-1 rounded-lg bg-slate-50 p-2 dark:bg-slate-700/50">
      <span className="text-slate-400 dark:text-slate-500">{icon}</span>
      <span className="text-[10px] font-medium text-slate-400 dark:text-slate-500 uppercase tracking-wide">
        {label}
      </span>
      <span className="text-xs font-semibold text-slate-900 dark:text-white text-center truncate w-full">
        {value}
      </span>
    </div>
  )
}

type ProposalCardActionsProps = {
  showPendingActions: boolean
  showModifyButton: boolean
  showPayButton: boolean
  isMutating: boolean
  acceptPending: boolean
  declinePending: boolean
  onAccept: (e: React.MouseEvent) => void
  onDecline: (e: React.MouseEvent) => void
  onModify: (e: React.MouseEvent) => void
  onPay: (e: React.MouseEvent) => void
}

function ProposalCardActions({
  showPendingActions,
  showModifyButton,
  showPayButton,
  isMutating,
  acceptPending,
  declinePending,
  onAccept,
  onDecline,
  onModify,
  onPay,
}: ProposalCardActionsProps) {
  const t = useTranslations("proposal")

  if (!showPendingActions && !showModifyButton && !showPayButton) {
    return null
  }

  return (
    <>
      <div className="border-t border-slate-100 dark:border-slate-700" />
      <div className="flex gap-2">
        {showPendingActions && (
          <PendingActionButtons
            isMutating={isMutating}
            acceptPending={acceptPending}
            declinePending={declinePending}
            onAccept={onAccept}
            onDecline={onDecline}
          />
        )}

        {showModifyButton && (
          <button
            type="button"
            onClick={onModify}
            className={cn(
              "w-full flex items-center justify-center gap-2 rounded-lg px-4 py-2",
              "text-sm font-medium transition-all duration-200",
              "border border-slate-200 dark:border-slate-600",
              "text-slate-700 dark:text-slate-300",
              "hover:border-rose-300 dark:hover:border-rose-500/30",
              "active:scale-[0.98]",
            )}
          >
            <Pencil className="h-4 w-4" strokeWidth={1.5} />
            {t("modify")}
          </button>
        )}

        {showPayButton && (
          <button
            type="button"
            onClick={onPay}
            className={cn(
              "w-full flex items-center justify-center gap-2 rounded-lg px-4 py-2",
              "text-sm font-semibold text-white transition-all duration-200",
              "bg-gradient-to-r from-rose-500 to-rose-600",
              "hover:shadow-glow active:scale-[0.98]",
            )}
          >
            <CreditCard className="h-4 w-4" strokeWidth={1.5} />
            {t("pay")}
          </button>
        )}
      </div>
    </>
  )
}

function PendingActionButtons({
  isMutating,
  acceptPending,
  declinePending,
  onAccept,
  onDecline,
}: {
  isMutating: boolean
  acceptPending: boolean
  declinePending: boolean
  onAccept: (e: React.MouseEvent) => void
  onDecline: (e: React.MouseEvent) => void
}) {
  const t = useTranslations("proposal")

  return (
    <>
      <button
        type="button"
        onClick={onDecline}
        disabled={isMutating}
        className={cn(
          "flex-1 flex items-center justify-center gap-2 rounded-lg px-4 py-2",
          "text-sm font-medium transition-all duration-200",
          "text-slate-500 hover:text-red-500",
          "active:scale-[0.98]",
          "disabled:opacity-50 disabled:cursor-not-allowed",
        )}
      >
        {declinePending ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <XCircle className="h-4 w-4" strokeWidth={1.5} />
        )}
        {t("decline")}
      </button>
      <button
        type="button"
        onClick={onAccept}
        disabled={isMutating}
        className={cn(
          "flex-1 flex items-center justify-center gap-2 rounded-lg px-4 py-2",
          "text-sm font-semibold text-white transition-all duration-200",
          "bg-gradient-to-r from-rose-500 to-rose-600",
          "hover:shadow-glow active:scale-[0.98]",
          "disabled:opacity-50 disabled:cursor-not-allowed",
        )}
      >
        {acceptPending ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <CheckCircle2 className="h-4 w-4" strokeWidth={1.5} />
        )}
        {t("accept")}
      </button>
    </>
  )
}

type StatusBadgeProps = {
  status: ProposalStatus
}

function StatusBadge({ status }: StatusBadgeProps) {
  const t = useTranslations("proposal")

  const config: Record<ProposalStatus, { label: string; icon: React.ElementType; className: string }> = {
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
      className: "bg-slate-50 text-slate-600 dark:bg-slate-500/10 dark:text-slate-400",
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
      className: "bg-slate-50 text-slate-600 dark:bg-slate-500/10 dark:text-slate-400",
    },
  }

  const entry = config[status] ?? config.pending
  const { label, icon: StatusIcon, className } = entry

  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium",
        className,
      )}
    >
      <StatusIcon className="h-3 w-3" strokeWidth={2} />
      {label}
    </span>
  )
}
