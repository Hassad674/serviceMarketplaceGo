import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Users } from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { DataTable } from "@/shared/components/data-table/data-table"
import { DataTablePagination } from "@/shared/components/data-table/data-table-pagination"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { useUsers } from "../hooks/use-users"
import { UsersFilters } from "./users-filters"
import { usersColumns } from "./users-columns"
import type { UserFilters, AdminUser } from "../types"

export function UsersPage() {
  const navigate = useNavigate()
  const [filters, setFilters] = useState<UserFilters>({
    role: "",
    status: "",
    search: "",
    page: 1,
    reported: false,
  })

  const { data, isLoading, error } = useUsers(filters)

  const users = data?.data ?? []
  const total = data?.total ?? 0
  const totalPages = data?.total_pages ?? 0

  function handleFiltersChange(newFilters: UserFilters) {
    setFilters(newFilters)
  }

  function handleRowClick(user: AdminUser) {
    navigate(`/users/${user.id}`)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Utilisateurs"
        description={total > 0 ? `${total} utilisateur${total > 1 ? "s" : ""} au total` : undefined}
      />

      <UsersFilters filters={filters} onChange={handleFiltersChange} />

      {isLoading && <TableSkeleton rows={8} cols={5} />}

      {error && (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Erreur lors du chargement des utilisateurs
        </div>
      )}

      {!isLoading && !error && users.length === 0 && (
        <EmptyState
          icon={Users}
          title="Aucun utilisateur"
          description="Aucun utilisateur ne correspond aux filtres s{'\u00E9'}lectionn{'\u00E9'}s."
        />
      )}

      {!isLoading && !error && users.length > 0 && (
        <>
          <DataTable columns={usersColumns} data={users} onRowClick={handleRowClick} />
          <DataTablePagination
            currentPage={filters.page}
            totalPages={totalPages}
            totalCount={total}
            onPageChange={(p) => setFilters((f) => ({ ...f, page: p }))}
          />
        </>
      )}
    </div>
  )
}
