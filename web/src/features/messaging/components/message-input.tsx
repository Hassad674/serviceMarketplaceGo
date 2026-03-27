"use client"

import { useState, useRef, useCallback, useEffect } from "react"
import { Paperclip, Send, Loader2, FileText, X, Mic, Square } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { getPresignedURL } from "../api/messaging-api"
import { FileUploadModal } from "./file-upload-modal"
import { useVoiceRecorder } from "../hooks/use-voice-recorder"

const TYPING_INTERVAL_MS = 2_000

/** Map an audio MIME type to a file extension the backend allowlist accepts. */
function voiceExtFromMime(mime: string): string {
  if (mime.includes("webm")) return "webm"
  if (mime.includes("mp4")) return "mp4"
  if (mime.includes("ogg")) return "ogg"
  if (mime.includes("wav")) return "wav"
  return "webm" // safe default
}

function formatRecordingDuration(seconds: number): string {
  const m = Math.floor(seconds / 60).toString().padStart(2, "0")
  const s = (seconds % 60).toString().padStart(2, "0")
  return `${m}:${s}`
}

interface ReplyTarget {
  id: string
  senderName: string
  content: string
}

interface MessageInputProps {
  conversationId: string
  otherUserId: string
  onSend: (content: string, replyToId?: string) => void
  onSendFile: (content: string, metadata: { url: string; filename: string; size: number; mime_type: string }) => void
  onSendVoice?: (content: string, metadata: { url: string; duration: number; size: number; mime_type: string }) => void
  onTyping: () => void
  isSending: boolean
  replyTo?: ReplyTarget | null
  onCancelReply?: () => void
}

