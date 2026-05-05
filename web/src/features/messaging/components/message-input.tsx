"use client"

import { useState, useRef, useCallback, useEffect } from "react"
import { Paperclip, Send, Loader2, FileText, X, Mic, Square, Plus, Trash2 } from "lucide-react"
import { useTranslations } from "next-intl"
import { useRouter } from "@i18n/navigation"
import { cn } from "@/shared/lib/utils"
import { useHasPermission } from "@/shared/hooks/use-permissions"
import { getPresignedURL } from "../api/messaging-api"
import { FileUploadModal } from "@/shared/components/file-upload-modal"
import { useVoiceRecorder } from "../hooks/use-voice-recorder"
import { Button } from "@/shared/components/ui/button"

import { Input } from "@/shared/components/ui/input"
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
  const canPropose = useHasPermission("proposals.create")
  const [value, setValue] = useState("")
  const [isUploading, setIsUploading] = useState(false)
  const [modalOpen, setModalOpen] = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
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
    setMobileMenuOpen(false)
  }

  const handleUploadFiles = useCallback(
    async (files: File[]) => {
      setIsUploading(true)
      setUploadError(null)
      try {
        for (const file of files) {
          const { upload_url, public_url } = await getPresignedURL(
            file.name,
            file.type,
          )
          await fetch(upload_url, {
            method: "PUT",
            body: file,
            headers: { "Content-Type": file.type },
          })
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
          className="border-t border-border bg-primary-soft px-4 py-2"
          role="alert"
        >
          <p className="text-xs text-primary-deep">{uploadError}</p>
        </div>
      )}

      {/* Reply preview bar */}
      {replyTo && (
        <div className="flex items-center gap-2 border-t border-border bg-primary-soft/40 px-4 py-2">
          <div className="min-w-0 flex-1 border-l-2 border-primary pl-2">
            <p className="text-xs font-semibold text-primary-deep">
              {t("replyingTo", { name: replyTo.senderName })}
            </p>
            <p className="truncate text-xs text-muted-foreground">
              {replyTo.content.length > 50
                ? replyTo.content.slice(0, 50) + "..."
                : replyTo.content}
            </p>
          </div>
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={onCancelReply}
            className="shrink-0 rounded-full p-1 text-muted-foreground hover:bg-card hover:text-foreground"
            aria-label={t("cancelReply")}
          >
            <X className="h-4 w-4" strokeWidth={1.6} />
          </Button>
        </div>
      )}

      {/* Voice recording bar -- replaces the normal input */}
      {isRecording && (
        <div className="flex items-center gap-3 border-t border-border bg-primary-soft px-4 py-3">
          {/* Cancel / trash */}
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={voice.cancelRecording}
            className="flex h-9 w-9 items-center justify-center rounded-full bg-card text-muted-foreground transition-all duration-200 hover:bg-card hover:text-foreground"
            aria-label={t("cancelRecording")}
          >
            <Trash2 className="h-4 w-4" strokeWidth={1.6} />
          </Button>

          {/* Pulsing dot + timer */}
          <span className="h-2.5 w-2.5 shrink-0 animate-pulse rounded-full bg-primary-deep" />
          <span className="font-mono text-sm font-medium text-primary-deep">
            {formatRecordingDuration(voice.duration)}
          </span>

          <div className="flex-1" />

          {/* Stop and send */}
          <Button
            variant="ghost"
            size="auto"
            type="button"
            onClick={handleStopAndSend}
            className="flex h-10 w-10 items-center justify-center rounded-full bg-primary text-primary-foreground transition-all duration-200 hover:bg-primary-deep active:scale-[0.96]"
            aria-label={t("sendMessage")}
          >
            {isVoiceUploading ? (
              <Loader2 className="h-[18px] w-[18px] animate-spin" strokeWidth={1.6} />
            ) : (
              <Square className="h-4 w-4" strokeWidth={2} fill="currentColor" />
            )}
          </Button>
        </div>
      )}

      {/* Normal input bar */}
      {!isRecording && (
        <form
          onSubmit={handleSubmit}
          className="flex items-center gap-2 border-t border-border bg-card px-4 py-3"
        >
          {/* Desktop: separate buttons for attach + proposal */}
          <div className="hidden items-center gap-1 md:flex">
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={() => setModalOpen(true)}
              disabled={isDisabled}
              className={cn(
                "inline-flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground transition-colors",
                "hover:bg-primary-soft hover:text-primary-deep",
                isDisabled && "pointer-events-none opacity-50",
              )}
              aria-label={t("fileUpload")}
            >
              {isUploading ? (
                <Loader2 className="h-[18px] w-[18px] animate-spin" strokeWidth={1.6} />
              ) : (
                <Paperclip className="h-[18px] w-[18px]" strokeWidth={1.6} />
              )}
            </Button>
            {canPropose && (
              <Button
                variant="ghost"
                size="auto"
                type="button"
                onClick={handleProposal}
                disabled={isDisabled}
                className={cn(
                  "inline-flex h-9 w-9 items-center justify-center rounded-full text-muted-foreground transition-colors",
                  "hover:bg-primary-soft hover:text-primary-deep",
                  isDisabled && "pointer-events-none opacity-50",
                )}
                aria-label={t("propose")}
              >
                <FileText className="h-[18px] w-[18px]" strokeWidth={1.6} />
              </Button>
            )}
          </div>

          {/* Mobile: "+" menu for attach + proposal */}
          <div className="relative md:hidden">
            <Button
              variant="ghost"
              size="auto"
              type="button"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
              disabled={isDisabled}
              className={cn(
                "inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-muted-foreground transition-all duration-200",
                mobileMenuOpen && "rotate-45 bg-primary-soft text-primary-deep",
                "hover:bg-primary-soft hover:text-primary-deep",
                isDisabled && "pointer-events-none opacity-50",
              )}
              aria-label={t("fileUpload")}
            >
              <Plus className="h-[18px] w-[18px]" strokeWidth={1.6} />
            </Button>

            {mobileMenuOpen && (
              <div
                className={cn(
                  "absolute bottom-full left-0 mb-2 flex flex-col gap-1",
                  "rounded-xl border border-border bg-card p-1.5",
                  "shadow-[0_8px_24px_rgba(42,31,21,0.12)]",
                  "animate-in fade-in slide-in-from-bottom-2 duration-200",
                )}
              >
                <Button
                  variant="ghost"
                  size="auto"
                  type="button"
                  onClick={() => {
                    setModalOpen(true)
                    setMobileMenuOpen(false)
                  }}
                  className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-foreground transition-colors hover:bg-primary-soft"
                >
                  <Paperclip className="h-4 w-4" strokeWidth={1.6} />
                  {t("fileUpload")}
                </Button>
                {canPropose && (
                  <Button
                    variant="ghost"
                    size="auto"
                    type="button"
                    onClick={handleProposal}
                    className="flex items-center gap-2 rounded-lg px-3 py-2 text-sm text-foreground transition-colors hover:bg-primary-soft"
                  >
                    <FileText className="h-4 w-4" strokeWidth={1.6} />
                    {t("propose")}
                  </Button>
                )}
              </div>
            )}
          </div>

          {/* Input */}
          <Input
            type="text"
            value={value}
            onChange={handleInputChange}
            onKeyDown={handleKeyDown}
            onFocus={() => setMobileMenuOpen(false)}
            placeholder={t("writeMessage")}
            aria-label={t("writeMessage")}
            disabled={isDisabled}
            className={cn(
              "h-10 flex-1 rounded-full border border-border bg-background px-4 text-sm text-foreground",
              "placeholder:text-muted-foreground transition-all duration-200",
              "focus:border-primary focus:bg-card focus:outline-none focus:ring-2 focus:ring-primary/20",
              isDisabled && "opacity-50",
            )}
          />

          {/* Primary action: mic when empty, send when has text */}
          <PrimaryActionButton
            hasContent={hasContent}
            canVoice={!!onSendVoice}
            isDisabled={isDisabled}
            isSending={isSending}
            onSend={() => { const trimmed = value.trim(); if (trimmed) { onSend(trimmed, replyTo?.id); setValue(""); onCancelReply?.() } }}
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
function PrimaryActionButton({
  hasContent,
  canVoice,
  isDisabled,
  isSending,
  // onSend is wired through form submission via the type="submit" button
  // below, not invoked from this component directly. Kept in the props
  // contract so callers stay explicit about wiring.
  onSend: _onSend,
  onMic,
  sendLabel,
  micLabel,
}: {
  hasContent: boolean
  canVoice: boolean
  isDisabled: boolean
  isSending: boolean
  onSend: () => void
  onMic: () => void
  sendLabel: string
  micLabel: string
}) {
  const baseClasses =
    "shrink-0 inline-flex h-10 w-10 items-center justify-center rounded-full transition-all duration-200"

  // Has text: send button
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
          <Loader2 className="h-[18px] w-[18px] animate-spin" strokeWidth={1.6} />
        ) : (
          <Send className="h-[18px] w-[18px]" strokeWidth={1.6} />
        )}
      </Button>
    )
  }

  // Empty input + voice available: mic button
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
        <Mic className="h-[18px] w-[18px]" strokeWidth={1.6} />
      </Button>
    )
  }

  // No voice, no text: disabled send
  return (
    <Button
      variant="ghost"
      size="auto"
      type="submit"
      disabled
      className={cn(baseClasses, "bg-border text-muted-foreground")}
      aria-label={sendLabel}
    >
      <Send className="h-[18px] w-[18px]" strokeWidth={1.6} />
    </Button>
  )
}
