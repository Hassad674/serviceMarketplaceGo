import type { ColumnDef } from "@tanstack/react-table"
import { Star, Video } from "lucide-react"
import { Badge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { formatDate } from "@/shared/lib/utils"
import type { AdminReview } from "../types"

function StarRating({ rating }: { rating: number }) {
  return (
    <div className="flex items-center gap-1">
      {Array.from({ length: 5 }, (_, i) => (
        <Star
          key={i}
          className={`h-3.5 w-3.5 ${
            i < rating
              ? "fill-amber-400 text-amber-400"
              : "fill-none text-gray-300"
          }`}
        />
      ))}
      <span className="ml-1 text-sm font-medium text-foreground">{rating}/5</span>
    </div>
  )
}

export const reviewsColumns: ColumnDef<AdminReview, unknown>[] = [
  {
    id: "reviewer",
    header: "Auteur",
    cell: ({ row }) => {
      const reviewer = row.original.reviewer
      return (
        <div className="flex items-center gap-2">
          <Avatar name={reviewer.display_name} size="sm" />
          <div>
            <p className="text-sm font-medium text-foreground">{reviewer.display_name}</p>
            <p className="text-xs text-muted-foreground">{reviewer.email}</p>
          </div>
        </div>
      )
    },
  },
  {
    id: "reviewed",
    header: "Destinataire",
    cell: ({ row }) => {
      const reviewed = row.original.reviewed
      return (
        <div className="flex items-center gap-2">
          <Avatar name={reviewed.display_name} size="sm" />
          <div>
            <p className="text-sm font-medium text-foreground">{reviewed.display_name}</p>
            <p className="text-xs text-muted-foreground">{reviewed.email}</p>
          </div>
        </div>
      )
    },
  },
  {
    id: "rating",
    header: "Note",
    cell: ({ row }) => <StarRating rating={row.original.global_rating} />,
  },
  {
    id: "comment",
    header: "Commentaire",
    cell: ({ row }) => {
      const comment = row.original.comment
      if (!comment) return <span className="text-muted-foreground">-</span>
      const truncated = comment.length > 60 ? comment.slice(0, 60) + "..." : comment
      return <span className="text-sm text-foreground">{truncated}</span>
    },
  },
  {
    id: "video",
    header: "Video",
    cell: ({ row }) => {
      if (!row.original.video_url) return null
      return (
        <div className="flex items-center gap-1 text-primary">
          <Video className="h-4 w-4" />
          <span className="text-xs">Oui</span>
        </div>
      )
    },
  },
  {
    id: "reports",
    header: "Signalements",
    cell: ({ row }) => {
      const count = row.original.pending_report_count
      if (!count) return null
      return <Badge variant="destructive">{count}</Badge>
    },
  },
  {
    accessorKey: "created_at",
    header: "Date",
    cell: ({ row }) => (
      <span className="text-sm text-muted-foreground">{formatDate(row.original.created_at)}</span>
    ),
  },
]
