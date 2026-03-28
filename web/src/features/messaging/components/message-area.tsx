"use client"

import { useRef, useEffect, useState, useCallback, useMemo } from "react"
import {
  MessageSquare,
  CheckCircle2,
  XCircle,
  Pencil,
  CreditCard,
  DollarSign,
  RotateCcw,
  Star,
} from "lucide-react"
import { useRouter } from "@i18n/navigation"
import { useTranslations } from "next-intl"
import { cn, formatCurrency } from "@/shared/lib/utils"
import type { Message, ProposalMessageMetadata, ReplyToInfo, VoiceMetadata } from "../types"
import { MessageStatusIcon } from "./message-status-icon"
import { FileMessage } from "./file-message"
import { VoiceMessage } from "./voice-message"
import { MessageContextMenu } from "./message-context-menu"
import { ProposalCard } from "./proposal-card"
import { MessageAreaSkeleton } from "./message-area-skeleton"

interface MessageAreaProps {
  messages: Message[]
  currentUserId: string
  isLoading: boolean
  hasMore: boolean
  onLoadMore: () => void
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
  onReply: (message: Message) => void
  conversationId: string
  onReview?: (proposalId: string, proposalTitle: string) => void
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
  }, [messages])

  if (isLoading) {
    return <MessageAreaSkeleton />
  }

  if (messages.length === 0) {
    return (
      <div className="flex flex-1 items-center justify-center">
        <div className="text-center">
          <MessageSquare
            className="mx-auto h-12 w-12 text-gray-200 dark:text-gray-700"
            strokeWidth={1}
          />
          <p className="mt-3 text-sm text-gray-400 dark:text-gray-500">
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
              className="text-xs text-gray-400 transition-colors hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
            >
              {t("loadMore")}
            </button>
          </div>
        )}

        {messages.map((message) => (
          <MessageBubble
            key={message.id}
            message={message}
            isOwn={message.sender_id === currentUserId || message.sender_id === "optimistic"}
            currentUserId={currentUserId}
            conversationId={conversationId}
            onEdit={onEdit}
            onDelete={onDelete}
            onReply={onReply}
            supersededProposalIds={supersededProposalIds}
            onReview={onReview}
          />
        ))}
      </div>
    </div>
  )
}

interface MessageBubbleProps {
  message: Message
  isOwn: boolean
  currentUserId: string
  conversationId: string
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
  onReply: (message: Message) => void
  supersededProposalIds: Set<string>
  onReview?: (proposalId: string, proposalTitle: string) => void
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
])

