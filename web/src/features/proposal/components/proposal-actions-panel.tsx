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

// Soleil v2 — Sticky proposal actions panel. Status card on top, then
// role-aware corail CTA group, then a ghost link to the conversation.

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
    <div className={cn("rounded-2xl p-4 text-center", config.bgClass)}>
      <config.icon
        className={cn("mx-auto h-6 w-6 mb-2", config.iconClass)}
        strokeWidth={1.7}
        aria-hidden="true"
      />
      <p className={cn("font-serif text-[16px] font-medium tracking-[-0.01em]", config.textClass)}>
        {config.label}
      </p>
      {config.subtitle && (
        <p className={cn("mt-1 text-[12px]", config.subtitleClass)}>
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
      bgClass: "bg-amber-soft",
      iconClass: "text-warning",
      textClass: "text-warning",
      subtitleClass: "text-warning/80",
    },
    accepted: {
      icon: CheckCircle2,
      label: t("accepted"),
      subtitle: t("waitingPayment"),
      bgClass: "bg-success-soft",
      iconClass: "text-success",
      textClass: "text-success",
      subtitleClass: "text-success/80",
    },
    paid: {
      icon: CreditCard,
      label: t("paid"),
      bgClass: "bg-primary-soft",
      iconClass: "text-primary",
      textClass: "text-primary-deep",
      subtitleClass: "text-primary-deep/80",
    },
    active: {
      icon: Star,
      label: t("active"),
      subtitle: t("missionActive"),
      bgClass: "bg-success-soft",
      iconClass: "text-success",
      textClass: "text-success",
      subtitleClass: "text-success/80",
    },
    completion_requested: {
      icon: Clock,
      label: t("completionRequested"),
      subtitle: t("awaitingCompletion"),
      bgClass: "bg-amber-soft",
      iconClass: "text-warning",
      textClass: "text-warning",
      subtitleClass: "text-warning/80",
    },
    completed: {
      icon: CheckCircle2,
      label: t("completed"),
      bgClass: "bg-border/40",
      iconClass: "text-muted-foreground",
      textClass: "text-foreground",
      subtitleClass: "text-muted-foreground",
    },
    declined: {
      icon: XCircle,
      label: t("declined"),
      bgClass: "bg-primary-soft",
      iconClass: "text-destructive",
      textClass: "text-destructive",
      subtitleClass: "text-destructive/80",
    },
    withdrawn: {
      icon: XCircle,
      label: t("withdrawn"),
      bgClass: "bg-border/40",
      iconClass: "text-muted-foreground",
      textClass: "text-foreground",
      subtitleClass: "text-muted-foreground",
    },
    disputed: {
      icon: AlertTriangle,
      label: t("disputed"),
      subtitle: t("disputeInProgress"),
      bgClass: "bg-amber-soft",
      iconClass: "text-warning",
      textClass: "text-warning",
      subtitleClass: "text-warning/80",
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

  // Active state — milestone sub-state determines the right CTA.
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
    <Button
      variant="ghost"
      size="auto"
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "flex w-full items-center justify-center gap-2 rounded-full px-5 py-3",
        "text-[13.5px] font-bold text-primary-foreground transition-all duration-200 ease-out",
        "bg-primary hover:bg-primary-deep hover:shadow-[0_4px_14px_rgba(232,93,74,0.28)]",
        "active:scale-[0.98]",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        "disabled:cursor-not-allowed disabled:opacity-60",
      )}
    >
      {pending ? (
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
      ) : (
        <Icon className="h-4 w-4" strokeWidth={1.7} aria-hidden="true" />
      )}
      {label}
    </Button>
  )
}

function OutlineButton({ onClick, disabled, icon: Icon, label }: ButtonProps) {
  return (
    <Button
      variant="ghost"
      size="auto"
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "flex w-full items-center justify-center gap-2 rounded-full px-5 py-3",
        "text-[13.5px] font-medium transition-all duration-200 ease-out",
        "border border-border-strong bg-card text-foreground",
        "hover:border-primary hover:bg-primary-soft hover:text-primary-deep",
        "active:scale-[0.98]",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        "disabled:cursor-not-allowed disabled:opacity-60",
      )}
    >
      <Icon className="h-4 w-4" strokeWidth={1.7} aria-hidden="true" />
      {label}
    </Button>
  )
}

function GhostDestructiveButton({ onClick, disabled, pending, icon: Icon, label }: ButtonProps) {
  return (
    <Button
      variant="ghost"
      size="auto"
      type="button"
      onClick={onClick}
      disabled={disabled}
      className={cn(
        "flex w-full items-center justify-center gap-2 rounded-full px-5 py-3",
        "text-[13.5px] font-medium transition-all duration-200 ease-out",
        "text-destructive hover:bg-destructive/10",
        "active:scale-[0.98]",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        "disabled:cursor-not-allowed disabled:opacity-60",
      )}
    >
      {pending ? (
        <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
      ) : (
        <Icon className="h-4 w-4" strokeWidth={1.7} aria-hidden="true" />
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
        "flex items-center justify-center gap-2 rounded-full px-5 py-2.5",
        "text-[13.5px] font-medium transition-colors duration-150",
        "text-primary hover:bg-primary-soft",
      )}
    >
      <MessageSquare className="h-4 w-4" strokeWidth={1.7} aria-hidden="true" />
      {t("goToConversation")}
    </Link>
  )
}
