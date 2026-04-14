"use client"

import { useRef, useEffect, useState, useCallback, useMemo } from "react"
import { MessageSquare, Phone, PhoneMissed, Reply, Pencil, Trash2, Flag } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn, formatCurrency } from "@/shared/lib/utils"
import type { Message, ProposalMessageMetadata, ReplyToInfo, VoiceMetadata } from "../types"
import { MessageStatusIcon } from "./message-status-icon"
import { FileMessage } from "./file-message"
import { VoiceMessage } from "./voice-message"
import { MessageContextMenu } from "./message-context-menu"
import { ProposalCard } from "./proposal-card"
import { MessageAreaSkeleton } from "./message-area-skeleton"
import {
  ProposalSystemMessage,
  PaymentRequestedMessage,
  CompletionRequestedMessage,
  EvaluationRequestMessage,
} from "./proposal-system-message"
import { DisputeSystemBubble } from "./dispute-system-message"

interface MessageAreaProps {
  messages: Message[]
  currentUserId: string
  isLoading: boolean
  hasMore: boolean
  onLoadMore: () => void
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
  onReply: (message: Message) => void
  onReport?: (messageId: string) => void
  conversationId: string
  // participants are ORG ids from the proposal's system message
  // metadata. The page uses them together with the viewer's org to
  // derive the review side.
  onReview?: (
    proposalId: string,
    proposalTitle: string,
    participants: { clientOrganizationId: string; providerOrganizationId: string },
  ) => void
}

export function MessageArea({
  messages,
  currentUserId,
  isLoading,
  hasMore,
  onLoadMore,
  onEdit,
  onDelete,
  onReply,
  onReport,
  conversationId,
  onReview,
}: MessageAreaProps) {
  const t = useTranslations("messaging")
  const scrollRef = useRef<HTMLDivElement>(null)
  const topSentinelRef = useRef<HTMLDivElement>(null)
  const prevMessageCountRef = useRef(0)

  // Scroll to bottom when new messages arrive at the end
  useEffect(() => {
    if (messages.length > prevMessageCountRef.current && scrollRef.current) {
      const isNewMessageAtEnd =
        messages.length > 0 &&
        prevMessageCountRef.current > 0 &&
        messages[messages.length - 1]?.id !==
          messages[prevMessageCountRef.current - 1]?.id
      if (isNewMessageAtEnd || prevMessageCountRef.current === 0) {
        scrollRef.current.scrollTo({
          top: scrollRef.current.scrollHeight,
          behavior: prevMessageCountRef.current === 0 ? "instant" : "smooth",
        })
      }
    }
    prevMessageCountRef.current = messages.length
  }, [messages])

  // Infinite scroll up to load older messages
  useEffect(() => {
    const sentinel = topSentinelRef.current
    const container = scrollRef.current
    if (!sentinel || !container || !hasMore) return

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          onLoadMore()
        }
      },
      { root: container, threshold: 0.1 },
    )

    observer.observe(sentinel)
    return () => observer.disconnect()
  }, [hasMore, onLoadMore])

  // Compute which proposal IDs have been superseded by a newer version.
  // A proposal_sent or proposal_modified is superseded when a newer
  // proposal_modified message references the same root via proposal_parent_id.
  // NOTE: This useMemo MUST be before any early return to respect Rules of Hooks.
  const supersededProposalIds = useMemo(() => {
    return computeSupersededIds(messages)
  }, [messages])

  // Compute which proposals have already resolved their
  // "completion_requested" state. Once a proposal_completed,
  // proposal_completion_rejected, milestone_released, or
  // milestone_auto_approved system message lands for a given proposal,
  // the earlier "Complétion demandée" card is stale — the client has
  // either approved, rejected, or the whole proposal has moved on.
  // Hiding it keeps the conversation timeline readable.
  const resolvedCompletionProposalIds = useMemo(() => {
    return computeResolvedCompletionIds(messages)
  }, [messages])

  if (isLoading) {
    return <MessageAreaSkeleton />
  }

  if (messages.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="text-center">
          <MessageSquare
            className="mx-auto h-12 w-12 text-slate-200 dark:text-slate-700"
            strokeWidth={1}
          />
          <p className="mt-3 text-sm text-slate-400 dark:text-slate-500">
            {t("noMessages")}
          </p>
        </div>
      </div>
    )
  }

  return (
    <div ref={scrollRef} className="flex-1 overflow-y-auto px-5 py-4">
      <div className="mx-auto flex max-w-4xl flex-col gap-3">
        {/* Load more sentinel */}
        {hasMore && (
          <div ref={topSentinelRef} className="flex justify-center py-2">
            <button
              onClick={onLoadMore}
              className="text-xs text-slate-400 transition-colors hover:text-slate-600 dark:text-slate-500 dark:hover:text-slate-300"
            >
              {t("loadMore")}
            </button>
          </div>
        )}

        {messages
          .filter((message) => {
            if (
              message.type === "proposal_completion_requested" &&
              isProposalMetadata(message.metadata)
            ) {
              const meta = message.metadata as ProposalMessageMetadata
              return !resolvedCompletionProposalIds.has(meta.proposal_id)
            }
            return true
          })
          .map((message) => (
            <MessageBubble
              key={message.id}
              message={message}
              isOwn={message.sender_id === currentUserId || message.sender_id === "optimistic"}
              currentUserId={currentUserId}
              conversationId={conversationId}
              onEdit={onEdit}
              onDelete={onDelete}
              onReply={onReply}
              onReport={onReport}
              supersededProposalIds={supersededProposalIds}
              onReview={onReview}
            />
          ))}
      </div>
    </div>
  )
}

