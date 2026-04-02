import type { ColumnDef } from "@tanstack/react-table"
import { RoleBadge, StatusBadge } from "@/shared/components/ui/badge"
import { Avatar } from "@/shared/components/ui/avatar"
import { formatDate } from "@/shared/lib/utils"
import type { AdminUser } from "../types"

export const usersColumns: ColumnDef<AdminUser, unknown>[] = [
  {
    id: "user",
    header: "Utilisateur",
    cell: ({ row }) => {
      const u = row.original
      const name = u.display_name || `${u.first_name} ${u.last_name}`
      return (
        <div className="flex items-center gap-3">
          <Avatar name={name} size="sm" />
          <div>
            <p className="font-medium text-foreground">{name}</p>
            <p className="text-xs text-muted-foreground">{u.email}</p>
          </div>
        </div>
      )
    },
  },
  {
    accessorKey: "role",
    header: "Rôle",
    cell: ({ row }) => <RoleBadge role={row.original.role} />,
  },
  {
    accessorKey: "status",
    header: "Statut",
    cell: ({ row }) => <StatusBadge status={row.original.status} />,
  },
  {
    accessorKey: "email_verified",
    header: "Email",
    cell: ({ row }) => (
      <span className={row.original.email_verified ? "text-success" : "text-muted-foreground"}>
        {row.original.email_verified ? "Vérifié" : "Non vérifié"}
      </span>
    ),
  },
  {
    accessorKey: "created_at",
    header: "Inscription",
    cell: ({ row }) => (
      <span className="text-muted-foreground">{formatDate(row.original.created_at)}</span>
    ),
  },
]
