"use client"

import {
  AlertTriangle,
  CheckCircle2,
  Clock,
  CreditCard,
  Loader2,
  MessageSquare,
  Pencil,
  Star,
  XCircle,
} from "lucide-react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import type { MilestoneResponse, ProposalResponse, ProposalStatus } from "../types"
import { Button } from "@/shared/components/ui/button"

export interface ActionsPanelProps {
  proposal: ProposalResponse
  // currentMilestone is the milestone whose sequence matches
  // proposal.current_milestone_sequence — the one the backend would
  // operate on for any per-milestone action. The button visibility
  // logic depends on its status, not only on the proposal macro
  // status: with multi-milestone proposals the macro stays "active"
  // while the cursor advances through pending_funding → funded →
  // submitted, and each sub-state owns a different CTA.
  currentMilestone: MilestoneResponse | undefined
  isRecipient: boolean
  isSender: boolean
  isClient: boolean
  isProvider: boolean
  isMutating: boolean
  acceptPending: boolean
  declinePending: boolean
  requestCompletionPending: boolean
  completePending: boolean
  rejectCompletionPending: boolean
  onAccept: () => void
  onDecline: () => void
  onModify: () => void
  onPay: () => void
  onRequestCompletion: () => void
  onCompleteProposal: () => void
  onRejectCompletion: () => void
}

export function ActionsPanel(props: ActionsPanelProps) {
  const canRespond = useHasPermission("proposals.respond")

  return (
    <div className="space-y-4">
      <StatusCard status={props.proposal.status} />
      {canRespond && <ActionButtons {...props} />}
      <ConversationLink conversationId={props.proposal.conversation_id} />
    </div>
  )
}

function StatusCard({ status }: { status: ProposalStatus }) {
  const t = useTranslations("proposal")
  const config = getStatusConfig(status, t)

  return (
    <div
      className={cn(
        "rounded-xl p-4 text-center",
        config.bgClass,
      )}
    >
      <config.icon
        className={cn("mx-auto h-6 w-6 mb-2", config.iconClass)}
        strokeWidth={1.5}
      />
      <p className={cn("text-sm font-semibold", config.textClass)}>
        {config.label}
      </p>
      {config.subtitle && (
        <p className={cn("mt-1 text-xs", config.subtitleClass)}>
          {config.subtitle}
        </p>
      )}
    </div>
  )
}

interface StatusConfig {
  icon: React.ElementType
  label: string
  subtitle?: string
  bgClass: string
  iconClass: string
  textClass: string
  subtitleClass: string
}

