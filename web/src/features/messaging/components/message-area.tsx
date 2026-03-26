"use client"

import { useRef, useEffect } from "react"
import { MessageSquare } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import type { Message } from "../types"

interface MessageAreaProps {
  messages: Message[]
}

export function MessageArea({ messages }: MessageAreaProps) {
  const t = useTranslations("messaging")
  const scrollRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (scrollRef.current) {
      scrollRef.current.scrollTop = scrollRef.current.scrollHeight
    }
  }, [messages])

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
        {messages.map((message) => (
          <MessageBubble key={message.id} message={message} />
        ))}
      </div>
    </div>
  )
}

interface MessageBubbleProps {
  message: Message
}

function MessageBubble({ message }: MessageBubbleProps) {
  return (
    <div
      className={cn(
        "flex",
        message.isOwn ? "justify-end" : "justify-start",
      )}
    >
      <div
        className={cn(
          "max-w-[75%] rounded-2xl px-4 py-2.5",
          message.isOwn
            ? "bg-rose-500 text-white"
            : "bg-gray-100 text-gray-900 dark:bg-gray-800 dark:text-gray-100",
        )}
      >
        <p className="text-sm leading-relaxed">{message.content}</p>
        <p
          className={cn(
            "mt-1 text-right text-[10px]",
            message.isOwn
              ? "text-rose-200"
              : "text-gray-400 dark:text-gray-500",
          )}
        >
          {message.sentAt}
        </p>
      </div>
    </div>
  )
}
