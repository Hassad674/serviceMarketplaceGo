import type { ColumnDef } from "@tanstack/react-table"
import { RoleBadge, Badge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { formatRelativeDate } from "@/shared/lib/utils"
import type { AdminConversation } from "../types"

function truncate(text: string, maxLen: number): string {
  if (text.length <= maxLen) return text
  return text.slice(0, maxLen) + "..."
}

export const conversationsColumns: ColumnDef<AdminConversation, unknown>[] = [
  {
    id: "participants",
    header: "Participants",
    cell: ({ row }) => {
      const participants = row.original.participants
      return (
        <div className="flex items-center gap-3">
          <div className="flex -space-x-2">
            {participants.slice(0, 2).map((p) => (
              <Avatar key={p.id} name={p.display_name} size="sm" />
            ))}
          </div>
          <div className="min-w-0">
            <p className="truncate text-sm font-medium text-foreground">
              {participants.map((p) => p.display_name).join(", ")}
            </p>
            <p className="truncate text-xs text-muted-foreground">
              {participants.map((p) => p.email).join(", ")}
            </p>
          </div>
        </div>
      )
    },
  },
  {
    id: "roles",
    header: "Roles",
    cell: ({ row }) => (
      <div className="flex flex-wrap gap-1">
        {row.original.participants.map((p) => (
          <RoleBadge key={p.id} role={p.role} />
        ))}
      </div>
    ),
  },
  {
    id: "last_message",
    header: "Dernier message",
    cell: ({ row }) => {
      const msg = row.original.last_message
      const reported = row.original.reported_message
      return (
        <div className="space-y-1">
          {msg ? (
            <span className="text-sm text-muted-foreground">
              {truncate(msg, 60)}
            </span>
          ) : (
            <span className="text-muted-foreground">Aucun message</span>
          )}
          {reported && (
            <p className="text-xs font-medium text-destructive">
              Message signale : {truncate(reported, 50)}
            </p>
          )}
        </div>
      )
    },
  },
  {
    accessorKey: "message_count",
    header: "Messages",
    cell: ({ row }) => (
      <span className="font-mono text-sm text-foreground">
        {row.original.message_count}
      </span>
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
    id: "activity",
    header: "Activite",
    cell: ({ row }) => {
      const lastAt = row.original.last_message_at
      if (!lastAt) {
        return <span className="text-muted-foreground">-</span>
      }
      return (
        <span className="text-sm text-muted-foreground">
          {formatRelativeDate(lastAt)}
        </span>
      )
    },
  },
]
