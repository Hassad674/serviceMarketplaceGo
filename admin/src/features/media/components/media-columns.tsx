import type { ColumnDef } from "@tanstack/react-table"
import type { AdminMedia } from "../types"
import { Badge } from "@/shared/components/ui/badge"
import { MediaPreview } from "./media-preview"
import { formatRelativeDate } from "@/shared/lib/utils"

const statusVariants = {
  pending: "warning",
  approved: "success",
  flagged: "destructive",
  rejected: "outline",
} as const

const statusLabels: Record<string, string> = {
  pending: "En attente",
  approved: "Approuv\u00E9",
  flagged: "Signal\u00E9",
  rejected: "Rejet\u00E9",
}

const contextLabels: Record<string, string> = {
  profile_photo: "Photo de profil",
  profile_video: "Vid\u00E9o de profil",
  message_attachment: "Pi\u00E8ce jointe",
  review_video: "Vid\u00E9o d'avis",
  job_video: "Vid\u00E9o d'offre",
  referrer_video: "Vid\u00E9o parrain",
  identity_document: "Pi\u00E8ce d'identit\u00E9",
}

export const mediaColumns: ColumnDef<AdminMedia, unknown>[] = [
  {
    id: "preview",
    header: "Aper\u00E7u",
    cell: ({ row }) => (
      <MediaPreview
        fileUrl={row.original.file_url}
        fileType={row.original.file_type}
        fileName={row.original.file_name}
        size="sm"
      />
    ),
  },
  {
    id: "file",
    header: "Fichier",
    cell: ({ row }) => {
      const m = row.original
      const typeLabel = m.file_type.split("/")[0]
      return (
        <div>
          <p className="max-w-[200px] truncate font-medium text-foreground text-sm">{m.file_name}</p>
          <p className="text-xs text-muted-foreground">{typeLabel}</p>
        </div>
      )
    },
  },
  {
    id: "uploader",
    header: "Utilisateur",
    cell: ({ row }) => {
      const m = row.original
      return (
        <div>
          <p className="font-medium text-foreground text-sm">{m.uploader_display_name}</p>
          <p className="text-xs text-muted-foreground">{m.uploader_email}</p>
        </div>
      )
    },
  },
  {
    id: "context",
    header: "Contexte",
    cell: ({ row }) => (
      <Badge variant="outline">
        {contextLabels[row.original.context] || row.original.context}
      </Badge>
    ),
  },
  {
    id: "status",
    header: "Statut",
    cell: ({ row }) => {
      const status = row.original.moderation_status
      const variant = statusVariants[status] || ("outline" as const)
      return (
        <Badge variant={variant}>
          {statusLabels[status] || status}
        </Badge>
      )
    },
  },
  {
    id: "score",
    header: "Score",
    cell: ({ row }) => {
      const score = row.original.moderation_score
      if (score === 0) return <span className="text-sm text-muted-foreground">-</span>
      return <span className="text-sm font-mono">{score.toFixed(1)}%</span>
    },
  },
  {
    id: "date",
    header: "Date",
    cell: ({ row }) => (
      <span className="text-sm text-muted-foreground">
        {formatRelativeDate(row.original.created_at)}
      </span>
    ),
  },
]
