"use client"

import { Phone, PhoneMissed } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import {
  ReferralSystemMessage,
  type ReferralSystemMessageMetadata,
} from "@/shared/components/referral/referral-system-message"
import type { Message, ProposalMessageMetadata } from "../types"
import { ProposalCard } from "./proposal-card"
import {
  CompletionRequestedMessage,
  EvaluationRequestMessage,
  PaymentRequestedMessage,
  ProposalSystemMessage,
} from "./proposal-system-message"
import { DisputeSystemBubble } from "./dispute-system-message"
import {
  DISPUTE_SYSTEM_TYPES,
  PROPOSAL_SYSTEM_TYPES,
  REFERRAL_SYSTEM_TYPES,
  isProposalMetadata,
} from "./message-area-utils"
import {
  TextMessageBubble,
  type TextBubbleActions,
} from "./text-message-bubble"

// MessageBubble dispatches a single Message to the right renderer
// based on its `type` discriminator. This file only owns the switch
// logic — the actual rendering of each branch lives in its own
// component (or in TextMessageBubble for the text/file/voice case).
//
// To keep the component prop count ≤ 4 (per CLAUDE.md), the
// interactive callbacks are grouped behind `actions` and the
// proposal-timeline state behind `state`.

export interface MessageBubbleState {
  isOwn: boolean
  currentUserId: string
  conversationId: string
  supersededProposalIds: Set<string>
}

export interface MessageBubbleActions extends TextBubbleActions {
  onReview?: (
    proposalId: string,
    proposalTitle: string,
    participants: {
      clientOrganizationId: string
      providerOrganizationId: string
    },
  ) => void
}

export interface MessageBubbleProps {
  message: Message
  state: MessageBubbleState
  actions: MessageBubbleActions
}

export function MessageBubble({
  message,
  state,
  actions,
}: MessageBubbleProps) {
  const t = useTranslations("messaging")
  const tp = useTranslations("proposal")

  // Proposal sent / modified — render the rich ProposalCard.
  if (
    (message.type === "proposal_sent" || message.type === "proposal_modified") &&
    isProposalMetadata(message.metadata)
  ) {
    const meta = message.metadata as ProposalMessageMetadata
    const isSuperseded = state.supersededProposalIds.has(meta.proposal_id)

    return (
      <div className={cn("flex flex-col gap-1", state.isOwn ? "items-end" : "items-start")}>
        {isSuperseded && (
          <span className="text-[10px] font-medium text-slate-400 dark:text-slate-500 px-2">
            {tp("supersededByVersion", { version: meta.proposal_version + 1 })}
          </span>
        )}
        <div className={cn(isSuperseded && "opacity-40 pointer-events-none")}>
          <ProposalCard
            metadata={message.metadata}
            isOwn={state.isOwn}
            currentUserId={state.currentUserId}
            conversationId={state.conversationId}
          />
        </div>
      </div>
    )
  }

  // System messages for proposal state changes.
  if (PROPOSAL_SYSTEM_TYPES.has(message.type) && isProposalMetadata(message.metadata)) {
    return <ProposalSystemMessage type={message.type} metadata={message.metadata} />
  }

  // Payment requested — system message with action.
  if (message.type === "proposal_payment_requested" && isProposalMetadata(message.metadata)) {
    return (
      <PaymentRequestedMessage
        metadata={message.metadata}
        currentUserId={state.currentUserId}
      />
    )
  }

  // Completion requested — system message with actions for client.
  if (message.type === "proposal_completion_requested" && isProposalMetadata(message.metadata)) {
    return (
      <CompletionRequestedMessage
        metadata={message.metadata}
        currentUserId={state.currentUserId}
      />
    )
  }

  // Evaluation request — system message with "Leave a review" button.
  // Double-blind reviews: the backend dispatches this message to BOTH
  // the client and the provider, so we intentionally do NOT gate on
  // `target_user_id` or role here. The modal derives the correct
  // review side from the viewer's org vs the proposal participants.
  if (message.type === "evaluation_request" && isProposalMetadata(message.metadata)) {
    return (
      <EvaluationRequestMessage
        metadata={message.metadata}
        onReview={actions.onReview}
      />
    )
  }

  // Call system messages.
  if (message.type === "call_ended" || message.type === "call_missed") {
    return <CallSystemBubble message={message} t={t} />
  }

  // Dispute system messages.
  if (DISPUTE_SYSTEM_TYPES.has(message.type)) {
    return (
      <DisputeSystemBubble
        type={message.type}
        metadata={(message.metadata ?? {}) as Record<string, unknown>}
        currentUserId={state.currentUserId}
        conversationId={state.conversationId}
      />
    )
  }

  // Referral (apport d'affaires) system messages — interactive card
  // with accept / reject / negotiate buttons scoped to the viewer's
  // role in the referral.
  if (REFERRAL_SYSTEM_TYPES.has(message.type)) {
    return (
      <ReferralSystemMessage
        type={message.type}
        metadata={(message.metadata ?? {}) as ReferralSystemMessageMetadata}
        content={message.content}
        currentUserId={state.currentUserId}
      />
    )
  }

  // Deleted message — soft tombstone.
  if (message.deleted_at) {
    return (
      <div className={cn("flex", state.isOwn ? "justify-end" : "justify-start")}>
        <div className="max-w-[75%] rounded-2xl bg-slate-100/60 px-4 py-2.5 dark:bg-slate-800/40">
          <p className="text-sm italic text-slate-400 dark:text-slate-500">
            {t("messageDeleted")}
          </p>
        </div>
      </div>
    )
  }

  // Default — text / file / voice rendering with full editor.
  return (
    <TextMessageBubble
      message={message}
      isOwn={state.isOwn}
      actions={{
        onEdit: actions.onEdit,
        onDelete: actions.onDelete,
        onReply: actions.onReply,
        onReport: actions.onReport,
      }}
    />
  )
}

function CallSystemBubble({
  message,
  t,
}: {
  message: Message
  t: ReturnType<typeof useTranslations>
}) {
  const meta = message.metadata as Record<string, unknown> | null
  const duration = meta?.duration as number | undefined
  const isCallMissed = message.type === "call_missed"

  const formatDuration = (secs: number) => {
    const m = Math.floor(secs / 60)
    const s = secs % 60
    return `${m}:${s.toString().padStart(2, "0")}`
  }

  return (
    <div className="flex justify-center py-2">
      <div className="flex items-center gap-2 rounded-full bg-slate-100 px-4 py-1.5 dark:bg-slate-800">
        {isCallMissed ? (
          <PhoneMissed className="h-3.5 w-3.5 text-red-500" />
        ) : (
          <Phone className="h-3.5 w-3.5 text-emerald-500" />
        )}
        <span className="text-xs font-medium text-slate-600 dark:text-slate-400">
          {isCallMissed
            ? t("callMissed")
            : `${t("callEnded")} — ${duration ? formatDuration(duration) : "0:00"}`}
        </span>
      </div>
    </div>
  )
}