function getStatusConfig(
  status: ProposalStatus,
  t: ReturnType<typeof useTranslations<"proposal">>,
): StatusConfig {
  const configs: Record<ProposalStatus, StatusConfig> = {
    pending: {
      icon: Clock,
      label: t("pending"),
      subtitle: t("waitingForResponse"),
      bgClass: "bg-amber-50 dark:bg-amber-500/10",
      iconClass: "text-amber-600 dark:text-amber-400",
      textClass: "text-amber-700 dark:text-amber-300",
      subtitleClass: "text-amber-600/70 dark:text-amber-400/70",
    },
    accepted: {
      icon: CheckCircle2,
      label: t("accepted"),
      subtitle: t("waitingPayment"),
      bgClass: "bg-green-50 dark:bg-green-500/10",
      iconClass: "text-green-600 dark:text-green-400",
      textClass: "text-green-700 dark:text-green-300",
      subtitleClass: "text-green-600/70 dark:text-green-400/70",
    },
    paid: {
      icon: CreditCard,
      label: t("paid"),
      bgClass: "bg-blue-50 dark:bg-blue-500/10",
      iconClass: "text-blue-600 dark:text-blue-400",
      textClass: "text-blue-700 dark:text-blue-300",
      subtitleClass: "text-blue-600/70 dark:text-blue-400/70",
    },
    active: {
      icon: Star,
      label: t("active"),
      subtitle: t("missionActive"),
      bgClass: "bg-emerald-50 dark:bg-emerald-500/10",
      iconClass: "text-emerald-600 dark:text-emerald-400",
      textClass: "text-emerald-700 dark:text-emerald-300",
      subtitleClass: "text-emerald-600/70 dark:text-emerald-400/70",
    },
    completion_requested: {
      icon: Clock,
      label: t("completionRequested"),
      subtitle: t("awaitingCompletion"),
      bgClass: "bg-amber-50 dark:bg-amber-500/10",
      iconClass: "text-amber-600 dark:text-amber-400",
      textClass: "text-amber-700 dark:text-amber-300",
      subtitleClass: "text-amber-600/70 dark:text-amber-400/70",
    },
    completed: {
      icon: CheckCircle2,
      label: t("completed"),
      bgClass: "bg-slate-50 dark:bg-slate-500/10",
      iconClass: "text-slate-600 dark:text-slate-400",
      textClass: "text-slate-700 dark:text-slate-300",
      subtitleClass: "text-slate-500 dark:text-slate-400",
    },
    declined: {
      icon: XCircle,
      label: t("declined"),
      bgClass: "bg-red-50 dark:bg-red-500/10",
      iconClass: "text-red-600 dark:text-red-400",
      textClass: "text-red-700 dark:text-red-300",
      subtitleClass: "text-red-500 dark:text-red-400",
    },
    withdrawn: {
      icon: XCircle,
      label: t("withdrawn"),
      bgClass: "bg-slate-50 dark:bg-slate-500/10",
      iconClass: "text-slate-500 dark:text-slate-400",
      textClass: "text-slate-600 dark:text-slate-400",
      subtitleClass: "text-slate-500 dark:text-slate-400",
    },
    disputed: {
      icon: AlertTriangle,
      label: t("disputed"),
      subtitle: t("disputeInProgress"),
      bgClass: "bg-orange-50 dark:bg-orange-500/10",
      iconClass: "text-orange-600 dark:text-orange-400",
      textClass: "text-orange-700 dark:text-orange-300",
      subtitleClass: "text-orange-500 dark:text-orange-400",
    },
  }

  return configs[status]
}

function ActionButtons({
  proposal,
  currentMilestone,
  isRecipient,
  isSender: _isSender,
  isClient,
  isProvider,
  isMutating,
  acceptPending,
  declinePending,
  requestCompletionPending,
  completePending,
  rejectCompletionPending,
  onAccept,
  onDecline,
  onModify,
  onPay,
  onRequestCompletion,
  onCompleteProposal,
  onRejectCompletion,
}: ActionsPanelProps) {
  const t = useTranslations("proposal")

  // Pending - recipient can accept/decline/modify
  if (proposal.status === "pending" && isRecipient) {
    return (
      <div className="space-y-2">
        <PrimaryButton
          onClick={onAccept}
          disabled={isMutating}
          pending={acceptPending}
          icon={CheckCircle2}
          label={t("accept")}
        />
        <OutlineButton
          onClick={onModify}
          disabled={isMutating}
          icon={Pencil}
          label={t("modify")}
        />
        <GhostDestructiveButton
          onClick={onDecline}
          disabled={isMutating}
          pending={declinePending}
          icon={XCircle}
          label={t("decline")}
        />
      </div>
    )
  }

  // Accepted - client can proceed to payment (first milestone funding)
  if (proposal.status === "accepted" && isClient) {
    return (
      <PrimaryButton
        onClick={onPay}
        disabled={false}
        pending={false}
        icon={CreditCard}
        label={t("proceedToPayment")}
      />
    )
  }

  // Active state — the macro status stays "active" while the cursor
  // walks from milestone N → milestone N+1. The right CTA depends on
  // what the CURRENT milestone needs next:
  //   - pending_funding → client funds it (multi-milestone pay flow)
  //   - funded          → provider submits it for approval
  // We fall through to the legacy "no CTA" branch for any other
  // sub-state so stale client views never render a broken button.
  if (proposal.status === "active") {
    if (
      isClient &&
      currentMilestone &&
      currentMilestone.status === "pending_funding"
    ) {
      return (
        <PrimaryButton
          onClick={onPay}
          disabled={false}
          pending={false}
          icon={CreditCard}
          label={t("proceedToPayment")}
        />
      )
    }
    if (
      isProvider &&
      currentMilestone &&
      currentMilestone.status === "funded"
    ) {
      return (
        <PrimaryButton
          onClick={onRequestCompletion}
          disabled={isMutating}
          pending={requestCompletionPending}
          icon={CheckCircle2}
          label={t("terminateMission")}
        />
      )
    }
  }

  // Completion requested - client can confirm or reject
  if (proposal.status === "completion_requested" && isClient) {
    return (
      <div className="space-y-2">
        <PrimaryButton
          onClick={onCompleteProposal}
          disabled={isMutating}
          pending={completePending}
          icon={CheckCircle2}
          label={t("confirmCompletion")}
        />
        <GhostDestructiveButton
          onClick={onRejectCompletion}
          disabled={isMutating}
          pending={rejectCompletionPending}
          icon={XCircle}
          label={t("rejectCompletion")}
        />
      </div>
    )
  }

  return null
}

