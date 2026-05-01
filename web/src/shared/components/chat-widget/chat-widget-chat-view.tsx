"use client"

import { useState, useRef, useEffect, useCallback } from "react"
import { ChevronLeft, ChevronDown, Paperclip, FileText, Send, Loader2, Mic, Square, Plus, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { MessageArea } from "@/features/messaging/components/message-area"
import { FileUploadModal } from "@/features/messaging/components/file-upload-modal"
import { useVoiceRecorder } from "@/features/messaging/hooks/use-voice-recorder"
import { getPresignedURL } from "@/features/messaging/api/messaging-api"
import type { Conversation, Message } from "@/features/messaging/types"
import type { PendingRecipient } from "./use-chat-widget"

import { Button } from "@/shared/components/ui/button"
import { Input } from "@/shared/components/ui/input"
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
  pendingRecipient?: PendingRecipient | null
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
  pendingRecipient,
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
  const canSendMessage = useHasPermission("messaging.send")

  if (!conversation && !pendingRecipient) {
    return (
      <div className="flex h-full items-center justify-center">
        <p className="text-xs text-gray-400 dark:text-gray-500">
          {t("noConversations")}
        </p>
      </div>
    )
  }

  const headerName = conversation?.other_org_name ?? pendingRecipient?.displayName ?? ""
  const isOnline = conversation?.online ?? false

  return (
    <div className="flex h-full flex-col">
      {/* Header */}
      <ChatViewHeader
        name={headerName}
        online={isOnline}
        typingUserName={typingUser ? headerName : undefined}
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
      {canSendMessage ? (
        <WidgetMessageInput
          conversationId={conversationId}
          otherUserId={conversation?.other_user_id ?? ""}
          onSend={onSend}
          onSendFile={onSendFile}
          onSendVoice={onSendVoice}
          onTyping={onTyping}
          isSending={isSending}
        />
      ) : (
        <WidgetNoSendPermissionBar />
      )}
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
      <Button variant="ghost" size="auto"
        onClick={onBack}
        className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
        aria-label={t("backToList")}
      >
        <ChevronLeft className="h-4 w-4" strokeWidth={1.5} />
      </Button>

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

      <Button variant="ghost" size="auto"
        onClick={onClose}
        className="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-gray-800 dark:hover:text-gray-300"
        aria-label={t("close")}
      >
        <ChevronDown className="h-4 w-4" strokeWidth={1.5} />
      </Button>
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
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
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
    setMobileMenuOpen(false)
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

  const handleStopAndSend = useCallback(async () => {
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
  }, [voice, onSendVoice, t])

  const handleStartRecording = useCallback(async () => {
    try {
      await voice.startRecording()
    } catch {
      setUploadError(t("microphonePermission"))
    }
  }, [voice, t])

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
        <div className="flex items-center gap-2 border-t border-gray-100 bg-rose-50 px-3 py-2 dark:border-gray-800 dark:bg-rose-900/20">
          {/* Cancel / trash */}
          <Button variant="ghost" size="auto"
            type="button"
            onClick={voice.cancelRecording}
            className={cn(
              "flex h-7 w-7 items-center justify-center rounded-full",
              "bg-white/80 text-gray-500 transition-all",
              "hover:bg-white hover:text-gray-700",
              "dark:bg-gray-800/80 dark:text-gray-400 dark:hover:bg-gray-800",
            )}
            aria-label={t("cancelRecording")}
          >
            <Trash2 className="h-3.5 w-3.5" strokeWidth={1.5} />
          </Button>

          {/* Red pulsing dot + timer */}
          <span className="h-2 w-2 shrink-0 animate-pulse rounded-full bg-red-500" />
          <span className="font-mono text-[11px] font-medium text-red-600 dark:text-red-400">
            {formatRecordingDuration(voice.duration)}
          </span>

          <div className="flex-1" />

          {/* Stop and send */}
          <Button variant="ghost" size="auto"
            type="button"
            onClick={handleStopAndSend}
            className="flex h-7 w-7 items-center justify-center rounded-full bg-rose-500 text-white transition-all hover:bg-rose-600 active:scale-[0.95]"
            aria-label={t("sendMessage")}
          >
            {isVoiceUploading ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" strokeWidth={1.5} />
            ) : (
              <Square className="h-3 w-3" strokeWidth={2} fill="currentColor" />
            )}
          </Button>
        </div>
      )}

      {/* Normal input bar */}
      {!isRecording && (
        <form
          onSubmit={handleSubmit}
          className="flex shrink-0 items-center gap-1.5 border-t border-gray-100 bg-white px-3 py-2 dark:border-gray-800 dark:bg-gray-900"
        >
          {/* "+" menu for attach + proposal (widget is always compact) */}
          <div className="relative">
            <Button variant="ghost" size="auto"
              type="button"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              disabled={isDisabled}
              className={cn(
                "shrink-0 rounded-full p-1.5 text-gray-400 transition-all duration-200",
                mobileMenuOpen && "rotate-45 bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-300",
                "hover:bg-gray-100 hover:text-gray-600",
                "dark:hover:bg-gray-800 dark:hover:text-gray-300",
                isDisabled && "pointer-events-none opacity-50",
              )}
              aria-label={t("fileUpload")}
            >
              <Plus className="h-4 w-4" strokeWidth={1.5} />
            </Button>

            {mobileMenuOpen && (
              <div
                className={cn(
                  "absolute bottom-full left-0 mb-2 flex flex-col gap-1",
                  "rounded-xl border border-gray-100 bg-white p-1.5 shadow-lg",
                  "dark:border-gray-700 dark:bg-gray-800",
                )}
              >
                <Button variant="ghost" size="auto"
                  type="button"
                  onClick={() => { setModalOpen(true); setMobileMenuOpen(false) }}
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-xs text-gray-600 transition-colors hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                >
                  <Paperclip className="h-3.5 w-3.5" strokeWidth={1.5} />
                  {t("fileUpload")}
                </Button>
                <Button variant="ghost" size="auto"
                  type="button"
                  onClick={handleProposal}
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-xs text-gray-600 transition-colors hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-gray-700"
                >
                  <FileText className="h-3.5 w-3.5" strokeWidth={1.5} />
                  {t("propose")}
                </Button>
              </div>
            )}
          </div>

          {/* Input */}
          <Input
            type="text"
            value={value}
            onChange={(e) => setValue(e.target.value)}
            onKeyDown={handleKeyDown}
            onFocus={() => setMobileMenuOpen(false)}
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

          {/* Primary action: mic when empty, send when has text */}
          <WidgetPrimaryAction
            hasContent={hasContent}
            canVoice={!!onSendVoice}
            isDisabled={isDisabled}
            isSending={isSending}
            onMic={handleStartRecording}
            sendLabel={t("sendMessage")}
            micLabel={t("voiceMessage")}
          />
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

/** The single right-hand button: mic when empty, send when has text. */
function WidgetPrimaryAction({
  hasContent,
  canVoice,
  isDisabled,
  isSending,
  onMic,
  sendLabel,
  micLabel,
}: {
  hasContent: boolean
  canVoice: boolean
  isDisabled: boolean
  isSending: boolean
  onMic: () => void
  sendLabel: string
  micLabel: string
}) {
  if (hasContent) {
    return (
      <Button variant="ghost" size="auto"
        type="submit"
        disabled={isDisabled}
        className={cn(
          "shrink-0 rounded-full p-2 transition-all duration-200",
          !isDisabled
            ? "bg-rose-500 text-white shadow-sm hover:bg-rose-600 active:scale-[0.96]"
            : "bg-gray-100 text-gray-300 dark:bg-gray-800 dark:text-gray-600",
        )}
        aria-label={sendLabel}
      >
        {isSending ? (
          <Loader2 className="h-4 w-4 animate-spin" strokeWidth={1.5} />
        ) : (
          <Send className="h-4 w-4" strokeWidth={1.5} />
        )}
      </Button>
    )
  }

  if (canVoice) {
    return (
      <Button variant="ghost" size="auto"
        type="button"
        onClick={onMic}
        disabled={isDisabled}
        className={cn(
          "shrink-0 rounded-full p-2 transition-all duration-200",
          !isDisabled
            ? "bg-rose-500 text-white shadow-sm hover:bg-rose-600 active:scale-[0.96]"
            : "bg-gray-100 text-gray-300 dark:bg-gray-800 dark:text-gray-600",
        )}
        aria-label={micLabel}
      >
        <Mic className="h-4 w-4" strokeWidth={1.5} />
      </Button>
    )
  }

  return (
    <Button variant="ghost" size="auto"
      type="submit"
      disabled
      className="shrink-0 rounded-full bg-gray-100 p-2 text-gray-300 dark:bg-gray-800 dark:text-gray-600"
      aria-label={sendLabel}
    >
      <Send className="h-4 w-4" strokeWidth={1.5} />
    </Button>
  )
}

/** Shown in place of the message input when the user lacks messaging.send permission. */
function WidgetNoSendPermissionBar() {
  const t = useTranslations("permissions")
  return (
    <div className="flex items-center justify-center border-t border-gray-100 bg-gray-50 px-3 py-3 dark:border-gray-800 dark:bg-gray-800/50">
      <p className="text-xs text-gray-400 dark:text-gray-500">{t("noMessagingSend")}</p>
    </div>
  )
}
