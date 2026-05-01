"use client"

import { useCallback, useRef, useState } from "react"
import { Flag, Pencil, Reply, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { Message, ReplyToInfo } from "../types"
import { MessageStatusIcon } from "./message-status-icon"
import { FileMessage } from "./file-message"
import { VoiceMessage } from "./voice-message"
import { MessageContextMenu } from "./message-context-menu"
import { isFileMetadata, isVoiceMetadata } from "./message-area-utils"
import { Button } from "@/shared/components/ui/button"

// TextMessageBubble renders the chat-style "text/file/voice" bubble
// with all interactive affordances: in-place edit, reply, delete,
// report, plus a long-press context menu on touch devices. Kept apart
// from message-bubble.tsx so the simple "system message" branches
// stay focused and free of interactive state.
//
// We group the action handlers + viewer state into a single
// `actions` object to keep the public surface ≤ 4 props per CLAUDE.md.

export interface TextBubbleActions {
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
  onReply: (message: Message) => void
  onReport?: (messageId: string) => void
}

export interface TextMessageBubbleProps {
  message: Message
  isOwn: boolean
  actions: TextBubbleActions
}

export function TextMessageBubble({
  message,
  isOwn,
  actions,
}: TextMessageBubbleProps) {
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
      actions.onEdit(message.id, trimmed)
    }
    setIsEditing(false)
  }, [editContent, message.content, message.id, actions])

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
          onContextMenu={(e) => {
            e.preventDefault()
            setShowMobileMenu(true)
          }}
        >
          {message.reply_to && (
            <ReplyPreviewBlock replyTo={message.reply_to} isOwn={isOwn} />
          )}

          {message.type === "file" && isFileMetadata(message.metadata) && (
            <FileMessage metadata={message.metadata} isOwn={isOwn} />
          )}

          {message.type === "voice" && isVoiceMetadata(message.metadata) && (
            <VoiceMessage metadata={message.metadata} isOwn={isOwn} />
          )}

          {message.type === "text" && !isEditing && (
            <p className="text-sm leading-relaxed">{message.content}</p>
          )}

          {isEditing && (
            <EditInput
              value={editContent}
              onChange={setEditContent}
              onSubmit={handleEditSubmit}
              onCancel={() => setIsEditing(false)}
            />
          )}

          {message.edited_at && !isEditing && (
            <span
              className={cn(
                "text-[10px] italic",
                isOwn ? "text-rose-200" : "text-slate-400 dark:text-slate-500",
              )}
            >
              {" "}
              ({t("messageEdited")})
            </span>
          )}

          <div
            className={cn(
              "mt-1 flex items-center gap-1",
              isOwn ? "justify-end" : "justify-start",
            )}
          >
            <p
              className={cn(
                "text-[10px]",
                isOwn ? "text-rose-200" : "text-slate-400 dark:text-slate-500",
              )}
            >
              {timeStr}
            </p>
            {isOwn && <MessageStatusIcon status={message.status} />}
          </div>
        </div>

        {/* Desktop context menu — hidden on touch devices */}
        {!message.id.startsWith("temp-") && !isEditing && (
          <div className="hidden sm:block">
            <MessageContextMenu
              onReply={() => actions.onReply(message)}
              onEdit={
                isOwn
                  ? () => {
                      setEditContent(message.content)
                      setIsEditing(true)
                    }
                  : undefined
              }
              onDelete={
                isOwn ? () => actions.onDelete(message.id) : undefined
              }
              onReport={
                !isOwn && actions.onReport
                  ? () => actions.onReport!(message.id)
                  : undefined
              }
            />
          </div>
        )}
      </div>

      {/* Mobile long-press menu overlay */}
      {showMobileMenu && !isEditing && (
        <MobileMenuOverlay
          isOwn={isOwn}
          onClose={() => setShowMobileMenu(false)}
          onReply={() => actions.onReply(message)}
          onEdit={() => {
            setEditContent(message.content)
            setIsEditing(true)
          }}
          onDelete={() => actions.onDelete(message.id)}
          onReport={
            actions.onReport ? () => actions.onReport!(message.id) : undefined
          }
        />
      )}
    </>
  )
}

