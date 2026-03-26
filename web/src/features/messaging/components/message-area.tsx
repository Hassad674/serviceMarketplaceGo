"use client"

import { useRef, useEffect, useState, useCallback } from "react"
import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { Message, ProposalMessageMetadata } from "../types"
import { MessageStatusIcon } from "./message-status-icon"
import { FileMessage } from "./file-message"
import { MessageContextMenu } from "./message-context-menu"
import { ProposalCard } from "./proposal-card"

interface MessageAreaProps {
  messages: Message[]
  currentUserId: string
  isLoading: boolean
  hasMore: boolean
  onLoadMore: () => void
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
}

export function MessageArea({
  messages,
  currentUserId,
  isLoading,
  hasMore,
  onLoadMore,
  onEdit,
  onDelete,
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
            onEdit={onEdit}
            onDelete={onDelete}
          />
        ))}
      </div>
    </div>
  )
}

interface MessageBubbleProps {
  message: Message
  isOwn: boolean
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
}

function isProposalMetadata(metadata: unknown): metadata is ProposalMessageMetadata {
  return metadata !== null && typeof metadata === "object" && "proposal_id" in (metadata as Record<string, unknown>)
}

function isFileMetadata(metadata: unknown): metadata is import("../types").FileMetadata {
  return metadata !== null && typeof metadata === "object" && "filename" in (metadata as Record<string, unknown>)
}

function MessageBubble({ message, isOwn, onEdit, onDelete }: MessageBubbleProps) {
  const t = useTranslations("messaging")
  const [isEditing, setIsEditing] = useState(false)
  const [editContent, setEditContent] = useState(message.content)

  const handleEditSubmit = useCallback(() => {
    const trimmed = editContent.trim()
    if (trimmed && trimmed !== message.content) {
      onEdit(message.id, trimmed)
    }
    setIsEditing(false)
  }, [editContent, message.content, message.id, onEdit])

  // Proposal message — render ProposalCard instead of a text bubble
  if (message.type === "proposal_sent" && isProposalMetadata(message.metadata)) {
    return (
      <div className={cn("flex", isOwn ? "justify-end" : "justify-start")}>
        <ProposalCard
          metadata={message.metadata}
          isOwn={isOwn}
          onAccept={() => console.log("Accept proposal:", message.metadata)}
          onDecline={() => console.log("Decline proposal:", message.metadata)}
        />
      </div>
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
        {/* File message */}
        {message.type === "file" && isFileMetadata(message.metadata) && (
          <FileMessage metadata={message.metadata} isOwn={isOwn} />
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

      {/* Context menu for own messages */}
      {isOwn && !message.id.startsWith("temp-") && !isEditing && (
        <MessageContextMenu
          onEdit={() => {
            setEditContent(message.content)
            setIsEditing(true)
          }}
          onDelete={() => onDelete(message.id)}
        />
      )}
    </div>
  )
}

function MessageAreaSkeleton() {
  return (
    <div className="flex-1 overflow-hidden px-5 py-4">
      <div className="mx-auto flex max-w-4xl flex-col gap-3">
        {[1, 2, 3, 4, 5].map((i) => (
          <div
            key={i}
            className={cn(
              "flex",
              i % 2 === 0 ? "justify-end" : "justify-start",
            )}
          >
            <div
              className={cn(
                "animate-pulse rounded-2xl px-4 py-2.5",
                i % 2 === 0 ? "bg-rose-200 dark:bg-rose-500/20" : "bg-gray-200 dark:bg-gray-700",
              )}
              style={{ width: `${40 + (i * 10) % 35}%`, height: "48px" }}
            />
          </div>
        ))}
      </div>
    </div>
  )
}