interface ButtonProps {
  onClick: () => void
  disabled: boolean
  pending?: boolean
  icon: React.ElementType
  label: string
}

function PrimaryButton({ onClick, disabled, pending, icon: Icon, label }: ButtonProps) {
  return (
    <Button variant="ghost" size="auto"
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "w-full flex items-center justify-center gap-2 rounded-xl px-4 py-3",
        "text-sm font-semibold text-white transition-all duration-200",
        "gradient-primary hover:shadow-glow active:scale-[0.98]",
        "disabled:opacity-50 disabled:cursor-not-allowed",
      )}
    >
      {pending ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : (
        <Icon className="h-4 w-4" strokeWidth={1.5} />
      )}
      {label}
    </Button>
  )
}

function OutlineButton({ onClick, disabled, icon: Icon, label }: ButtonProps) {
  return (
    <Button variant="ghost" size="auto"
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "w-full flex items-center justify-center gap-2 rounded-xl px-4 py-3",
        "text-sm font-medium transition-all duration-200",
        "border border-slate-200 dark:border-slate-600",
        "text-slate-700 dark:text-slate-300",
        "hover:bg-slate-50 dark:hover:bg-slate-700 hover:border-slate-300",
        "active:scale-[0.98]",
        "disabled:opacity-50 disabled:cursor-not-allowed",
      )}
    >
      <Icon className="h-4 w-4" strokeWidth={1.5} />
      {label}
    </Button>
  )
}

function GhostDestructiveButton({ onClick, disabled, pending, icon: Icon, label }: ButtonProps) {
  return (
    <Button variant="ghost" size="auto"
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "w-full flex items-center justify-center gap-2 rounded-xl px-4 py-3",
        "text-sm font-medium transition-all duration-200",
        "text-red-600 dark:text-red-400",
        "hover:bg-red-50 dark:hover:bg-red-500/10",
        "active:scale-[0.98]",
        "disabled:opacity-50 disabled:cursor-not-allowed",
      )}
    >
      {pending ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : (
        <Icon className="h-4 w-4" strokeWidth={1.5} />
      )}
      {label}
    </Button>
  )
}

function ConversationLink({ conversationId }: { conversationId: string }) {
  const t = useTranslations("proposal")

  return (
    <Link
      href={`/messages?id=${conversationId}`}
      className={cn(
        "flex items-center justify-center gap-2 rounded-xl px-4 py-2.5",
        "text-sm font-medium transition-all duration-200",
        "text-rose-600 dark:text-rose-400",
        "hover:bg-rose-50 dark:hover:bg-rose-500/10",
      )}
    >
      <MessageSquare className="h-4 w-4" strokeWidth={1.5} />
      {t("goToConversation")}
    </Link>
  )
}
