"use client"

import { useState, useEffect, useCallback, useRef } from "react"
import {
  ArrowLeft,
  Phone,
  MoreHorizontal,
  Paperclip,
  FileText,
  Send,
  Loader2,
  Mic,
  Square,
  Trash2,
  Smile,
} from "lucide-react"
import Image from "next/image"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { Portrait } from "@/shared/components/ui/portrait"
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

// Stable portrait id derived from a string seed (keeps the conversation
// avatar consistent without keeping a global lookup).
function portraitIdFor(seed: string): number {
  let h = 0
  for (let i = 0; i < seed.length; i++) {
    h = (h * 31 + seed.charCodeAt(i)) >>> 0
  }
  return h % 6
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

// ChatWidgetChatView — Soleil v2 widget conversation view.
//
// Reuses MessageArea (the canonical bubble renderer) so behaviour and
// proposal/system bubbles stay identical to the full /messages page.
// The header and composer are widget-specific compact variants.
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
        <p className="text-xs text-muted-foreground">{t("noConversations")}</p>
      </div>
    )
  }

  const headerName =
    conversation?.other_org_name ?? pendingRecipient?.displayName ?? ""
  const isOnline = conversation?.online ?? false
  const photoUrl = conversation?.other_photo_url

  return (
    <div className="flex h-full flex-col">
      <ChatViewHeader
        name={headerName}
        photoUrl={photoUrl}
        seedId={conversation?.id ?? pendingRecipient?.orgId ?? ""}
        online={isOnline}
        typing={!!typingUser}
        onBack={onBack}
        onClose={onClose}
      />

      <div className="flex min-h-0 flex-1 flex-col bg-background">
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
  photoUrl?: string
  seedId: string
  online: boolean
  typing: boolean
  onBack: () => void
  onClose: () => void
}

