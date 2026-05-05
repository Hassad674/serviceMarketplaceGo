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

import { Input } from "@/shared/components/ui/input"

// TextMessageBubble — Soleil v2 chat bubble (text/file/voice).
//
// Own bubbles → corail bg right-aligned with last-corner squared.
// Other bubbles → ivoire-card bg left-aligned with first-corner squared
// (the standard chat shape). Time labels in Geist Mono mini.

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
            "max-w-[75%] select-none px-4 py-2.5",
            isOwn
              ? "rounded-2xl rounded-br-md bg-primary text-primary-foreground"
              : "rounded-2xl rounded-bl-md bg-card text-foreground border border-border",
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
                isOwn ? "text-primary-foreground/70" : "text-muted-foreground",
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
                "font-mono text-[10px]",
                isOwn ? "text-primary-foreground/80" : "text-muted-foreground",
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
      className="fixed inset-0 z-50 flex items-center justify-center bg-foreground/30 sm:hidden"
      onClick={onClose}
    >
      <div
        className="w-56 overflow-hidden rounded-2xl border border-border bg-card shadow-[0_8px_24px_rgba(42,31,21,0.12)]"
        onClick={(e) => e.stopPropagation()}
      >
        <Button
          variant="ghost"
          size="auto"
          onClick={() => {
            onClose()
            onReply()
          }}
          className="flex w-full items-center gap-3 px-4 py-3 text-sm text-foreground active:bg-primary-soft"
        >
          <Reply className="h-4 w-4" strokeWidth={1.6} />
          {t("reply")}
        </Button>
        {isOwn && (
          <Button
            variant="ghost"
            size="auto"
            onClick={() => {
              onClose()
              onEdit()
            }}
            className="flex w-full items-center gap-3 px-4 py-3 text-sm text-foreground active:bg-primary-soft"
          >
            <Pencil className="h-4 w-4" strokeWidth={1.6} />
            {t("editMessage")}
          </Button>
        )}
        {isOwn && (
          <Button
            variant="ghost"
            size="auto"
            onClick={() => {
              onClose()
              onDelete()
            }}
            className="flex w-full items-center gap-3 px-4 py-3 text-sm text-destructive active:bg-primary-soft"
          >
            <Trash2 className="h-4 w-4" strokeWidth={1.6} />
            {t("deleteMessage")}
          </Button>
        )}
        {onReport && (
          <>
            <div className="mx-3 border-t border-border" />
            <Button
              variant="ghost"
              size="auto"
              onClick={() => {
                onClose()
                onReport()
              }}
              className="flex w-full items-center gap-3 px-4 py-3 text-sm text-destructive active:bg-primary-soft"
            >
              <Flag className="h-4 w-4" strokeWidth={1.6} />
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
      <Input
        type="text"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Enter") onSubmit()
          if (e.key === "Escape") onCancel()
        }}
        className="w-full rounded-lg bg-card/30 px-2 py-1 text-sm text-inherit outline-none"
        autoFocus
      />
      <div className="flex gap-1 text-[10px]">
        <Button
          variant="ghost"
          size="auto"
          onClick={onSubmit}
          className="rounded px-2 py-0.5 hover:bg-card/30"
        >
          {t("save")}
        </Button>
        <Button
          variant="ghost"
          size="auto"
          onClick={onCancel}
          className="rounded px-2 py-0.5 hover:bg-card/30"
        >
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
    replyTo.content.length > 80
      ? replyTo.content.slice(0, 80) + "..."
      : replyTo.content

  return (
    <div
      className={cn(
        "mb-1.5 rounded border-l-2 px-2 py-1",
        isOwn
          ? "border-primary-foreground/40 bg-primary-foreground/15"
          : "border-primary bg-primary-soft/40",
      )}
    >
      <p
        className={cn(
          "truncate text-xs",
          isOwn ? "text-primary-foreground/80" : "text-muted-foreground",
        )}
      >
        {truncated || "..."}
      </p>
    </div>
  )
}
