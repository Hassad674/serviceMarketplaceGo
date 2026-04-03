import { useState } from "react"
import { useParams, useNavigate, Link } from "react-router-dom"
import {
  ArrowLeft, MessageSquare, Calendar, Hash,
  EyeOff, FileText, ExternalLink,
} from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { Card, CardContent } from "@/shared/components/ui/card"
import { Button } from "@/shared/components/ui/button"
import { RoleBadge, Badge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { formatDate, formatRelativeDate } from "@/shared/lib/utils"
import { useConversation, useConversationMessages } from "../hooks/use-conversations"
import type { AdminMessage, ConversationParticipant } from "../types"

export function ConversationDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [cursor] = useState("")

  const { data: convData, isLoading: convLoading, error: convError } = useConversation(id!)
  const { data: msgData, isLoading: msgsLoading, error: msgsError } =
    useConversationMessages(id!, cursor)

  const conversation = convData?.data
  const messages = msgData?.data ?? []
  const participants = conversation?.participants ?? []

  if (convLoading || (msgsLoading && cursor === "")) {
    return <DetailSkeleton />
  }

  if (convError || msgsError || !conversation) {
    return (
      <div className="space-y-6">
        <BackButton onClick={() => navigate("/conversations")} />
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Erreur lors du chargement de la conversation
        </div>
      </div>
    )
  }

  const title = participants.map((p) => p.display_name).join(" et ")

  return (
    <div className="space-y-6">
      <BackButton onClick={() => navigate("/conversations")} />
      <PageHeader title={title} />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-4">
        {/* Messages area */}
        <div className="lg:col-span-3 space-y-4">
          <MessageList messages={messages} />

          {/* Read-only indicator */}
          <div className="flex items-center justify-center gap-2 rounded-lg border border-border bg-muted/50 px-4 py-3 text-sm text-muted-foreground">
            <EyeOff className="h-4 w-4" />
            Lecture seule — vue de mod&eacute;ration
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
        </div>
      </div>
    </div>
  )
}

/* ─── Sub-components ──────────────────────────────────────────────── */

function BackButton({ onClick }: { onClick: () => void }) {
  return (
    <Button variant="ghost" size="sm" onClick={onClick}>
      <ArrowLeft className="h-4 w-4" /> Retour aux conversations
    </Button>
  )
}

function MessageList({ messages }: { messages: AdminMessage[] }) {
  if (messages.length === 0) {
    return (
      <Card>
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
    <Card>
      <CardContent className="p-0">
        <div className="divide-y divide-border">
          {messages.map((msg) => (
            <MessageRow key={msg.id} message={msg} />
          ))}
        </div>
      </CardContent>
    </Card>
  )
}

const MESSAGE_TYPE_LABELS: Record<string, string> = {
  proposal_sent: "Proposition envoyée",
  proposal_accepted: "Proposition acceptée",
  proposal_declined: "Proposition refusée",
  proposal_modified: "Proposition modifiée",
  proposal_paid: "Paiement effectué",
  proposal_payment_requested: "Paiement demandé",
  proposal_completion_requested: "Achèvement demandé",
  proposal_completed: "Mission terminée",
  proposal_completion_rejected: "Achèvement rejeté",
  evaluation_request: "Demande d'évaluation",
  call_ended: "Appel terminé",
  call_missed: "Appel manqué",
}

function isSystemMessage(type: string): boolean {
  return type !== "text" && type !== "file" && type !== "voice"
}

function MessageRow({ message }: { message: AdminMessage }) {
  if (isSystemMessage(message.type)) {
    return <SystemMessageRow message={message} />
  }

  return (
    <div className="flex gap-3 px-4 py-3">
      <Avatar name={message.sender_name} size="sm" className="mt-0.5 shrink-0" />
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium text-foreground">
            {message.sender_name}
          </span>
          <RoleBadge role={message.sender_role} />
          <span className="ml-auto shrink-0 text-xs text-muted-foreground">
            {formatRelativeDate(message.created_at)}
          </span>
        </div>
        {message.type === "file" && message.metadata ? (
          <FileMessageContent metadata={message.metadata} />
        ) : (
          <p className="mt-1 whitespace-pre-wrap text-sm text-foreground/80">
            {message.content}
          </p>
        )}
        {message.reply_to_id && (
          <p className="mt-1 text-xs text-muted-foreground italic">
            En r&eacute;ponse &agrave; un message
          </p>
        )}
      </div>
    </div>
  )
}

function SystemMessageRow({ message }: { message: AdminMessage }) {
  const label = MESSAGE_TYPE_LABELS[message.type] || message.type
  return (
    <div className="flex items-center justify-center gap-2 px-4 py-2">
      <Badge variant="outline">{label}</Badge>
      <span className="text-xs text-muted-foreground">
        {formatRelativeDate(message.created_at)}
      </span>
    </div>
  )
}

function FileMessageContent({ metadata }: { metadata: Record<string, unknown> }) {
  const filename = (metadata.filename as string) || "Fichier"
  return (
    <div className="mt-1 inline-flex items-center gap-2 rounded-lg border border-border bg-muted/50 px-3 py-2 text-sm">
      <FileText className="h-4 w-4 text-muted-foreground" />
      <span className="text-foreground">{filename}</span>
    </div>
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
        <InfoRow icon={Calendar} label="Création" value={formatDate(createdAt)} />
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

function DetailSkeleton() {
  return (
    <div className="space-y-6">
      <Skeleton className="h-8 w-48" />
      <Skeleton className="h-10 w-72" />
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-4">
        <div className="lg:col-span-3 space-y-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-xl" />
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
