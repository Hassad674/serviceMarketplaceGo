import { cn } from "@/shared/lib/utils"
import { Avatar } from "@/shared/components/ui/avatar"
import { RoleBadge } from "@/shared/components/ui/badge"
import { ImageContent, DocumentContent, VoiceContent, isImageMimeType } from "./message-content"
import type { AdminMessage } from "../types"

/**
 * Background colors for different senders in the chat.
 * Cycles through these to distinguish participants.
 */
const SENDER_COLORS = [
  "bg-gray-100",
  "bg-blue-50",
  "bg-amber-50",
  "bg-emerald-50",
  "bg-violet-50",
  "bg-rose-50",
] as const

type MessageBubbleProps = {
  message: AdminMessage
  senderColorIndex: number
  highlighted?: boolean
}

function formatTime(dateStr: string): string {
  return new Date(dateStr).toLocaleTimeString("fr-FR", {
    hour: "2-digit",
    minute: "2-digit",
  })
}

export function MessageBubble({ message, senderColorIndex, highlighted }: MessageBubbleProps) {
  const bgColor = SENDER_COLORS[senderColorIndex % SENDER_COLORS.length]

  return (
    <div
      id={`msg-${message.id}`}
      className={cn(
        "flex items-start gap-3 rounded-2xl p-1 transition-all duration-300",
        highlighted && "ring-2 ring-red-500 bg-red-50/30 animate-highlight-fade",
      )}
    >
      <Avatar name={message.sender_name} size="sm" className="mt-1 shrink-0" />
      <div className="min-w-0 max-w-[85%] flex-1">
        <BubbleHeader
          senderName={message.sender_name}
          senderRole={message.sender_role}
          timestamp={message.created_at}
          moderationStatus={message.moderation_status}
        />
        <div className={cn("mt-1 rounded-2xl px-4 py-3", bgColor)}>
          <BubbleContent message={message} />
        </div>
        <ModerationInfo message={message} />
      </div>
    </div>
  )
}

function BubbleHeader({ senderName, senderRole, timestamp, moderationStatus }: {
  senderName: string
  senderRole: string
  timestamp: string
  moderationStatus: string
}) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm font-medium text-foreground">{senderName}</span>
      <RoleBadge role={senderRole} />
      {moderationStatus === "flagged" && (
        <ModerationBadge status="flagged" />
      )}
      {moderationStatus === "hidden" && (
        <ModerationBadge status="hidden" />
      )}
      <span className="ml-auto text-xs text-muted-foreground">
        {formatTime(timestamp)}
      </span>
    </div>
  )
}

function ModerationBadge({ status }: { status: "flagged" | "hidden" }) {
  const styles = status === "hidden"
    ? "bg-red-100 text-red-700 border-red-200"
    : "bg-amber-100 text-amber-700 border-amber-200"
  const label = status === "hidden" ? "Masque" : "Signale"

  return (
    <span className={cn(
      "inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-[11px] font-medium",
      styles,
    )}>
      {label}
    </span>
  )
}

function ModerationInfo({ message }: { message: AdminMessage }) {
  if (!message.moderation_status) {
    return null
  }

  const labels = message.moderation_labels ?? []
  const significantLabels = labels.filter((l) => l.Score >= 0.3)

  if (significantLabels.length === 0) {
    return null
  }

  return (
    <div className="mt-1 flex flex-wrap gap-1">
      {significantLabels.map((label) => (
        <span
          key={label.Name}
          className="inline-flex items-center rounded bg-gray-100 px-1.5 py-0.5 text-[10px] text-gray-600"
        >
          {label.Name} ({Math.round(label.Score * 100)}%)
        </span>
      ))}
    </div>
  )
}

function BubbleContent({ message }: { message: AdminMessage }) {
  if (message.type === "voice" && message.metadata) {
    return <VoiceContent metadata={message.metadata} />
  }

  if (message.type === "file" && message.metadata) {
    if (isImageMimeType(message.metadata)) {
      return <ImageContent metadata={message.metadata} />
    }
    return <DocumentContent metadata={message.metadata} />
  }

  return (
    <p className="whitespace-pre-wrap text-sm leading-relaxed text-foreground/90">
      {message.content}
    </p>
  )
}
