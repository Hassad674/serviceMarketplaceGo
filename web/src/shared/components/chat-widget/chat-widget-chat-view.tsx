"use client"

import { useState, useRef, useEffect, useCallback } from "react"
import { ChevronLeft, ChevronDown, Paperclip, FileText, Send, Loader2, Mic, Square } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { MessageArea } from "@/features/messaging/components/message-area"
import { FileUploadModal } from "@/features/messaging/components/file-upload-modal"
import { useVoiceRecorder } from "@/features/messaging/hooks/use-voice-recorder"
import { getPresignedURL } from "@/features/messaging/api/messaging-api"
import type { Conversation, Message } from "@/features/messaging/types"

const TYPING_INTERVAL_MS = 2_000

/** Map an audio MIME type to a file extension the backend allowlist accepts. */
function voiceExtFromMime(mime: string): string {
  if (mime.includes("webm")) return "webm"
  if (mime.includes("mp4")) return "mp4"
  if (mime.includes("ogg")) return "ogg"
  if (mime.includes("wav")) return "wav"
  return "webm"
}

interface ChatWidgetChatViewProps {
  conversation: Conversation | null
  conversationId: string | null
  messages: Message[]
  currentUserId: string
  isLoading: boolean
  hasMore: boolean
  isSending: boolean
  typingUser: { userId: string } | undefined
  onLoadMore: () => void
  onSend: (content: string) => void
  onSendFile: (files: File[]) => Promise<void>
  onSendVoice?: (content: string, metadata: { url: string; duration: number; size: number; mime_type: string }) => void
  onEdit: (messageId: string, content: string) => void
  onDelete: (messageId: string) => void
  onTyping: () => void
  onBack: () => void
  onClose: () => void
}

export function ChatWidgetChatView({
  conversation,
  conversationId,
  messages,
  currentUserId,
  isLoading,
  hasMore,
  isSending,
  typingUser,
  onLoadMore,
  onSend,
  onSendFile,
  onSendVoice,
  onEdit,
  onDelete,
  onTyping,
  onBack,
  onClose,
}: ChatWidgetChatViewProps) {
  const t = useTranslations("messaging")

  if (!conversation) {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-xs text-gray-400 dark:text-gray-500">
          {t("noConversations")}
        </p>
      </div>
    )
  }

  return (
    <div className="flex h-full flex-col">
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
          onReply={() => {}}
          conversationId={conversationId ?? ""}
        />
      </div>

      {/* Full-featured input with file + proposal + voice buttons */}
      <WidgetMessageInput
        conversationId={conversationId}
        otherUserId={conversation.other_user_id}
        onSend={onSend}
        onSendFile={onSendFile}
        onSendVoice={onSendVoice}
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
    <div className="flex h-12 shrink-0 items-center gap-2 border-b border-gray-100 px-3 dark:border-gray-800">
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
        <ChevronDown className="h-4 w-4" strokeWidth={1.5} />
      </button>
    </div>
  )
}

function formatRecordingDuration(seconds: number): string {
  const m = Math.floor(seconds / 60).toString().padStart(2, "0")
  const s = (seconds % 60).toString().padStart(2, "0")
  return `${m}:${s}`
}

interface WidgetMessageInputProps {
  conversationId: string | null
  otherUserId: string
  onSend: (content: string) => void
  onSendFile: (files: File[]) => Promise<void>
  onSendVoice?: (content: string, metadata: { url: string; duration: number; size: number; mime_type: string }) => void
  onTyping: () => void
  isSending: boolean
}

