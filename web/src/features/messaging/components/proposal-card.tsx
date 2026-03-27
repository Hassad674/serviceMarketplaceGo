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
  const isClient = metadata.proposal_client_id === currentUserId

  function handleAccept(e: React.MouseEvent) {
    e.stopPropagation()
    acceptMutation.mutate(metadata.proposal_id, {
      onSuccess: () => {
        if (isClient) {
          router.push(`/projects/pay?proposal=${metadata.proposal_id}`)
        }
      },
    })
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

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={handleViewDetail}
      onKeyDown={(e) => { if (e.key === "Enter") handleViewDetail() }}
      className={cn(
        "w-full max-w-[420px] rounded-2xl border overflow-hidden cursor-pointer",
        "transition-all duration-200",
        "bg-white dark:bg-gray-800/80",
        "border-gray-200 dark:border-gray-700",
        "shadow-sm hover:shadow-md hover:border-rose-200 dark:hover:border-rose-500/30",
      )}
    >
      {/* Header gradient bar */}
      <div className="h-1.5 gradient-primary" />

      {/* Content */}
      <div className="px-5 pt-4 pb-5 space-y-4">
        {/* Header row */}
        <div className="flex items-start justify-between gap-3">
          <div className="flex items-center gap-3 min-w-0">
            <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-rose-100 dark:bg-rose-500/20">
              <Handshake className="h-5 w-5 text-rose-600 dark:text-rose-400" strokeWidth={1.5} />
            </div>
            <div className="min-w-0">
              <p className="text-xs font-medium text-gray-500 dark:text-gray-400">
                {isCounterProposal
                  ? t("counterProposal", { version: metadata.proposal_version })
                  : t("proposalFrom", { name: metadata.proposal_sender_name })}
              </p>
              <h3 className="truncate text-sm font-bold text-gray-900 dark:text-white">
                {metadata.proposal_title}
              </h3>
            </div>
          </div>
          <StatusBadge status={metadata.proposal_status} />
        </div>

        {/* Divider */}
        <div className="border-t border-gray-100 dark:border-gray-700" />

        {/* Details grid */}
        <div className="grid grid-cols-2 gap-3">
          {/* Amount */}
          <DetailItem
            icon={<EuroIcon />}
            label={t("totalAmount")}
            value={formatCurrency(metadata.proposal_amount / 100)}
            highlight
          />

          {/* Deadline */}
          {metadata.proposal_deadline && (
            <DetailItem
              icon={<Calendar className="h-4 w-4" strokeWidth={1.5} />}
              label={t("proposalDeadline")}
              value={new Intl.DateTimeFormat("fr-FR", {
                day: "numeric",
                month: "short",
                year: "numeric",
              }).format(new Date(metadata.proposal_deadline))}
            />
          )}

          {/* Documents count */}
          {metadata.proposal_documents_count > 0 && (
            <DetailItem
              icon={<Paperclip className="h-4 w-4" strokeWidth={1.5} />}
              label={t("proposalDocuments")}
              value={`${metadata.proposal_documents_count}`}
            />
          )}

          {/* Version */}
          {metadata.proposal_version > 1 && (
            <DetailItem
              icon={<Pencil className="h-4 w-4" strokeWidth={1.5} />}
              label={t("version")}
              value={`v${metadata.proposal_version}`}
            />
          )}
        </div>

        {/* Pending action buttons (Accept / Decline) */}
        {showPendingActions && (
          <>
            <div className="border-t border-gray-100 dark:border-gray-700" />
            <div className="flex gap-2">
              <button
                type="button"
                onClick={handleDecline}
                disabled={isMutating}
                className={cn(
                  "flex-1 flex items-center justify-center gap-2 rounded-xl px-4 py-2.5",
                  "text-sm font-medium transition-all duration-200",
                  "border border-gray-200 dark:border-gray-600",
                  "text-gray-700 dark:text-gray-300",
                  "hover:bg-gray-50 dark:hover:bg-gray-700 hover:border-gray-300 dark:hover:border-gray-500",
                  "active:scale-[0.98]",
                  "disabled:opacity-50 disabled:cursor-not-allowed",
                )}
              >
                {declineMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <XCircle className="h-4 w-4" strokeWidth={1.5} />
                )}
                {t("decline")}
              </button>
              <button
                type="button"
                onClick={handleAccept}
                disabled={isMutating}
                className={cn(
                  "flex-1 flex items-center justify-center gap-2 rounded-xl px-4 py-2.5",
                  "text-sm font-semibold text-white transition-all duration-200",
                  "gradient-primary",
                  "hover:shadow-glow active:scale-[0.98]",
                  "disabled:opacity-50 disabled:cursor-not-allowed",
                )}
              >
                {acceptMutation.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <CheckCircle2 className="h-4 w-4" strokeWidth={1.5} />
                )}
                {t("accept")}
              </button>
            </div>
          </>
        )}

        {/* Modify button (sender, pending) */}
        {showModifyButton && (
          <>
            <div className="border-t border-gray-100 dark:border-gray-700" />
            <button
              type="button"
              onClick={handleModify}
              className={cn(
                "w-full flex items-center justify-center gap-2 rounded-xl px-4 py-2.5",
                "text-sm font-medium transition-all duration-200",
                "border border-gray-200 dark:border-gray-600",
                "text-gray-700 dark:text-gray-300",
                "hover:bg-gray-50 dark:hover:bg-gray-700",
                "active:scale-[0.98]",
              )}
            >
              <Pencil className="h-4 w-4" strokeWidth={1.5} />
              {t("modify")}
            </button>
          </>
        )}

        {/* Pay button (client, accepted) */}
        {showPayButton && (
          <>
            <div className="border-t border-gray-100 dark:border-gray-700" />
            <button
              type="button"
              onClick={handlePay}
              className={cn(
                "w-full flex items-center justify-center gap-2 rounded-xl px-4 py-2.5",
                "text-sm font-semibold text-white transition-all duration-200",
                "gradient-primary",
                "hover:shadow-glow active:scale-[0.98]",
              )}
            >
              <CreditCard className="h-4 w-4" strokeWidth={1.5} />
              {t("pay")}
            </button>
          </>
        )}

        {/* View details hint */}
        <div className="flex items-center justify-center gap-1.5 pt-1">
          <ExternalLink className="h-3 w-3 text-gray-400 dark:text-gray-500" strokeWidth={1.5} />
          <span className="text-[10px] font-medium text-gray-400 dark:text-gray-500">
            {t("viewDetails")}
          </span>
        </div>
      </div>
    </div>
  )
}

function EuroIcon() {
  return (
    <span className="flex h-4 w-4 items-center justify-center text-sm font-bold text-current">
      &euro;
    </span>
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
        "inline-flex shrink-0 items-center gap-1 rounded-full px-2.5 py-1 text-xs font-medium",
        className,
      )}
    >
      <StatusIcon className="h-3 w-3" strokeWidth={2} />
      {label}
    </span>
  )
}

type DetailItemProps = {
  icon: React.ReactNode
  label: string
  value: string
  highlight?: boolean
}

function DetailItem({ icon, label, value, highlight }: DetailItemProps) {
  return (
    <div className="flex items-start gap-2">
      <div className="mt-0.5 text-gray-400 dark:text-gray-500">
        {icon}
      </div>
      <div className="min-w-0">
        <p className="text-[10px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
          {label}
        </p>
        <p
          className={cn(
            "truncate text-sm",
            highlight
              ? "font-bold text-gray-900 dark:text-white"
              : "font-medium text-gray-700 dark:text-gray-300",
          )}
        >
          {value}
        </p>
      </div>
    </div>
  )
}