function ChatViewHeader({
  name,
  photoUrl,
  seedId,
  online,
  typing,
  onBack,
  onClose,
}: ChatViewHeaderProps) {
  const t = useTranslations("messaging")

  return (
    <div className="flex shrink-0 items-center gap-3 border-b border-border bg-card px-4 py-3">
      <Button
        variant="ghost"
        size="auto"
        onClick={onBack}
        className="rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-border hover:text-foreground"
        aria-label={t("messagingWidget_back")}
      >
        <ArrowLeft className="h-4 w-4" strokeWidth={1.6} />
      </Button>

      <div className="relative shrink-0">
        {photoUrl ? (
          <Image
            src={photoUrl}
            alt={name}
            width={36}
            height={36}
            className="h-9 w-9 rounded-full object-cover"
            unoptimized
          />
        ) : (
          <Portrait id={portraitIdFor(seedId)} size={36} alt={name} />
        )}
        {online && (
          <span
            className="absolute bottom-0 right-0 h-2.5 w-2.5 rounded-full border-2 border-card bg-emerald-500"
            aria-label={t("online")}
          />
        )}
      </div>

      <div className="min-w-0 flex-1">
        <p className="truncate font-serif text-[15px] font-semibold leading-tight text-foreground">
          {name}
        </p>
        {typing ? (
          <p className="truncate text-[11px] italic text-primary">
            {t("typingShort")}
          </p>
        ) : (
          <p
            className={cn(
              "text-[11px] font-medium",
              online ? "text-success" : "text-muted-foreground",
            )}
          >
            {online ? t("online") : t("offline")}
          </p>
        )}
      </div>

      <Button
        variant="ghost"
        size="auto"
        className="rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-primary-soft hover:text-primary-deep"
        aria-label={t("sendMessage")}
        type="button"
      >
        <Phone className="h-4 w-4" strokeWidth={1.6} />
      </Button>
      <Button
        variant="ghost"
        size="auto"
        onClick={onClose}
        className="rounded-full p-1.5 text-muted-foreground transition-colors hover:bg-border hover:text-foreground"
        aria-label={t("messagingWidget_close")}
        type="button"
      >
        <MoreHorizontal className="h-4 w-4" strokeWidth={1.6} />
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
  const [menuOpen, setMenuOpen] = useState(false)
  const onTypingRef = useRef(onTyping)
  const hasContent = value.trim().length > 0

  const voice = useVoiceRecorder()
  const isRecording = voice.state === "recording"
  const isVoiceUploading = voice.state === "uploading"

  useEffect(() => {
    onTypingRef.current = onTyping
  }, [onTyping])

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
    setMenuOpen(false)
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
        <div
          className="border-t border-border bg-primary-soft px-4 py-1.5"
          role="alert"
        >
          <p className="text-[11px] text-primary-deep">{uploadError}</p>
        </div>
      )}

      {isRecording && (
        <div className="flex items-center gap-2 border-t border-border bg-primary-soft px-3 py-2">
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={voice.cancelRecording}
            className="flex h-8 w-8 items-center justify-center rounded-full bg-card text-muted-foreground hover:text-foreground"
            aria-label={t("cancelRecording")}
          >
            <Trash2 className="h-3.5 w-3.5" strokeWidth={1.6} />
          </Button>
          <span className="h-2 w-2 shrink-0 animate-pulse rounded-full bg-primary-deep" />
          <span className="font-mono text-[11px] font-medium text-primary-deep">
            {formatRecordingDuration(voice.duration)}
          </span>
          <div className="flex-1" />
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={handleStopAndSend}
            className="flex h-8 w-8 items-center justify-center rounded-full bg-primary text-primary-foreground transition-all hover:bg-primary-deep active:scale-[0.96]"
            aria-label={t("sendMessage")}
          >
            {isVoiceUploading ? (
              <Loader2 className="h-3.5 w-3.5 animate-spin" strokeWidth={1.6} />
            ) : (
              <Square className="h-3 w-3" strokeWidth={2} fill="currentColor" />
            )}
          </Button>
        </div>
      )}

      {!isRecording && (
        <form
          onSubmit={handleSubmit}
          className="flex shrink-0 items-center gap-2 border-t border-border bg-card px-3 py-2.5"
        >
          {/* Plus menu (attach + proposal) */}
          <div className="relative">
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={() => setMenuOpen(!menuOpen)}
              disabled={isDisabled}
              className={cn(
                "shrink-0 rounded-full p-1.5 text-muted-foreground transition-all duration-200",
                menuOpen && "rotate-45 bg-primary-soft text-primary-deep",
                "hover:bg-border hover:text-foreground",
                isDisabled && "pointer-events-none opacity-50",
              )}
              aria-label={t("fileUpload")}
            >
              <Paperclip className="h-4 w-4" strokeWidth={1.6} />
            </Button>

            {menuOpen && (
              <div className="absolute bottom-full left-0 mb-2 flex flex-col gap-1 rounded-xl border border-border bg-card p-1.5 shadow-[0_8px_24px_rgba(42,31,21,0.12)]">
                <Button
                  variant="ghost"
                  size="auto"
                  type="button"
                  onClick={() => {
                    setModalOpen(true)
                    setMenuOpen(false)
                  }}
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-xs text-foreground transition-colors hover:bg-primary-soft"
                >
                  <Paperclip className="h-3.5 w-3.5" strokeWidth={1.6} />
                  {t("fileUpload")}
                </Button>
                <Button
                  variant="ghost"
                  size="auto"
                  type="button"
                  onClick={handleProposal}
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-xs text-foreground transition-colors hover:bg-primary-soft"
                >
                  <FileText className="h-3.5 w-3.5" strokeWidth={1.6} />
                  {t("propose")}
                </Button>
              </div>
            )}
          </div>

          {/* Input — pill-shaped */}
          <div className="relative flex-1">
            <Input
              type="text"
              value={value}
              onChange={(e) => setValue(e.target.value)}
              onKeyDown={handleKeyDown}
              onFocus={() => setMenuOpen(false)}
              placeholder={t("writeMessage")}
              aria-label={t("writeMessage")}
              disabled={isDisabled}
              className={cn(
                "h-10 w-full rounded-full border border-border bg-background pl-4 pr-9 text-sm text-foreground",
                "placeholder:text-muted-foreground transition-all duration-200",
                "focus:border-primary focus:bg-card focus:outline-none focus:ring-2 focus:ring-primary/20",
                isDisabled && "opacity-50",
              )}
            />
            <span className="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground">
              <Smile className="h-4 w-4" strokeWidth={1.6} aria-hidden />
            </span>
          </div>

          {/* Primary action */}
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
  const baseClasses =
    "shrink-0 inline-flex h-9 w-9 items-center justify-center rounded-full transition-all duration-200"

  if (hasContent) {
    return (
      <Button
        variant="ghost"
        size="auto"
        type="submit"
        disabled={isDisabled}
        className={cn(
          baseClasses,
          !isDisabled
            ? "bg-primary text-primary-foreground hover:bg-primary-deep active:scale-[0.96]"
            : "bg-border text-muted-foreground",
        )}
        aria-label={sendLabel}
      >
        {isSending ? (
          <Loader2 className="h-4 w-4 animate-spin" strokeWidth={1.6} />
        ) : (
          <Send className="h-4 w-4" strokeWidth={1.6} />
        )}
      </Button>
    )
  }

  if (canVoice) {
    return (
      <Button
        variant="ghost"
        size="auto"
        type="button"
        onClick={onMic}
        disabled={isDisabled}
        className={cn(
          baseClasses,
          !isDisabled
            ? "bg-primary text-primary-foreground hover:bg-primary-deep active:scale-[0.96]"
            : "bg-border text-muted-foreground",
        )}
        aria-label={micLabel}
      >
        <Mic className="h-4 w-4" strokeWidth={1.6} />
      </Button>
    )
  }

  return (
    <Button
      variant="ghost"
      size="auto"
      type="submit"
      disabled
      className={cn(baseClasses, "bg-border text-muted-foreground")}
      aria-label={sendLabel}
    >
      <Send className="h-4 w-4" strokeWidth={1.6} />
    </Button>
  )
}

/** Shown in place of the message input when the user lacks messaging.send permission. */
function WidgetNoSendPermissionBar() {
  const t = useTranslations("permissions")
  return (
    <div className="flex items-center justify-center border-t border-border bg-background px-3 py-3">
      <p className="text-xs text-muted-foreground">{t("noMessagingSend")}</p>
    </div>
  )
}