function WidgetMessageInput({
  conversationId,
  otherUserId,
  onSend,
  onSendFile,
  onSendVoice,
  onTyping,
  isSending,
}: WidgetMessageInputProps) {
  const t = useTranslations("messaging")
  const router = useRouter()
  const [value, setValue] = useState("")
  const [modalOpen, setModalOpen] = useState(false)
  const [isUploading, setIsUploading] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const onTypingRef = useRef(onTyping)
  const hasContent = value.trim().length > 0

  const voice = useVoiceRecorder()
  const isRecording = voice.state === "recording"
  const isVoiceUploading = voice.state === "uploading"

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

  function handleProposal() {
    if (conversationId) {
      router.push(`/projects/new?to=${otherUserId}&conversation=${conversationId}`)
    }
  }

  const handleUploadFiles = useCallback(
    async (files: File[]) => {
      setIsUploading(true)
      setUploadError(null)
      try {
        await onSendFile(files)
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
        <div className="border-t border-gray-100 bg-red-50 px-3 py-1.5 dark:border-gray-800 dark:bg-red-900/20" role="alert">
          <p className="text-[11px] text-red-600 dark:text-red-400">{uploadError}</p>
        </div>
      )}

      {/* Voice recording bar */}
      {isRecording && (
        <div className="flex items-center gap-2 border-t border-gray-100 bg-red-50 px-3 py-2 dark:border-gray-800 dark:bg-red-900/20">
          <span className="h-2 w-2 shrink-0 animate-pulse rounded-full bg-red-500" />
          <span className="text-[11px] font-medium text-red-600 dark:text-red-400">
            {t("recording")}
          </span>
          <span className="font-mono text-[11px] text-red-500 dark:text-red-400">
            {formatRecordingDuration(voice.duration)}
          </span>
          <div className="flex-1" />
          <button
            type="button"
            onClick={voice.cancelRecording}
            className="rounded px-2 py-1 text-[10px] text-gray-500 hover:bg-gray-200 dark:text-gray-400 dark:hover:bg-gray-700"
            aria-label={t("cancelRecording")}
          >
            {t("cancelRecording")}
          </button>
          <button
            type="button"
            onClick={handleVoiceToggle}
            className="flex h-7 w-7 items-center justify-center rounded-full bg-red-500 text-white transition-all hover:bg-red-600 active:scale-[0.95]"
            aria-label={t("sendMessage")}
          >
            {isVoiceUploading ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" strokeWidth={1.5} />
            ) : (
              <Square className="h-3 w-3" strokeWidth={2} fill="currentColor" />
            )}
          </button>
        </div>
      )}

      {/* Normal input bar */}
      {!isRecording && (
        <form
          onSubmit={handleSubmit}
          className="flex shrink-0 items-center gap-1.5 border-t border-gray-100 bg-white px-3 py-2 dark:border-gray-800 dark:bg-gray-900"
        >
          {/* File attachment button */}
          <button
            type="button"
            onClick={() => setModalOpen(true)}
            disabled={isDisabled}
            className={cn(
              "shrink-0 rounded-lg p-1.5 text-gray-400 transition-colors",
              "hover:bg-gray-100 hover:text-gray-600",
              "dark:hover:bg-gray-800 dark:hover:text-gray-300",
              isDisabled && "pointer-events-none opacity-50",
            )}
            aria-label={t("fileUpload")}
          >
            {isUploading ? (
              <Loader2 className="h-4 w-4 animate-spin" strokeWidth={1.5} />
            ) : (
              <Paperclip className="h-4 w-4" strokeWidth={1.5} />
            )}
          </button>

          {/* Proposal button */}
          <button
            type="button"
            onClick={handleProposal}
            disabled={isDisabled}
            className={cn(
              "shrink-0 rounded-lg p-1.5 text-gray-400 transition-colors",
              "hover:bg-rose-50 hover:text-rose-600",
              "dark:hover:bg-rose-500/10 dark:hover:text-rose-400",
              isDisabled && "pointer-events-none opacity-50",
            )}
            aria-label={t("propose")}
          >
            <FileText className="h-4 w-4" strokeWidth={1.5} />
          </button>

          {/* Voice recorder button */}
          {onSendVoice && (
            <button
              type="button"
              onClick={handleVoiceToggle}
              disabled={isDisabled}
              className={cn(
                "shrink-0 rounded-lg p-1.5 text-gray-400 transition-colors",
                "hover:bg-purple-50 hover:text-purple-600",
                "dark:hover:bg-purple-500/10 dark:hover:text-purple-400",
                isDisabled && "pointer-events-none opacity-50",
              )}
              aria-label={t("voiceMessage")}
            >
              <Mic className="h-4 w-4" strokeWidth={1.5} />
            </button>
          )}

          {/* Input */}
          <input
            type="text"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder={t("writeMessage")}
            aria-label={t("writeMessage")}
            disabled={isDisabled}
            className={cn(
              "h-9 flex-1 rounded-full bg-gray-100/80 px-3.5 text-xs text-gray-900",
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
              "shrink-0 rounded-full p-2 transition-all duration-200",
              value.trim() && !isDisabled
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
