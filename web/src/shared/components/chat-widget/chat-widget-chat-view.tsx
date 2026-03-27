"use client"

import { useState, useRef, useEffect, useCallback } from "react"
import { ChevronLeft, X, Send, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { MessageArea } from "@/features/messaging/components/message-area"
import type { Conversation, Message } from "@/features/messaging/types"

const TYPING_INTERVAL_MS = 2_000

interface ChatWidgetChatViewProps {
  conversation: Conversation | null
  messages: Message[]
  currentUserId: string
  isLoading: boolean
  hasMore: boolean
  isSending: boolean
  typingUser: { userId: string } | undefined
  onLoadMore: () => void
  onSend: (content: string) => void
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
  onTyping: () => void
  onBack: () => void
  onClose: () => void
}

export function ChatWidgetChatView({
  conversation,
  messages,
  currentUserId,
  isLoading,
  hasMore,
  isSending,
  typingUser,
  onLoadMore,
  onSend,
  onEdit,
  onDelete,
  onTyping,
  onBack,
  onClose,
}: ChatWidgetChatViewProps) {
  const t = useTranslations("messaging")

  if (!conversation) {
    return (
      <div className="flex h-[500px] items-center justify-center">
        <p className="text-xs text-gray-400 dark:text-gray-500">
          {t("noConversations")}
        </p>
      </div>
    )
  }

  return (
    <div className="flex h-[500px] flex-col">
      {/* Header */}
      <ChatViewHeader
        name={conversation.other_user_name}
        online={conversation.online}
        typingUserName={typingUser ? conversation.other_user_name : undefined}
        onBack={onBack}
        onClose={onClose}
      />

      {/* Message area */}
      <div className="flex min-h-0 flex-1 flex-col">
        <MessageArea
          messages={messages}
          currentUserId={currentUserId}
          isLoading={isLoading}
          hasMore={hasMore}
          onLoadMore={onLoadMore}
          onEdit={onEdit}
          onDelete={onDelete}
        />
      </div>

      {/* Input */}
      <CompactMessageInput
        onSend={onSend}
        onTyping={onTyping}
        isSending={isSending}
      />
    </div>
  )
}

interface ChatViewHeaderProps {
  name: string
  online: boolean
  typingUserName: string | undefined
  onBack: () => void
  onClose: () => void
}

function ChatViewHeader({
  name,
  online,
  typingUserName,
  onBack,
  onClose,
}: ChatViewHeaderProps) {
  const t = useTranslations("messaging")

  return (
    <div className="flex h-14 shrink-0 items-center gap-2 border-b border-gray-100 px-3 dark:border-gray-800">
      <button
        onClick={onBack}
        className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
        aria-label={t("backToList")}
      >
        <ChevronLeft className="h-4 w-4" strokeWidth={1.5} />
      </button>

      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-semibold text-gray-900 dark:text-white">
          {name}
        </p>
        {typingUserName ? (
          <p className="truncate text-[10px] italic text-rose-500 dark:text-rose-400">
            {t("typingShort")}
          </p>
        ) : (
          <p
            className={cn(
              "text-[10px]",
              online
                ? "text-emerald-500"
                : "text-gray-400 dark:text-gray-500",
            )}
          >
            {online ? t("online") : t("offline")}
          </p>
        )}
      </div>

      <button
        onClick={onClose}
        className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
        aria-label={t("close")}
      >
        <X className="h-4 w-4" strokeWidth={1.5} />
      </button>
    </div>
  )
}

interface CompactMessageInputProps {
  onSend: (content: string) => void
  onTyping: () => void
  isSending: boolean
}

function CompactMessageInput({
  onSend,
  onTyping,
  isSending,
}: CompactMessageInputProps) {
  const t = useTranslations("messaging")
  const [value, setValue] = useState("")
  const onTypingRef = useRef(onTyping)
  const hasContent = value.trim().length > 0

  useEffect(() => {
    onTypingRef.current = onTyping
  }, [onTyping])

  // Send typing events periodically while input has content
  useEffect(() => {
    if (!hasContent) return
    onTypingRef.current()
    const interval = setInterval(() => {
      onTypingRef.current()
    }, TYPING_INTERVAL_MS)
    return () => clearInterval(interval)
  }, [hasContent])

  const handleSubmit = useCallback(
    (e: React.FormEvent) => {
      e.preventDefault()
      const trimmed = value.trim()
      if (!trimmed || isSending) return
      onSend(trimmed)
      setValue("")
    },
    [value, isSending, onSend],
  )

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex shrink-0 items-center gap-2 border-t border-gray-100 bg-white px-3 py-2.5 dark:border-gray-800 dark:bg-gray-900"
    >
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={t("writeMessage")}
        aria-label={t("writeMessage")}
        disabled={isSending}
        className={cn(
          "h-9 flex-1 rounded-full bg-gray-100/80 px-3.5 text-xs text-gray-900",
          "placeholder:text-gray-400 transition-all duration-200",
          "focus:bg-white focus:shadow-sm focus:ring-2 focus:ring-rose-500/20 focus:outline-none",
          "dark:bg-gray-800/80 dark:text-gray-100 dark:placeholder:text-gray-500",
          "dark:focus:bg-gray-800 dark:focus:ring-rose-400/20",
          isSending && "opacity-50",
        )}
      />
      <button
        type="submit"
        disabled={!value.trim() || isSending}
        className={cn(
          "shrink-0 rounded-full p-2 transition-all duration-200",
          value.trim() && !isSending
            ? "bg-rose-500 text-white shadow-sm hover:bg-rose-600 active:scale-[0.96]"
            : "bg-gray-100 text-gray-300 dark:bg-gray-800 dark:text-gray-600",
        )}
        aria-label={t("sendMessage")}
      >
        {isSending ? (
          <Loader2 className="h-4 w-4 animate-spin" strokeWidth={1.5} />
        ) : (
          <Send className="h-4 w-4" strokeWidth={1.5} />
        )}
      </button>
    </form>
  )
}
