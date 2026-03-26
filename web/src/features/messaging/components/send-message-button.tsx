"use client"

import { useState } from "react"
import { MessageSquare, Send, X } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { useStartConversation } from "../hooks/use-start-conversation"

interface SendMessageButtonProps {
  targetUserId: string
}

export function SendMessageButton({ targetUserId }: SendMessageButtonProps) {
  const t = useTranslations("messaging")
  const [isOpen, setIsOpen] = useState(false)
  const [message, setMessage] = useState("")
  const startConversation = useStartConversation()

  function handleSend() {
    const trimmed = message.trim()
    if (!trimmed) return
    startConversation.mutate(
      { otherUserId: targetUserId, content: trimmed },
      { onSuccess: () => setIsOpen(false) },
    )
  }

  if (!isOpen) {
    return (
      <button
        onClick={() => setIsOpen(true)}
        className={cn(
          "inline-flex items-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium",
          "bg-rose-500 text-white shadow-sm",
          "transition-all duration-200 hover:bg-rose-600 hover:shadow-md active:scale-[0.98]",
        )}
      >
        <MessageSquare className="h-4 w-4" strokeWidth={1.5} />
        {t("startConversation")}
      </button>
    )
  }

  return (
    <div
      className={cn(
        "rounded-xl border border-gray-200 bg-white p-4 shadow-md",
        "dark:border-gray-700 dark:bg-gray-800",
      )}
    >
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-white">
          {t("startConversation")}
        </h3>
        <button
          onClick={() => setIsOpen(false)}
          className="rounded-md p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
          aria-label={t("close")}
        >
          <X className="h-4 w-4" strokeWidth={1.5} />
        </button>
      </div>
      <textarea
        value={message}
        onChange={(e) => setMessage(e.target.value)}
        placeholder={t("writeMessage")}
        rows={3}
        className={cn(
          "w-full resize-none rounded-lg bg-gray-100/80 px-3 py-2 text-sm text-gray-900",
          "placeholder:text-gray-400 transition-all duration-200",
          "focus:bg-white focus:shadow-sm focus:ring-2 focus:ring-rose-500/20 focus:outline-none",
          "dark:bg-gray-700/80 dark:text-gray-100 dark:placeholder:text-gray-500",
          "dark:focus:bg-gray-700 dark:focus:ring-rose-400/20",
        )}
      />
      <div className="mt-3 flex justify-end">
        <button
          onClick={handleSend}
          disabled={!message.trim() || startConversation.isPending}
          className={cn(
            "inline-flex items-center gap-2 rounded-lg px-4 py-2 text-sm font-medium",
            "transition-all duration-200",
            message.trim() && !startConversation.isPending
              ? "bg-rose-500 text-white shadow-sm hover:bg-rose-600 hover:shadow-md active:scale-[0.98]"
              : "bg-gray-100 text-gray-400 dark:bg-gray-700 dark:text-gray-500",
          )}
        >
          {startConversation.isPending ? (
            <span className="h-4 w-4 animate-spin rounded-full border-2 border-white/30 border-t-white" />
          ) : (
            <Send className="h-4 w-4" strokeWidth={1.5} />
          )}
          {t("sendMessage")}
        </button>
      </div>
    </div>
  )
}
