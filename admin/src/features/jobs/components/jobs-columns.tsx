import type { ColumnDef } from "@tanstack/react-table"
import { Badge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { formatDate, formatCurrency } from "@/shared/lib/utils"
import type { AdminJob } from "../types"

const statusLabels: Record<string, string> = {
  open: "Ouvert",
  closed: "Ferme",
}

const statusVariants: Record<string, "success" | "outline"> = {
  open: "success",
  closed: "outline",
}

export const jobsColumns: ColumnDef<AdminJob, unknown>[] = [
  {
    accessorKey: "title",
    header: "Titre",
    cell: ({ row }) => (
      <span className="font-medium text-foreground">{row.original.title}</span>
    ),
  },
  {
    id: "author",
    header: "Auteur",
    cell: ({ row }) => {
      const author = row.original.author
      return (
        <div className="flex items-center gap-2">
          <Avatar name={author.display_name} size="sm" />
          <div>
            <p className="text-sm font-medium text-foreground">{author.display_name}</p>
            <p className="text-xs text-muted-foreground">{author.email}</p>
          </div>
        </div>
      )
    },
  },
  {
    accessorKey: "status",
    header: "Statut",
    cell: ({ row }) => (
      <Badge variant={statusVariants[row.original.status] ?? "outline"}>
        {statusLabels[row.original.status] ?? row.original.status}
      </Badge>
    ),
  },
  {
    id: "budget",
    header: "Budget",
    cell: ({ row }) => (
      <span className="text-sm text-foreground">
        {formatCurrency(row.original.min_budget)} - {formatCurrency(row.original.max_budget)}
      </span>
    ),
  },
  {
    accessorKey: "application_count",
    header: "Candidatures",
    cell: ({ row }) => (
      <span className="text-sm text-muted-foreground">{row.original.application_count}</span>
    ),
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
