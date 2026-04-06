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
}

function formatTime(dateStr: string): string {
  return new Date(dateStr).toLocaleTimeString("fr-FR", {
    hour: "2-digit",
    minute: "2-digit",
  })
}

export function MessageBubble({ message, senderColorIndex }: MessageBubbleProps) {
  const bgColor = SENDER_COLORS[senderColorIndex % SENDER_COLORS.length]

  return (
    <div className="flex items-start gap-3">
      <Avatar name={message.sender_name} size="sm" className="mt-1 shrink-0" />
      <div className="min-w-0 max-w-[85%] flex-1">
        <BubbleHeader
          senderName={message.sender_name}
          senderRole={message.sender_role}
          timestamp={message.created_at}
        />
        <div className={cn("mt-1 rounded-2xl px-4 py-3", bgColor)}>
          <BubbleContent message={message} />
        </div>
      </div>
    </div>
  )
}

function BubbleHeader({ senderName, senderRole, timestamp }: {
  senderName: string
  senderRole: string
  timestamp: string
}) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm font-medium text-foreground">{senderName}</span>
      <RoleBadge role={senderRole} />
      <span className="ml-auto text-xs text-muted-foreground">
        {formatTime(timestamp)}
      </span>
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
