"use client"

import { useState } from "react"
import { Paperclip, Smile, Mic, Send } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"

interface MessageInputProps {
  onSend?: (content: string) => void
}

export function MessageInput({ onSend }: MessageInputProps) {
  const t = useTranslations("messaging")
  const [value, setValue] = useState("")

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = value.trim()
    if (!trimmed) return
    onSend?.(trimmed)
    setValue("")
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  return (
    <form
      onSubmit={handleSubmit}
      className="flex items-center gap-2 border-t border-gray-100 bg-white px-4 py-3 dark:border-gray-800 dark:bg-gray-900"
    >
      {/* Attachment */}
      <button
        type="button"
        className="shrink-0 rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
        aria-label="Attach file"
      >
        <Paperclip className="h-[18px] w-[18px]" strokeWidth={1.5} />
      </button>

      {/* Emoji */}
      <button
        type="button"
        className="shrink-0 rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
        aria-label="Add emoji"
      >
        <Smile className="h-[18px] w-[18px]" strokeWidth={1.5} />
      </button>

      {/* Input */}
      <input
        type="text"
        value={value}
        onChange={(e) => setValue(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder={t("writeMessage")}
        className={cn(
          "h-10 flex-1 rounded-full bg-gray-100/80 px-4 text-sm text-gray-900",
          "placeholder:text-gray-400 transition-all duration-200",
          "focus:bg-white focus:shadow-sm focus:ring-2 focus:ring-rose-500/20 focus:outline-none",
          "dark:bg-gray-800/80 dark:text-gray-100 dark:placeholder:text-gray-500",
          "dark:focus:bg-gray-800 dark:focus:ring-rose-400/20",
        )}
      />

      {/* Voice */}
      <button
        type="button"
        className="shrink-0 rounded-lg p-2 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
        aria-label="Voice note"
      >
        <Mic className="h-[18px] w-[18px]" strokeWidth={1.5} />
      </button>

      {/* Send */}
      <button
        type="submit"
        disabled={!value.trim()}
        className={cn(
          "shrink-0 rounded-full p-2.5 transition-all duration-200",
          value.trim()
            ? "bg-rose-500 text-white shadow-sm hover:bg-rose-600 hover:shadow-md active:scale-[0.96]"
            : "bg-gray-100 text-gray-300 dark:bg-gray-800 dark:text-gray-600",
        )}
        aria-label="Send message"
      >
        <Send className="h-[18px] w-[18px]" strokeWidth={1.5} />
      </button>
    </form>
  )
}