function computeSupersededIds(messages: Message[]): Set<string> {
  const superseded = new Set<string>()
  const parentIds = new Set<string>()
  for (const msg of messages) {
    if (msg.type === "proposal_modified" && isProposalMetadata(msg.metadata)) {
      const meta = msg.metadata as ProposalMessageMetadata
      if (meta.proposal_parent_id) {
        parentIds.add(meta.proposal_parent_id)
      }
    }
  }
  // Any proposal_sent whose proposal_id is a parent_id of a modified version is superseded
  for (const msg of messages) {
    if ((msg.type === "proposal_sent" || msg.type === "proposal_modified") && isProposalMetadata(msg.metadata)) {
      const meta = msg.metadata as ProposalMessageMetadata
      if (parentIds.has(meta.proposal_id)) {
        superseded.add(meta.proposal_id)
      }
    }
  }
  // Also mark older modified versions as superseded (keep only the latest)
  const versionMap = new Map<string, number>()
  for (const msg of messages) {
    if ((msg.type === "proposal_sent" || msg.type === "proposal_modified") && isProposalMetadata(msg.metadata)) {
      const meta = msg.metadata as ProposalMessageMetadata
      const rootId = meta.proposal_parent_id ?? meta.proposal_id
      const current = versionMap.get(rootId) ?? 0
      if (meta.proposal_version > current) {
        versionMap.set(rootId, meta.proposal_version)
      }
    }
  }
  for (const msg of messages) {
    if ((msg.type === "proposal_sent" || msg.type === "proposal_modified") && isProposalMetadata(msg.metadata)) {
      const meta = msg.metadata as ProposalMessageMetadata
      const rootId = meta.proposal_parent_id ?? meta.proposal_id
      const maxVersion = versionMap.get(rootId) ?? 1
      if (meta.proposal_version < maxVersion) {
        superseded.add(meta.proposal_id)
      }
    }
  }
  return superseded
}

// computeResolvedCompletionIds returns the set of proposal ids whose
// "completion_requested" state has been resolved by a subsequent system
// message in the same conversation. The message types listed below all
// signal that the client has already acted on (or moved past) the
// completion request, making the earlier yellow card stale.
//
// We deliberately include milestone_released / milestone_auto_approved
// so that approving milestone N of a multi-milestone proposal hides the
// old "Complétion demandée" bubble for THAT proposal, even though the
// proposal as a whole will see more completion requests for milestones
// N+1, N+2, etc. Each new request gets its own fresh card after the
// provider re-submits.
function computeResolvedCompletionIds(messages: Message[]): Set<string> {
  const resolved = new Set<string>()
  const resolverTypes = new Set([
    "proposal_completed",
    "proposal_completion_rejected",
    "milestone_released",
    "milestone_auto_approved",
    "proposal_cancelled",
    "proposal_auto_closed",
  ])
  for (const msg of messages) {
    if (resolverTypes.has(msg.type) && isProposalMetadata(msg.metadata)) {
      const meta = msg.metadata as ProposalMessageMetadata
      resolved.add(meta.proposal_id)
    }
  }
  return resolved
}

interface MessageBubbleProps {
  message: Message
  isOwn: boolean
  currentUserId: string
  conversationId: string
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
  onReply: (message: Message) => void
  onReport?: (messageId: string) => void
  supersededProposalIds: Set<string>
  onReview?: (
    proposalId: string,
    proposalTitle: string,
    participants: { clientOrganizationId: string; providerOrganizationId: string },
  ) => void
}