function MessageBubble({
  message,
  isOwn,
  currentUserId,
  conversationId,
  onEdit,
  onDelete,
  onReply,
  supersededProposalIds,
  onReview,
}: MessageBubbleProps) {
  const t = useTranslations("messaging")
  const tp = useTranslations("proposal")
  const router = useRouter()
  const [isEditing, setIsEditing] = useState(false)
  const [editContent, setEditContent] = useState(message.content)

  const handleEditSubmit = useCallback(() => {
    const trimmed = editContent.trim()
    if (trimmed && trimmed !== message.content) {
      onEdit(message.id, trimmed)
    }
    setIsEditing(false)
  }, [editContent, message.content, message.id, onEdit])

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
          <span className="text-[10px] font-medium text-gray-400 dark:text-gray-500 px-2">
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
    const payMeta = message.metadata
    return (
      <div className="flex justify-center py-2">
        <div className="flex items-center gap-3 rounded-xl bg-blue-50 px-4 py-2.5 dark:bg-blue-500/10">
          <CreditCard className="h-4 w-4 text-blue-600 dark:text-blue-400" strokeWidth={1.5} />
          <span className="text-sm font-medium text-blue-700 dark:text-blue-300">
            {tp("paymentRequested")}
          </span>
          {payMeta.proposal_client_id === currentUserId && (
            <button
              type="button"
              onClick={() => router.push(`/projects/pay?proposal=${payMeta.proposal_id}`)}
              className={cn(
                "rounded-lg px-3 py-1 text-xs font-semibold text-white",
                "gradient-primary hover:shadow-glow active:scale-[0.98]",
                "transition-all duration-200",
              )}
            >
              {tp("payNow")}
            </button>
          )}
        </div>
      </div>
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
  // Only visible to the client (target_user_id in metadata).
  if (message.type === "evaluation_request" && isProposalMetadata(message.metadata)) {
    const meta = message.metadata as ProposalMessageMetadata
    if (meta.target_user_id && meta.target_user_id !== currentUserId) {
      return null
    }
    return (
      <EvaluationRequestMessage
        metadata={message.metadata}
        onReview={onReview}
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
        <div className="max-w-[75%] rounded-2xl bg-gray-100/60 px-4 py-2.5 dark:bg-gray-800/40">
          <p className="text-sm italic text-gray-400 dark:text-gray-500">
            {t("messageDeleted")}
          </p>
        </div>
      </div>
    )
  }

  const timeStr = new Date(message.created_at).toLocaleTimeString([], {
    hour: "2-digit",
    minute: "2-digit",
  })

  return (
    <div
      className={cn(
        "group flex items-start gap-1",
        isOwn ? "flex-row-reverse" : "flex-row",
      )}
    >
      <div
        className={cn(
          "max-w-[75%] rounded-2xl px-4 py-2.5",
          isOwn
            ? "bg-rose-500 text-white"
            : "bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100",
        )}
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
          <div className="space-y-2">
            <input
              type="text"
              value={editContent}
              onChange={(e) => setEditContent(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleEditSubmit()
                if (e.key === "Escape") setIsEditing(false)
              }}
              className="w-full rounded-lg bg-white/20 px-2 py-1 text-sm text-inherit outline-none"
              autoFocus
            />
            <div className="flex gap-1 text-[10px]">
              <button
                onClick={handleEditSubmit}
                className="rounded px-2 py-0.5 hover:bg-white/20"
              >
                {t("save")}
              </button>
              <button
                onClick={() => setIsEditing(false)}
                className="rounded px-2 py-0.5 hover:bg-white/20"
              >
                {t("cancel")}
              </button>
            </div>
          </div>
        )}

        {/* Edited label */}
        {message.edited_at && !isEditing && (
          <span
            className={cn(
              "text-[10px] italic",
              isOwn ? "text-rose-200" : "text-gray-400 dark:text-gray-500",
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
                : "text-gray-400 dark:text-gray-500",
            )}
          >
            {timeStr}
          </p>
          {isOwn && <MessageStatusIcon status={message.status} />}
        </div>
      </div>

      {/* Context menu — Reply on all messages, Edit/Delete on own */}
      {!message.id.startsWith("temp-") && !isEditing && (
        <MessageContextMenu
          onReply={() => onReply(message)}
          onEdit={isOwn ? () => {
            setEditContent(message.content)
            setIsEditing(true)
          } : undefined}
          onDelete={isOwn ? () => onDelete(message.id) : undefined}
        />
      )}
    </div>
  )
}

function ProposalSystemMessage({
  type,
  metadata,
}: {
  type: string
  metadata: ProposalMessageMetadata
}) {
  const t = useTranslations("proposal")

  const config: Record<string, { icon: React.ElementType; text: string; className: string }> = {
    proposal_accepted: {
      icon: CheckCircle2,
      text: t("proposalAccepted"),
      className: "bg-green-50 text-green-700 dark:bg-green-500/10 dark:text-green-300",
    },
    proposal_declined: {
      icon: XCircle,
      text: t("proposalDeclined"),
      className: "bg-red-50 text-red-700 dark:bg-red-500/10 dark:text-red-300",
    },
    proposal_paid: {
      icon: DollarSign,
      text: t("proposalPaid"),
      className: "bg-blue-50 text-blue-700 dark:bg-blue-500/10 dark:text-blue-300",
    },
    proposal_completed: {
      icon: CheckCircle2,
      text: t("missionCompleted"),
      className: "bg-emerald-50 text-emerald-700 dark:bg-emerald-500/10 dark:text-emerald-300",
    },
    proposal_completion_rejected: {
      icon: RotateCcw,
      text: t("completionRejected"),
      className: "bg-gray-50 text-gray-700 dark:bg-gray-500/10 dark:text-gray-300",
    },
  }

  const entry = config[type]
  if (!entry) return null

  const Icon = entry.icon

  return (
    <div className="flex justify-center py-2">
      <div className={cn("flex items-center gap-2 rounded-xl px-4 py-2", entry.className)}>
        <Icon className="h-4 w-4" strokeWidth={1.5} />
        <span className="text-sm font-medium">
          {entry.text} — {metadata.proposal_title} ({formatCurrency(metadata.proposal_amount / 100)})
        </span>
      </div>
    </div>
  )
}

function CompletionRequestedMessage({
  metadata,
  currentUserId,
}: {
  metadata: ProposalMessageMetadata
  currentUserId: string
}) {
  const t = useTranslations("proposal")
  const router = useRouter()

  return (
    <div className="flex justify-center py-2">
      <div className="flex flex-col items-center gap-2 rounded-xl bg-amber-50 px-5 py-3 dark:bg-amber-500/10">
        <div className="flex items-center gap-2">
          <CheckCircle2 className="h-4 w-4 text-amber-600 dark:text-amber-400" strokeWidth={1.5} />
          <span className="text-sm font-medium text-amber-700 dark:text-amber-300">
            {t("completionRequested")} — {metadata.proposal_title} ({formatCurrency(metadata.proposal_amount / 100)})
          </span>
        </div>
        {metadata.proposal_client_id === currentUserId && (
          <button
            type="button"
            onClick={() => router.push(`/projects/${metadata.proposal_id}`)}
            className={cn(
              "rounded-lg px-3 py-1 text-xs font-semibold text-white",
              "gradient-primary hover:shadow-glow active:scale-[0.98]",
              "transition-all duration-200",
            )}
          >
            {t("viewDetails")}
          </button>
        )}
      </div>
    </div>
  )
}

function EvaluationRequestMessage({
  metadata,
  onReview,
}: {
  metadata: ProposalMessageMetadata
  onReview?: (proposalId: string, proposalTitle: string) => void
}) {
  const t = useTranslations("review")

  return (
    <div className="flex justify-center py-2">
      <div className="flex flex-col items-center gap-2 rounded-xl bg-emerald-50 px-5 py-3 dark:bg-emerald-500/10">
        <div className="flex items-center gap-2">
          <Star className="h-4 w-4 text-emerald-600 dark:text-emerald-400" strokeWidth={1.5} />
          <span className="text-sm font-medium text-emerald-700 dark:text-emerald-300">
            {t("evaluationRequest")}
          </span>
        </div>
        <button
          type="button"
          onClick={() => onReview?.(metadata.proposal_id, metadata.proposal_title)}
          className={cn(
            "rounded-lg px-3 py-1 text-xs font-semibold text-white",
            "gradient-primary hover:shadow-glow active:scale-[0.98]",
            "transition-all duration-200",
          )}
        >
          {t("leaveReview")}
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
            : "text-gray-500 dark:text-gray-400",
        )}
      >
        {truncated || "..."}
      </p>
    </div>
  )
}

