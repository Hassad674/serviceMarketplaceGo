import { useState, useMemo, useEffect } from "react"
import { useParams, useNavigate, useSearchParams, Link } from "react-router-dom"
import {
  ArrowLeft, MessageSquare, Calendar, Hash,
  EyeOff, ExternalLink, Flag,
} from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { Card, CardContent } from "@/shared/components/ui/card"
import { Button } from "@/shared/components/ui/button"
import { RoleBadge, Badge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { ReportList } from "@/shared/components/ui/report-list"
import { ResolveReportDialog } from "@/shared/components/ui/resolve-report-dialog"
import { formatDate, formatRelativeDate } from "@/shared/lib/utils"
import { useConversation, useConversationMessages } from "../hooks/use-conversations"
import { useConversationReports, useResolveReport } from "@/shared/hooks/use-reports"
import { MessageBubble } from "./message-bubble"
import { SystemMessage, isSystemMessage } from "./system-message"
import type { AdminMessage, ConversationParticipant } from "../types"

export function ConversationDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const [cursor] = useState("")

  const highlightId = searchParams.get("highlight") ?? undefined

  const { data: convData, isLoading: convLoading, error: convError } = useConversation(id!)
  const { data: msgData, isLoading: msgsLoading, error: msgsError } =
    useConversationMessages(id!, cursor)

  const conversation = convData?.data
  const messages = msgData?.data ?? []
  const participants = conversation?.participants ?? []

  // Build a map of sender_id -> color index for consistent bubble colors
  const senderColorMap = useMemo(() => {
    return buildSenderColorMap(messages)
  }, [messages])

  // Scroll to highlighted message once messages are loaded
  useEffect(() => {
    if (!highlightId || msgsLoading) return
    const el = document.getElementById(`msg-${highlightId}`)
    if (el) {
      el.scrollIntoView({ behavior: "smooth", block: "center" })
    }
  }, [highlightId, msgsLoading])

  if (convLoading || (msgsLoading && cursor === "")) {
    return <DetailSkeleton />
  }

  if (convError || msgsError || !conversation) {
    return (
      <div className="space-y-6">
        <BackButton onClick={() => navigate(-1)} />
        <ErrorBanner />
      </div>
    )
  }

  const title = participants.map((p) => p.display_name).join(" et ")

  return (
    <div className="space-y-6">
      <BackButton onClick={() => navigate(-1)} />
      <PageHeader title={title} />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-4">
        {/* Chat area */}
        <div className="lg:col-span-3 flex flex-col">
          <ChatArea
            messages={messages}
            senderColorMap={senderColorMap}
            highlightId={highlightId}
          />

          {/* Read-only indicator */}
          <div className="mt-4 flex items-center justify-center gap-2 rounded-xl border border-border bg-muted/50 px-4 py-3 text-sm text-muted-foreground">
            <EyeOff className="h-4 w-4" />
            Lecture seule — vue de moderation
          </div>
        </div>

        {/* Sidebar */}
        <div className="space-y-4">
          {participants.map((p) => (
            <ParticipantCard key={p.id} participant={p} />
          ))}

          <ConversationInfoCard
            createdAt={conversation.created_at}
            messageCount={conversation.message_count}
            lastMessageAt={conversation.last_message_at}
          />

          <ReportsSection conversationId={id!} />
        </div>
      </div>
    </div>
  )
}

/* ── Helpers ─────────────────────────────────────────────────────── */

function buildSenderColorMap(messages: AdminMessage[]): Map<string, number> {
  const map = new Map<string, number>()
  let index = 0
  for (const msg of messages) {
    if (!map.has(msg.sender_id) && !isSystemMessage(msg.type)) {
      map.set(msg.sender_id, index)
      index++
    }
  }
  return map
}

/* ── Sub-components ──────────────────────────────────────────────── */

function BackButton({ onClick }: { onClick: () => void }) {
  return (
    <Button variant="ghost" size="sm" onClick={onClick}>
      <ArrowLeft className="h-4 w-4" /> Retour aux conversations
    </Button>
  )
}

function ErrorBanner() {
  return (
    <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
      Erreur lors du chargement de la conversation
    </div>
  )
}