function isProposalMetadata(metadata: unknown): metadata is ProposalMessageMetadata {
  return metadata !== null && typeof metadata === "object" && "proposal_id" in (metadata as Record<string, unknown>)
}

function isFileMetadata(metadata: unknown): metadata is import("../types").FileMetadata {
  return metadata !== null && typeof metadata === "object" && "filename" in (metadata as Record<string, unknown>)
}

function isVoiceMetadata(metadata: unknown): metadata is VoiceMetadata {
  return metadata !== null && typeof metadata === "object" && "duration" in (metadata as Record<string, unknown>)
}

const PROPOSAL_SYSTEM_TYPES = new Set([
  "proposal_accepted",
  "proposal_declined",
  "proposal_paid",
  "proposal_completed",
  "proposal_completion_rejected",
  "proposal_modified",
  // Phase 12: new milestone-scoped types emitted by the proposal
  // service (mid-project release notifications) and the scheduler
  // worker (auto-approval, auto-close).
  "milestone_released",
  "milestone_auto_approved",
  "proposal_cancelled",
  "proposal_auto_closed",
])

const DISPUTE_REASON_LABELS: Record<string, string> = {
  work_not_conforming: "Travail non conforme",
  non_delivery: "Non-livraison",
  insufficient_quality: "Qualite insuffisante",
  client_ghosting: "Client injoignable",
  scope_creep: "Hors du scope",
  refusal_to_validate: "Refus de valider",
  harassment: "Harcelement",
  other: "Autre",
}

const DISPUTE_SYSTEM_TYPES = new Set([
  "dispute_opened",
  "dispute_counter_proposal",
  "dispute_counter_accepted",
  "dispute_counter_rejected",
  "dispute_escalated",
  "dispute_resolved",
  "dispute_cancelled",
  "dispute_auto_resolved",
  "dispute_cancellation_requested",
  "dispute_cancellation_refused",
])

