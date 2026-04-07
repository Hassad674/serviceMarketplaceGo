import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Flag, Image, MessageSquare, ExternalLink } from "lucide-react"
import { Card } from "@/shared/components/ui/card"
import { Badge, RoleBadge } from "@/shared/components/ui/badge"
import { Button } from "@/shared/components/ui/button"
import { ResolveReportDialog } from "@/shared/components/ui/resolve-report-dialog"
import { formatRelativeDate } from "@/shared/lib/utils"
import {
  useApproveMedia,
  useRejectMedia,
  useDeleteMedia,
  useApproveMessageModeration,
  useHideMessage,
  useApproveReviewModeration,
  useDeleteReview,
  useResolveReport,
} from "../hooks/use-moderation"
import type { ModerationItem } from "../types"

type ModerationCardProps = {
  item: ModerationItem
}

export function ModerationCard({ item }: ModerationCardProps) {
  return (
    <Card className="p-5 transition-all duration-200 ease-out hover:shadow-md">
      <div className="space-y-3">
        <TopRow item={item} />
        <ContentPreview item={item} />
        <div className="flex items-center justify-between">
          <UserInfo item={item} />
          <ActionButtons item={item} />
        </div>
      </div>
    </Card>
  )
}

function TopRow({ item }: { item: ModerationItem }) {
  return (
    <div className="flex items-center justify-between">
      <div className="flex items-center gap-2">
        <SourceBadge source={item.source} score={item.moderation_score} />
        <ReasonBadge item={item} />
      </div>
      <span className="text-xs text-muted-foreground">
        {formatRelativeDate(item.created_at)}
      </span>
    </div>
  )
}

function SourceBadge({ source, score }: { source: string; score: number }) {
  if (source === "human_report") {
    return (
      <Badge variant="destructive" className="gap-1">
        <Flag className="h-3 w-3" />
        Signalement
      </Badge>
    )
  }
  if (source === "auto_media") {
    return (
      <Badge variant="warning" className="gap-1">
        <Image className="h-3 w-3" />
        {`D\u00E9tection m\u00E9dia`}
      </Badge>
    )
  }
  // auto_text
  const isHighScore = score >= 0.8
  return (
    <Badge variant={isHighScore ? "destructive" : "warning"} className="gap-1">
      <MessageSquare className="h-3 w-3" />
      {`D\u00E9tection texte`}
    </Badge>
  )
}

function ReasonBadge({ item }: { item: ModerationItem }) {
  if (!item.reason) return null
  if (item.source === "auto_text" || item.source === "auto_media") {
    if (item.moderation_score > 0) {
      return (
        <span className="text-xs text-muted-foreground">
          Score: {(item.moderation_score * 100).toFixed(0)}%
        </span>
      )
    }
    return null
  }
  // human_report: show the reason
  const reasonLabels: Record<string, string> = {
    harassment: "Harc\u00E8lement",
    fraud: "Fraude",
    fraud_or_scam: "Fraude / Arnaque",
    spam: "Spam",
    inappropriate_content: "Contenu inappropri\u00E9",
    fake_profile: "Faux profil",
    unprofessional_behavior: "Comportement non professionnel",
    misleading_description: "Description trompeuse",
    other: "Autre",
  }
  return (
    <Badge variant="outline">
      {reasonLabels[item.reason] || item.reason}
    </Badge>
  )
}

function ContentPreview({ item }: { item: ModerationItem }) {
  if (!item.content_preview) return null
  return (
    <p className="line-clamp-2 text-sm text-foreground">
      {item.content_preview}
    </p>
  )
}

function UserInfo({ item }: { item: ModerationItem }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm font-medium text-foreground">
        {item.user_involved.display_name}
      </span>
      <RoleBadge role={item.user_involved.role} />
    </div>
  )
}

function ActionButtons({ item }: { item: ModerationItem }) {
  const navigate = useNavigate()

  function handleViewClick() {
    if (item.content_url) {
      navigate(item.content_url)
    }
  }

  return (
    <div className="flex items-center gap-2">
      {item.content_url && (
        <Button variant="ghost" size="sm" onClick={handleViewClick}>
          <ExternalLink className="mr-1 h-3.5 w-3.5" />
          Voir
        </Button>
      )}
      <SourceActions item={item} />
    </div>
  )
}

function SourceActions({ item }: { item: ModerationItem }) {
  if (item.source === "human_report") {
    return <ReportActions item={item} />
  }
  if (item.source === "auto_text" && item.content_type === "message") {
    return <MessageActions item={item} />
  }
  if (item.source === "auto_text" && item.content_type === "review") {
    return <ReviewActions item={item} />
  }
  if (item.source === "auto_media") {
    return <MediaActions item={item} />
  }
  return null
}

function ReportActions({ item }: { item: ModerationItem }) {
  const [dialogOpen, setDialogOpen] = useState(false)
  const resolveReport = useResolveReport()

  function handleConfirm(status: "resolved" | "dismissed", adminNote: string) {
    resolveReport.mutate(
      { reportId: item.content_id, status, adminNote },
      { onSuccess: () => setDialogOpen(false) },
    )
  }

  return (
    <>
      <Button variant="primary" size="sm" onClick={() => setDialogOpen(true)}>
        {`R\u00E9soudre`}
      </Button>
      <ResolveReportDialog
        open={dialogOpen}
        onClose={() => setDialogOpen(false)}
        onConfirm={handleConfirm}
        isPending={resolveReport.isPending}
      />
    </>
  )
}

function MessageActions({ item }: { item: ModerationItem }) {
  const approve = useApproveMessageModeration()
  const hide = useHideMessage()

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => approve.mutate(item.content_id)}
        disabled={approve.isPending}
      >
        Approuver
      </Button>
      <Button
        variant="destructive"
        size="sm"
        onClick={() => hide.mutate(item.content_id)}
        disabled={hide.isPending}
      >
        Masquer
      </Button>
    </>
  )
}

function ReviewActions({ item }: { item: ModerationItem }) {
  const approve = useApproveReviewModeration()
  const remove = useDeleteReview()

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => approve.mutate(item.content_id)}
        disabled={approve.isPending}
      >
        Approuver
      </Button>
      <Button
        variant="destructive"
        size="sm"
        onClick={() => remove.mutate(item.content_id)}
        disabled={remove.isPending}
      >
        Supprimer
      </Button>
    </>
  )
}

function MediaActions({ item }: { item: ModerationItem }) {
  const approve = useApproveMedia()
  const reject = useRejectMedia()
  const remove = useDeleteMedia()

  return (
    <>
      <Button
        variant="outline"
        size="sm"
        onClick={() => approve.mutate(item.content_id)}
        disabled={approve.isPending}
      >
        Approuver
      </Button>
      <Button
        variant="secondary"
        size="sm"
        onClick={() => reject.mutate(item.content_id)}
        disabled={reject.isPending}
      >
        Rejeter
      </Button>
      <Button
        variant="destructive"
        size="sm"
        onClick={() => remove.mutate(item.content_id)}
        disabled={remove.isPending}
      >
        Supprimer
      </Button>
    </>
  )
}
