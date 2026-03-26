"use client"

import { useState, useRef, useCallback, useEffect } from "react"
import { Paperclip, Send, Loader2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { cn } from "@/shared/lib/utils"
import { getPresignedURL } from "../api/messaging-api"
import { FileUploadModal } from "./file-upload-modal"

const TYPING_INTERVAL_MS = 2_000

interface MessageInputProps {
  onSend: (content: string) => void
  onSendFile: (content: string, metadata: { url: string; filename: string; size: number; mime_type: string }) => void
  onTyping: () => void
  isSending: boolean
}

export function MessageInput({ onSend, onSendFile, onTyping, isSending }: MessageInputProps) {
  const t = useTranslations("messaging")
  const [value, setValue] = useState("")
  const [isUploading, setIsUploading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const onTypingRef = useRef(onTyping)
  const hasContent = value.trim().length > 0

  // Keep callback ref in sync to avoid re-creating the interval effect
  useEffect(() => {
    onTypingRef.current = onTyping
  }, [onTyping])

  // Send typing events every 2s while the input has content
  useEffect(() => {
    if (!hasContent) return

    // Fire immediately when input goes from empty to non-empty
    onTypingRef.current()

    const interval = setInterval(() => {
      onTypingRef.current()
    }, TYPING_INTERVAL_MS)

    return () => clearInterval(interval)
  }, [hasContent])

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
  }

  const handleUploadFiles = useCallback(
    async (files: File[]) => {
      setIsUploading(true)
      setUploadError(null)
      try {
        for (const file of files) {
          // Get presigned URL
          const { upload_url, public_url } = await getPresignedURL(
            file.name,
            file.type,
          )

          // Upload to storage via presigned URL
          await fetch(upload_url, {
            method: "PUT",
            body: file,
            headers: { "Content-Type": file.type },
          })

          // Send file message
          onSendFile(file.name, {
            url: public_url,
            filename: file.name,
            size: file.size,
            mime_type: file.type,
          })
        }
      } catch {
        setUploadError(t("uploadFailed"))
      } finally {
        setIsUploading(false)
        setModalOpen(false)
      }
    },
    [onSendFile, t],
  )

  const isDisabled = isSending || isUploading

  return (
    <>
      {uploadError && (
        <div className="border-t border-gray-100 bg-red-50 px-4 py-2 dark:border-gray-800 dark:bg-red-900/20" role="alert">
          <p className="text-xs text-red-600 dark:text-red-400">{uploadError}</p>
        </div>
      )}
      <form
        onSubmit={handleSubmit}
        className="flex items-center gap-2 border-t border-gray-100 bg-white px-4 py-3 dark:border-gray-800 dark:bg-gray-900"
      >
        {/* File attachment button */}
        <button
          type="button"
          onClick={() => setModalOpen(true)}
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
          aria-label={t("writeMessage")}
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

      <FileUploadModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onUploadFiles={handleUploadFiles}
        uploading={isUploading}
      />
    </>
  )
}