function MessageBubble({
  message,
  isOwn,
  currentUserId,
  conversationId,
  onEdit,
  onDelete,
  onReply,
  onReport,
  supersededProposalIds,
  onReview,
}: MessageBubbleProps) {
  const t = useTranslations("messaging")
  const tp = useTranslations("proposal")

  // Proposal sent or modified — render ProposalCard
  if (
    (message.type === "proposal_sent" || message.type === "proposal_modified") &&
    isProposalMetadata(message.metadata)
  ) {
    const meta = message.metadata as ProposalMessageMetadata
    const isSuperseded = supersededProposalIds.has(meta.proposal_id)

    return (
      <div className={cn("flex flex-col gap-1", isOwn ? "items-end" : "items-start")}>
        {isSuperseded && (
          <span className="text-[10px] font-medium text-slate-400 dark:text-slate-500 px-2">
            {tp("supersededByVersion", { version: meta.proposal_version + 1 })}
          </span>
        )}
        <div className={cn(isSuperseded && "opacity-40 pointer-events-none")}>
          <ProposalCard
            metadata={message.metadata}
            isOwn={isOwn}
            currentUserId={currentUserId}
            conversationId={conversationId}
          />
        </div>
      </div>
    )
  }

  // System messages for proposal state changes
  if (PROPOSAL_SYSTEM_TYPES.has(message.type) && isProposalMetadata(message.metadata)) {
    return (
      <ProposalSystemMessage
        type={message.type}
        metadata={message.metadata}
      />
    )
  }

  // Payment requested — special system message with action
  if (message.type === "proposal_payment_requested" && isProposalMetadata(message.metadata)) {
    return (
      <PaymentRequestedMessage
        metadata={message.metadata}
        currentUserId={currentUserId}
      />
    )
  }

  // Completion requested — special system message with actions for client
  if (message.type === "proposal_completion_requested" && isProposalMetadata(message.metadata)) {
    return (
      <CompletionRequestedMessage
        metadata={message.metadata}
        currentUserId={currentUserId}
      />
    )
  }

  // Evaluation request — system message with "Leave a review" button.
  // Double-blind reviews: the backend now dispatches this message to
  // BOTH the client and the provider, so we intentionally do NOT gate
  // on `target_user_id` or role here. The modal derives the correct
  // review side from the viewer's org vs the proposal participants.
  if (message.type === "evaluation_request" && isProposalMetadata(message.metadata)) {
    return (
      <EvaluationRequestMessage
        metadata={message.metadata}
        onReview={onReview}
      />
    )
  }

  // Call system messages
  if (message.type === "call_ended" || message.type === "call_missed") {
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

  // Dispute system messages
  if (DISPUTE_SYSTEM_TYPES.has(message.type)) {
    return (
      <DisputeSystemBubble
        type={message.type}
        metadata={(message.metadata ?? {}) as Record<string, unknown>}
        currentUserId={currentUserId}
        conversationId={conversationId}
      />
    )
  }

  // Deleted message
  if (message.deleted_at) {
    return (
      <div
        className={cn(
          "flex",
          isOwn ? "justify-end" : "justify-start",
        )}
      >
        <div className="max-w-[75%] rounded-2xl bg-slate-100/60 px-4 py-2.5 dark:bg-slate-800/40">
          <p className="text-sm italic text-slate-400 dark:text-slate-500">
            {t("messageDeleted")}
          </p>
        </div>
      </div>
    )
  }

  return (
    <TextMessageBubble
      message={message}
      isOwn={isOwn}
      onEdit={onEdit}
      onDelete={onDelete}
      onReply={onReply}
      onReport={onReport}
    />
  )
}

function TextMessageBubble({
  message,
  isOwn,
  onEdit,
  onDelete,
  onReply,
  onReport,
}: {
  message: Message
  isOwn: boolean
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
  onReply: (message: Message) => void
  onReport?: (messageId: string) => void
}) {
  const t = useTranslations("messaging")
  const [isEditing, setIsEditing] = useState(false)
  const [editContent, setEditContent] = useState(message.content)
  const [showMobileMenu, setShowMobileMenu] = useState(false)
  const longPressRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const handleTouchStart = useCallback(() => {
    longPressRef.current = setTimeout(() => {
      setShowMobileMenu(true)
    }, 500)
  }, [])

  const handleTouchEnd = useCallback(() => {
    if (longPressRef.current) {
      clearTimeout(longPressRef.current)
      longPressRef.current = null
    }
  }, [])

  const handleTouchMove = useCallback(() => {
    if (longPressRef.current) {
      clearTimeout(longPressRef.current)
      longPressRef.current = null
    }
  }, [])

  const handleEditSubmit = useCallback(() => {
    const trimmed = editContent.trim()
    if (trimmed && trimmed !== message.content) {
      onEdit(message.id, trimmed)
    }
    setIsEditing(false)
  }, [editContent, message.content, message.id, onEdit])

  const timeStr = new Date(message.created_at).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  })

  return (
    <>
    <div
      className={cn(
        "group flex items-start gap-1",
        isOwn ? "flex-row-reverse" : "flex-row",
      )}
    >
      <div
        className={cn(
          "max-w-[75%] rounded-2xl px-4 py-2.5 select-none",
          isOwn
            ? "bg-rose-500 text-white"
            : "bg-slate-100 text-slate-900 dark:bg-slate-800 dark:text-slate-100",
        )}
        onTouchStart={handleTouchStart}
        onTouchEnd={handleTouchEnd}
        onTouchMove={handleTouchMove}
        onContextMenu={(e) => { e.preventDefault(); setShowMobileMenu(true) }}
      >
        {/* Reply preview block */}
        {message.reply_to && (
          <ReplyPreviewBlock replyTo={message.reply_to} isOwn={isOwn} />
        )}

        {/* File message */}
        {message.type === "file" && isFileMetadata(message.metadata) && (
          <FileMessage metadata={message.metadata} isOwn={isOwn} />
        )}

        {/* Voice message */}
        {message.type === "voice" && isVoiceMetadata(message.metadata) && (
          <VoiceMessage metadata={message.metadata} isOwn={isOwn} />
        )}

        {/* Text content */}
        {message.type === "text" && !isEditing && (
          <p className="text-sm leading-relaxed">{message.content}</p>
        )}

        {/* Edit mode */}
        {isEditing && (
          <EditInput
            value={editContent}
            onChange={setEditContent}
            onSubmit={handleEditSubmit}
            onCancel={() => setIsEditing(false)}
          />
        )}

        {/* Edited label */}
        {message.edited_at && !isEditing && (
          <span
            className={cn(
              "text-[10px] italic",
              isOwn ? "text-rose-200" : "text-slate-400 dark:text-slate-500",
            )}
          >
            {" "}({t("messageEdited")})
          </span>
        )}

        {/* Time + status */}
        <div
          className={cn(
            "mt-1 flex items-center gap-1",
            isOwn ? "justify-end" : "justify-start",
          )}
        >
          <p
            className={cn(
              "text-[10px]",
              isOwn
                ? "text-rose-200"
                : "text-slate-400 dark:text-slate-500",
            )}
          >
            {timeStr}
          </p>
          {isOwn && <MessageStatusIcon status={message.status} />}
        </div>
      </div>

      {/* Context menu — desktop only (hidden on touch devices) */}
      {!message.id.startsWith("temp-") && !isEditing && (
        <div className="hidden sm:block">
          <MessageContextMenu
            onReply={() => onReply(message)}
            onEdit={isOwn ? () => {
              setEditContent(message.content)
              setIsEditing(true)
            } : undefined}
            onDelete={isOwn ? () => onDelete(message.id) : undefined}
            onReport={!isOwn && onReport ? () => onReport(message.id) : undefined}
          />
        </div>
      )}
    </div>

    {/* Mobile long-press menu overlay */}
    {showMobileMenu && !isEditing && (
      <div
        className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 sm:hidden"
        onClick={() => setShowMobileMenu(false)}
      >
        <div
          className="w-56 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl dark:border-slate-700 dark:bg-slate-800"
          onClick={(e) => e.stopPropagation()}
        >
          <button
            onClick={() => { setShowMobileMenu(false); onReply(message) }}
            className="flex w-full items-center gap-3 px-4 py-3 text-sm text-slate-700 active:bg-slate-50 dark:text-slate-300 dark:active:bg-slate-700"
          >
            <Reply className="h-4 w-4" strokeWidth={1.5} />
            {t("reply")}
          </button>
          {isOwn && (
            <button
              onClick={() => {
                setShowMobileMenu(false)
                setEditContent(message.content)
                setIsEditing(true)
              }}
              className="flex w-full items-center gap-3 px-4 py-3 text-sm text-slate-700 active:bg-slate-50 dark:text-slate-300 dark:active:bg-slate-700"
            >
              <Pencil className="h-4 w-4" strokeWidth={1.5} />
              {t("editMessage")}
            </button>
          )}
          {isOwn && (
            <button
              onClick={() => { setShowMobileMenu(false); onDelete(message.id) }}
              className="flex w-full items-center gap-3 px-4 py-3 text-sm text-red-600 active:bg-red-50 dark:text-red-400 dark:active:bg-red-500/10"
            >
              <Trash2 className="h-4 w-4" strokeWidth={1.5} />
              {t("deleteMessage")}
            </button>
          )}
          {onReport && (
            <>
              <div className="mx-3 border-t border-slate-200 dark:border-slate-700" />
              <button
                onClick={() => { setShowMobileMenu(false); onReport(message.id) }}
                className="flex w-full items-center gap-3 px-4 py-3 text-sm text-red-600 active:bg-red-50 dark:text-red-400 dark:active:bg-red-500/10"
              >
                <Flag className="h-4 w-4" strokeWidth={1.5} />
                {t("report")}
              </button>
            </>
          )}
        </div>
      </div>
    )}
    </>
  )
}

