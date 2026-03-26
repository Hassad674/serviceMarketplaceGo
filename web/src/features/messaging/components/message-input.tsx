"use client"

import { useState, useRef, useCallback } from "react"
import { Paperclip, Send, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { getPresignedURL } from "../api/messaging-api"

const TYPING_THROTTLE_MS = 2_000
const MAX_FILE_SIZE = 10 * 1024 * 1024 // 10MB

interface MessageInputProps {
  onSend: (content: string) => void
  onSendFile: (content: string, metadata: { file_key: string; filename: string; size: number; mime_type: string }) => void
  onTyping: () => void
  isSending: boolean
}

export function MessageInput({ onSend, onSendFile, onTyping, isSending }: MessageInputProps) {
  const t = useTranslations("messaging")
  const [value, setValue] = useState("")
  const [isUploading, setIsUploading] = useState(false)
  const lastTypingRef = useRef(0)
  const fileInputRef = useRef<HTMLInputElement>(null)

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = value.trim()
    if (!trimmed || isSending) return
    onSend(trimmed)
    setValue("")
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault()
      handleSubmit(e)
    }
  }

  function handleInputChange(e: React.ChangeEvent<HTMLInputElement>) {
    setValue(e.target.value)

    // Throttled typing indicator
    const now = Date.now()
    if (now - lastTypingRef.current > TYPING_THROTTLE_MS) {
      lastTypingRef.current = now
      onTyping()
    }
  }

  const handleFileSelect = useCallback(
    async (e: React.ChangeEvent<HTMLInputElement>) => {
      const file = e.target.files?.[0]
      if (!file) return

      if (file.size > MAX_FILE_SIZE) {
        // File too large — silently ignore (could show toast in the future)
        return
      }

      setIsUploading(true)
      try {
        // Get presigned URL
        const { upload_url, file_key } = await getPresignedURL(
          file.name,
          file.type,
        )

        // Upload to storage
        await fetch(upload_url, {
          method: "PUT",
          body: file,
          headers: { "Content-Type": file.type },
        })

        // Send file message
        onSendFile(file.name, {
          file_key,
          filename: file.name,
          size: file.size,
          mime_type: file.type,
        })
      } catch {
        // Upload failed — silent for now
      } finally {
        setIsUploading(false)
        // Reset file input
        if (fileInputRef.current) {
          fileInputRef.current.value = ""
        }
      }
    },
    [onSendFile],
  )

  const isDisabled = isSending || isUploading

  return (
    <form
      onSubmit={handleSubmit}
      className="flex items-center gap-2 border-t border-gray-100 bg-white px-4 py-3 dark:border-gray-800 dark:bg-gray-900"
    >
      {/* File attachment */}
      <input
        ref={fileInputRef}
        type="file"
        onChange={handleFileSelect}
        className="hidden"
        aria-label={t("fileUpload")}
      />
      <button
        type="button"
        onClick={() => fileInputRef.current?.click()}
        disabled={isDisabled}
        className={cn(
          "shrink-0 rounded-lg p-2 text-gray-400 transition-colors",
          "hover:bg-gray-100 hover:text-gray-600",
          "dark:hover:bg-gray-800 dark:hover:text-gray-300",
          isDisabled && "pointer-events-none opacity-50",
        )}
        aria-label={t("fileUpload")}
      >
        {isUploading ? (
          <Loader2 className="h-[18px] w-[18px] animate-spin" strokeWidth={1.5} />
        ) : (
          <Paperclip className="h-[18px] w-[18px]" strokeWidth={1.5} />
        )}
      </button>

      {/* Input */}
      <input
        type="text"
        value={value}
        onChange={handleInputChange}
        onKeyDown={handleKeyDown}
        placeholder={t("writeMessage")}
        disabled={isDisabled}
        className={cn(
          "h-10 flex-1 rounded-full bg-gray-100/80 px-4 text-sm text-gray-900",
          "placeholder:text-gray-400 transition-all duration-200",
          "focus:bg-white focus:shadow-sm focus:ring-2 focus:ring-rose-500/20 focus:outline-none",
          "dark:bg-gray-800/80 dark:text-gray-100 dark:placeholder:text-gray-500",
          "dark:focus:bg-gray-800 dark:focus:ring-rose-400/20",
          isDisabled && "opacity-50",
        )}
      />

      {/* Send */}
      <button
        type="submit"
        disabled={!value.trim() || isDisabled}
        className={cn(
          "shrink-0 rounded-full p-2.5 transition-all duration-200",
          value.trim() && !isDisabled
            ? "bg-rose-500 text-white shadow-sm hover:bg-rose-600 hover:shadow-md active:scale-[0.96]"
            : "bg-gray-100 text-gray-300 dark:bg-gray-800 dark:text-gray-600",
        )}
        aria-label={t("sendMessage")}
      >
        {isSending ? (
          <Loader2 className="h-[18px] w-[18px] animate-spin" strokeWidth={1.5} />
        ) : (
          <Send className="h-[18px] w-[18px]" strokeWidth={1.5} />
        )}
      </button>
    </form>
  )
}