interface MobileMenuOverlayProps {
  isOwn: boolean
  onClose: () => void
  onReply: () => void
  onEdit: () => void
  onDelete: () => void
  onReport?: () => void
}

function MobileMenuOverlay({
  isOwn,
  onClose,
  onReply,
  onEdit,
  onDelete,
  onReport,
}: MobileMenuOverlayProps) {
  const t = useTranslations("messaging")

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 sm:hidden"
      onClick={onClose}
    >
      <div
        className="w-56 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-xl dark:border-slate-700 dark:bg-slate-800"
        onClick={(e) => e.stopPropagation()}
      >
        <Button variant="ghost" size="auto"
          onClick={() => {
            onClose()
            onReply()
          }}
          className="flex w-full items-center gap-3 px-4 py-3 text-sm text-slate-700 active:bg-slate-50 dark:text-slate-300 dark:active:bg-slate-700"
        >
          <Reply className="h-4 w-4" strokeWidth={1.5} />
          {t("reply")}
        </Button>
        {isOwn && (
          <Button variant="ghost" size="auto"
            onClick={() => {
              onClose()
              onEdit()
            }}
            className="flex w-full items-center gap-3 px-4 py-3 text-sm text-slate-700 active:bg-slate-50 dark:text-slate-300 dark:active:bg-slate-700"
          >
            <Pencil className="h-4 w-4" strokeWidth={1.5} />
            {t("editMessage")}
          </Button>
        )}
        {isOwn && (
          <Button variant="ghost" size="auto"
            onClick={() => {
              onClose()
              onDelete()
            }}
            className="flex w-full items-center gap-3 px-4 py-3 text-sm text-red-600 active:bg-red-50 dark:text-red-400 dark:active:bg-red-500/10"
          >
            <Trash2 className="h-4 w-4" strokeWidth={1.5} />
            {t("deleteMessage")}
          </Button>
        )}
        {onReport && (
          <>
            <div className="mx-3 border-t border-slate-200 dark:border-slate-700" />
            <Button variant="ghost" size="auto"
              onClick={() => {
                onClose()
                onReport()
              }}
              className="flex w-full items-center gap-3 px-4 py-3 text-sm text-red-600 active:bg-red-50 dark:text-red-400 dark:active:bg-red-500/10"
            >
              <Flag className="h-4 w-4" strokeWidth={1.5} />
              {t("report")}
            </Button>
          </>
        )}
      </div>
    </div>
  )
}

interface EditInputProps {
  value: string
  onChange: (val: string) => void
  onSubmit: () => void
  onCancel: () => void
}

function EditInput({ value, onChange, onSubmit, onCancel }: EditInputProps) {
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
        <Button variant="ghost" size="auto" onClick={onSubmit} className="rounded px-2 py-0.5 hover:bg-white/20">
          {t("save")}
        </Button>
        <Button variant="ghost" size="auto" onClick={onCancel} className="rounded px-2 py-0.5 hover:bg-white/20">
          {t("cancel")}
        </Button>
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
  const truncated =
    replyTo.content.length > 80 ? replyTo.content.slice(0, 80) + "..." : replyTo.content

  return (
    <div
      className={cn(
        "mb-1.5 rounded border-l-2 border-rose-400 px-2 py-1",
        isOwn ? "bg-white/15" : "bg-rose-500/8 dark:bg-rose-400/10",
      )}
    >
      <p
        className={cn(
          "truncate text-xs",
          isOwn ? "text-white/80" : "text-slate-500 dark:text-slate-400",
        )}
      >
        {truncated || "..."}
      </p>
    </div>
  )
}