function EditInput({
  value,
  onChange,
  onSubmit,
  onCancel,
}: {
  value: string
  onChange: (val: string) => void
  onSubmit: () => void
  onCancel: () => void
}) {
  const t = useTranslations("messaging")

  return (
    <div className="space-y-2">
      <input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") onSubmit()
          if (e.key === "Escape") onCancel()
        }}
        className="w-full rounded-lg bg-white/20 px-2 py-1 text-sm text-inherit outline-none"
        autoFocus
      />
      <div className="flex gap-1 text-[10px]">
        <button onClick={onSubmit} className="rounded px-2 py-0.5 hover:bg-white/20">
          {t("save")}
        </button>
        <button onClick={onCancel} className="rounded px-2 py-0.5 hover:bg-white/20">
          {t("cancel")}
        </button>
      </div>
    </div>
  )
}

function ReplyPreviewBlock({
  replyTo,
  isOwn,
}: {
  replyTo: ReplyToInfo
  isOwn: boolean
}) {
  const truncated = replyTo.content.length > 80
    ? replyTo.content.slice(0, 80) + "..."
    : replyTo.content

  return (
    <div
      className={cn(
        "mb-1.5 rounded border-l-2 border-rose-400 px-2 py-1",
        isOwn
          ? "bg-white/15"
          : "bg-rose-500/8 dark:bg-rose-400/10",
      )}
    >
      <p
        className={cn(
          "truncate text-xs",
          isOwn
            ? "text-white/80"
            : "text-slate-500 dark:text-slate-400",
        )}
      >
        {truncated || "..."}
      </p>
    </div>
  )
}