function ChatArea({ messages, senderColorMap, highlightId }: {
  messages: AdminMessage[]
  senderColorMap: Map<string, number>
  highlightId?: string
}) {
  if (messages.length === 0) {
    return (
      <Card className="flex-1">
        <CardContent className="p-0">
          <EmptyState
            icon={MessageSquare}
            title="Aucun message"
            description="Cette conversation ne contient aucun message."
          />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className="flex-1">
      <CardContent className="p-0">
        <div className="flex max-h-[70vh] flex-col gap-3 overflow-y-auto px-5 py-4">
          {messages.map((msg) => (
            <ChatMessage
              key={msg.id}
              message={msg}
              senderColorMap={senderColorMap}
              highlighted={msg.id === highlightId}
            />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

function ChatMessage({ message, senderColorMap, highlighted }: {
  message: AdminMessage
  senderColorMap: Map<string, number>
  highlighted?: boolean
}) {
  if (isSystemMessage(message.type)) {
    return <SystemMessage message={message} />
  }

  const colorIndex = senderColorMap.get(message.sender_id) ?? 0
  return (
    <MessageBubble
      message={message}
      senderColorIndex={colorIndex}
      highlighted={highlighted}
    />
  )
}

function ParticipantCard({ participant }: { participant: ConversationParticipant }) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-3 pt-4 text-center">
        <Avatar name={participant.display_name} size="md" />
        <div>
          <p className="text-sm font-semibold text-foreground">{participant.display_name}</p>
          <p className="text-xs text-muted-foreground">{participant.email}</p>
        </div>
        <RoleBadge role={participant.role} />
        <Link
          to={`/users/${participant.id}`}
          className="inline-flex items-center gap-1 text-xs font-medium text-primary hover:underline"
        >
          <ExternalLink className="h-3 w-3" />
          Voir le profil
        </Link>
      </CardContent>
    </Card>
  )
}

function ConversationInfoCard({ createdAt, messageCount, lastMessageAt }: {
  createdAt: string
  messageCount: number
  lastMessageAt: string | null
}) {
  return (
    <Card>
      <CardContent className="space-y-3 pt-4">
        <h4 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
          Infos
        </h4>
        <InfoRow icon={Calendar} label="Creation" value={formatDate(createdAt)} />
        <InfoRow icon={Hash} label="Messages" value={String(messageCount)} />
        {lastMessageAt && (
          <InfoRow
            icon={MessageSquare}
            label="Dernier message"
            value={formatRelativeDate(lastMessageAt)}
          />
        )}
      </CardContent>
    </Card>
  )
}

function InfoRow({ icon: Icon, label, value }: {
  icon: React.ElementType
  label: string
  value: string
}) {
  return (
    <div className="flex items-center justify-between text-sm">
      <div className="flex items-center gap-2 text-muted-foreground">
        <Icon className="h-4 w-4" />
        {label}
      </div>
      <span className="font-medium text-foreground">{value}</span>
    </div>
  )
}

function ReportsSection({ conversationId }: { conversationId: string }) {
  const { data, isLoading } = useConversationReports(conversationId)
  const resolveMutation = useResolveReport()
  const [resolveTarget, setResolveTarget] = useState<{
    id: string
    defaultStatus: "resolved" | "dismissed"
  } | null>(null)

  const reports = data?.data ?? []

  if (isLoading) {
    return (
      <Card>
        <CardContent className="pt-4">
          <Skeleton className="h-24" />
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardContent className="space-y-3 pt-4">
        <div className="flex items-center gap-2">
          <Flag className="h-4 w-4 text-muted-foreground" />
          <h4 className="text-sm font-semibold uppercase tracking-wider text-muted-foreground">
            Signalements
          </h4>
          {reports.length > 0 && (
            <Badge variant="destructive">{reports.length}</Badge>
          )}
        </div>
        <ReportList
          reports={reports}
          onResolve={(reportId) =>
            setResolveTarget({ id: reportId, defaultStatus: "resolved" })
          }
          onDismiss={(reportId) =>
            setResolveTarget({ id: reportId, defaultStatus: "dismissed" })
          }
          isResolving={resolveMutation.isPending}
        />
        {resolveTarget && (
          <ResolveReportDialog
            open={!!resolveTarget}
            onClose={() => setResolveTarget(null)}
            onConfirm={(status, adminNote) => {
              resolveMutation.mutate(
                { reportId: resolveTarget.id, status, adminNote },
                { onSuccess: () => setResolveTarget(null) },
              )
            }}
            isPending={resolveMutation.isPending}
            defaultStatus={resolveTarget.defaultStatus}
          />
        )}
      </CardContent>
    </Card>
  )
}

function DetailSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-10 w-72" />
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-4">
        <div className="lg:col-span-3 space-y-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-xl" />
          ))}
        </div>
        <div className="space-y-4">
          <Skeleton className="h-40 rounded-xl" />
          <Skeleton className="h-40 rounded-xl" />
          <Skeleton className="h-32 rounded-xl" />
        </div>
      </div>
    </div>
  )
}
