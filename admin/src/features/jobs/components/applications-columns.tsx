import type { ColumnDef } from "@tanstack/react-table"
import { Badge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { formatDate } from "@/shared/lib/utils"
import type { AdminJobApplication } from "../types"

export const applicationsColumns: ColumnDef<AdminJobApplication, unknown>[] = [
  {
    id: "candidate",
    header: "Candidat",
    cell: ({ row }) => {
      const candidate = row.original.candidate
      return (
        <div className="flex items-center gap-2">
          <Avatar name={candidate.display_name} size="sm" />
          <div>
            <p className="text-sm font-medium text-foreground">{candidate.display_name}</p>
            <p className="text-xs text-muted-foreground">{candidate.email}</p>
          </div>
        </div>
      )
    },
  },
  {
    id: "job",
    header: "Offre",
    cell: ({ row }) => (
      <span className="text-sm font-medium text-foreground">{row.original.job.title}</span>
    ),
  },
  {
    accessorKey: "message",
    header: "Message",
    cell: ({ row }) => {
      const message = row.original.message
      const truncated = message.length > 80 ? message.slice(0, 80) + "..." : message
      return (
        <span className="text-sm text-muted-foreground">{truncated || "\u2014"}</span>
      )
    },
  },
  {
    id: "reports",
    header: "Signalements",
    cell: ({ row }) => {
      const count = row.original.pending_report_count
      if (!count) return null
      return (
        <Badge variant="destructive">{count}</Badge>
      )
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