export function MessageInput({
  conversationId,
  otherUserId,
  onSend,
  onSendFile,
  onSendVoice,
  onTyping,
  isSending,
  replyTo,
  onCancelReply,
}: MessageInputProps) {
  const t = useTranslations("messaging")
  const router = useRouter()
  const [value, setValue] = useState("")
  const [isUploading, setIsUploading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const onTypingRef = useRef(onTyping)
  const hasContent = value.trim().length > 0

  const voice = useVoiceRecorder()
  const isRecording = voice.state === "recording"
  const isVoiceUploading = voice.state === "uploading"

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
    onSend(trimmed, replyTo?.id)
    setValue("")
    onCancelReply?.()
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

  function handleProposal() {
    router.push(`/projects/new?to=${otherUserId}&conversation=${conversationId}`)
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

  const handleVoiceToggle = useCallback(async () => {
    if (isRecording) {
      // Capture duration BEFORE stopping (stopRecording clears the timer)
      const capturedDuration = voice.duration
      const blob = await voice.stopRecording()
      if (!blob || !onSendVoice) return
      voice.setUploading(true)
      try {
        const ext = voiceExtFromMime(blob.type)
        const filename = `voice-${Date.now()}.${ext}`
        const { upload_url, public_url } = await getPresignedURL(filename, blob.type)
        await fetch(upload_url, { method: "PUT", body: blob, headers: { "Content-Type": blob.type } })
        onSendVoice(t("voiceMessage"), {
          url: public_url,
          duration: capturedDuration,
          size: blob.size,
          mime_type: blob.type,
        })
      } catch {
        setUploadError(t("uploadFailed"))
      } finally {
        voice.setUploading(false)
      }
    } else {
      try {
        await voice.startRecording()
      } catch {
        setUploadError(t("microphonePermission"))
      }
    }
  }, [isRecording, voice, onSendVoice, t])

  const isDisabled = isSending || isUploading || isVoiceUploading

  return (
    <>
      {uploadError && (
        <div className="border-t border-gray-100 bg-red-50 px-4 py-2 dark:border-gray-800 dark:bg-red-900/20" role="alert">
          <p className="text-xs text-red-600 dark:text-red-400">{uploadError}</p>
        </div>
      )}

      {/* Reply preview bar */}
      {replyTo && (
        <div className="flex items-center gap-2 border-t border-gray-100 bg-gray-50 px-4 py-2 dark:border-gray-800 dark:bg-gray-800/50">
          <div className="min-w-0 flex-1 border-l-2 border-rose-500 pl-2">
            <p className="text-xs font-semibold text-rose-500">{t("replyingTo", { name: replyTo.senderName })}</p>
            <p className="truncate text-xs text-gray-500 dark:text-gray-400">
              {replyTo.content.length > 50 ? replyTo.content.slice(0, 50) + "..." : replyTo.content}
            </p>
          </div>
          <button
            type="button"
            onClick={onCancelReply}
            className="shrink-0 rounded-md p-1 text-gray-400 hover:bg-gray-200 hover:text-gray-600 dark:hover:bg-gray-700 dark:hover:text-gray-300"
            aria-label={t("cancelReply")}
          >
            <X className="h-4 w-4" strokeWidth={1.5} />
          </button>
        </div>
      )}

      {/* Voice recording bar */}
      {isRecording && (
        <div className="flex items-center gap-3 border-t border-gray-100 bg-red-50 px-4 py-3 dark:border-gray-800 dark:bg-red-900/20">
          <span className="h-2.5 w-2.5 shrink-0 animate-pulse rounded-full bg-red-500" />
          <span className="text-sm font-medium text-red-600 dark:text-red-400">
            {t("recording")}
          </span>
          <span className="font-mono text-sm text-red-500 dark:text-red-400">
            {formatRecordingDuration(voice.duration)}
          </span>
          <div className="flex-1" />
          <button
            type="button"
            onClick={voice.cancelRecording}
            className="rounded-lg px-3 py-1.5 text-xs font-medium text-gray-500 transition-colors hover:bg-gray-200 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-gray-700 dark:hover:text-gray-200"
            aria-label={t("cancelRecording")}
          >
            {t("cancelRecording")}
          </button>
          <button
            type="button"
            onClick={handleVoiceToggle}
            className={cn(
              "flex h-9 w-9 items-center justify-center rounded-full",
              "bg-red-500 text-white transition-all duration-200",
              "hover:bg-red-600 active:scale-[0.95]",
            )}
            aria-label={t("sendMessage")}
          >
            {isVoiceUploading ? (
              <Loader2 className="h-4 w-4 animate-spin" strokeWidth={1.5} />
            ) : (
              <Square className="h-4 w-4" strokeWidth={2} fill="currentColor" />
            )}
          </button>
        </div>
      )}

      {/* Normal input bar */}
      {!isRecording && (
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

          {/* Proposal button */}
          <button
            type="button"
            onClick={handleProposal}
            disabled={isDisabled}
            className={cn(
              "shrink-0 rounded-lg p-2 text-gray-400 transition-colors",
              "hover:bg-rose-50 hover:text-rose-600",
              "dark:hover:bg-rose-500/10 dark:hover:text-rose-400",
              isDisabled && "pointer-events-none opacity-50",
            )}
            aria-label={t("propose")}
          >
            <FileText className="h-[18px] w-[18px]" strokeWidth={1.5} />
          </button>

          {/* Voice recorder button */}
          {onSendVoice && (
            <button
              type="button"
              onClick={handleVoiceToggle}
              disabled={isDisabled}
              className={cn(
                "shrink-0 rounded-lg p-2 text-gray-400 transition-colors",
                "hover:bg-purple-50 hover:text-purple-600",
                "dark:hover:bg-purple-500/10 dark:hover:text-purple-400",
                isDisabled && "pointer-events-none opacity-50",
              )}
              aria-label={t("voiceMessage")}
            >
              <Mic className="h-[18px] w-[18px]" strokeWidth={1.5} />
            </button>
          )}

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
      )}

      <FileUploadModal
        open={modalOpen}
        onClose={() => setModalOpen(false)}
        onUploadFiles={handleUploadFiles}
        uploading={isUploading}
      />
    </>
  )
}
