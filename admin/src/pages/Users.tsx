import { useState, useCallback } from "react"
import { useNavigate } from "react-router-dom"
import { useQuery } from "@tanstack/react-query"
import { adminApi } from "@/lib/api-client.ts"
import { formatDate } from "@/lib/utils.ts"
import { Search, ChevronLeft, ChevronRight, Users } from "lucide-react"

interface AdminUser {
  id: string
  email: string
  first_name: string
  last_name: string
  display_name: string
  role: string
  referrer_enabled: boolean
  is_admin: boolean
  status: string
  email_verified: boolean
  created_at: string
}

interface UsersResponse {
  data: AdminUser[]
  next_cursor: string
  has_more: boolean
  total: number
}

const roleLabels: Record<string, string> = {
  agency: "Prestataire",
  enterprise: "Entreprise",
  provider: "Freelance",
}

const roleBadgeColors: Record<string, string> = {
  agency: "bg-blue-100 text-blue-700",
  enterprise: "bg-purple-100 text-purple-700",
  provider: "bg-rose-100 text-rose-700",
}

const statusLabels: Record<string, string> = {
  active: "Actif",
  suspended: "Suspendu",
  banned: "Banni",
}

const statusBadgeColors: Record<string, string> = {
  active: "bg-green-100 text-green-700",
  suspended: "bg-amber-100 text-amber-700",
  banned: "bg-red-100 text-red-700",
}

const roleFilterOptions = [
  { value: "", label: "Tous" },
  { value: "agency", label: "Prestataire" },
  { value: "enterprise", label: "Entreprise" },
  { value: "provider", label: "Freelance" },
]

const statusFilterOptions = [
  { value: "", label: "Tous" },
  { value: "active", label: "Actif" },
  { value: "suspended", label: "Suspendu" },
  { value: "banned", label: "Banni" },
]

function UserAvatar({ user }: { user: AdminUser }) {
  const initials = (user.first_name?.[0] ?? "") + (user.last_name?.[0] ?? "")
  return (
    <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-rose-100 text-sm font-semibold text-rose-600">
      {initials.toUpperCase() || "?"}
    </div>
  )
}

export default function UsersPage() {
  const navigate = useNavigate()
  const [search, setSearch] = useState("")
  const [roleFilter, setRoleFilter] = useState("")
  const [statusFilter, setStatusFilter] = useState("")
  const [cursorStack, setCursorStack] = useState<string[]>([])
  const [currentCursor, setCurrentCursor] = useState("")

  const buildQueryString = useCallback(() => {
    const params = new URLSearchParams()
    params.set("limit", "20")
    if (currentCursor) params.set("cursor", currentCursor)
    if (search) params.set("search", search)
    if (roleFilter) params.set("role", roleFilter)
    if (statusFilter) params.set("status", statusFilter)
    return params.toString()
  }, [currentCursor, search, roleFilter, statusFilter])

  const { data, isLoading, isError } = useQuery<UsersResponse>({
    queryKey: ["admin-users", currentCursor, search, roleFilter, statusFilter],
    queryFn: () => adminApi<UsersResponse>(`/api/v1/admin/users?${buildQueryString()}`),
  })

  const handleFilterChange = useCallback(
    (setter: (v: string) => void) => (value: string) => {
      setter(value)
      setCursorStack([])
      setCurrentCursor("")
    },
    [],
  )

  const handleSearch = useCallback(
    (value: string) => {
      setSearch(value)
      setCursorStack([])
      setCurrentCursor("")
    },
    [],
  )

  const handleNextPage = useCallback(() => {
    if (data?.next_cursor) {
      setCursorStack((prev) => [...prev, currentCursor])
      setCurrentCursor(data.next_cursor)
    }
  }, [data?.next_cursor, currentCursor])

  const handlePrevPage = useCallback(() => {
    setCursorStack((prev) => {
      const copy = [...prev]
      const previousCursor = copy.pop() ?? ""
      setCurrentCursor(previousCursor)
      return copy
    })
  }, [])

  const users = data?.data ?? []
  const hasMore = data?.has_more ?? false
  const total = data?.total ?? 0
  const hasPrevious = cursorStack.length > 0

  return (
    <div>
      {/* Header */}
      <div className="mb-6 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-rose-100">
            <Users className="h-5 w-5 text-rose-600" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-foreground">Utilisateurs</h1>
            <p className="text-sm text-muted-foreground">
              {total} utilisateur{total !== 1 ? "s" : ""} au total
            </p>
          </div>
        </div>
      </div>

      {/* Filters */}
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            placeholder="Rechercher par nom ou email..."
            value={search}
            onChange={(e) => handleSearch(e.target.value)}
            className="h-10 w-full rounded-lg border border-border bg-card pl-9 pr-4 text-sm shadow-xs outline-none transition-colors focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10"
          />
        </div>
        <select
          value={roleFilter}
          onChange={(e) => handleFilterChange(setRoleFilter)(e.target.value)}
          className="h-10 rounded-lg border border-border bg-card px-3 text-sm shadow-xs outline-none transition-colors focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10"
        >
          {roleFilterOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>
              Role: {opt.label}
            </option>
          ))}
        </select>
        <select
          value={statusFilter}
          onChange={(e) => handleFilterChange(setStatusFilter)(e.target.value)}
          className="h-10 rounded-lg border border-border bg-card px-3 text-sm shadow-xs outline-none transition-colors focus:border-rose-500 focus:ring-4 focus:ring-rose-500/10"
        >
          {statusFilterOptions.map((opt) => (
            <option key={opt.value} value={opt.value}>
              Statut: {opt.label}
            </option>
          ))}
        </select>
      </div>

      {/* Table */}
      <div className="overflow-hidden rounded-xl border border-border bg-card shadow-sm">
        {isLoading ? (
          <TableSkeleton />
        ) : isError ? (
          <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
            <p className="text-sm">Erreur lors du chargement des utilisateurs.</p>
          </div>
        ) : users.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-muted-foreground">
            <Users className="mb-3 h-10 w-10 opacity-40" />
            <p className="text-sm">Aucun utilisateur trouv&eacute;.</p>
          </div>
        ) : (
          <table className="w-full text-left text-sm">
            <thead className="border-b border-border bg-muted/50">
              <tr>
                <th className="px-6 py-3 font-medium text-muted-foreground">Utilisateur</th>
                <th className="px-6 py-3 font-medium text-muted-foreground">R&ocirc;le</th>
                <th className="px-6 py-3 font-medium text-muted-foreground">Statut</th>
                <th className="px-6 py-3 font-medium text-muted-foreground">Email v&eacute;rifi&eacute;</th>
                <th className="px-6 py-3 font-medium text-muted-foreground">Date d'inscription</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {users.map((u) => (
                <tr
                  key={u.id}
                  onClick={() => navigate(`/users/${u.id}`)}
                  className="cursor-pointer transition-colors hover:bg-muted/30"
                >
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-3">
                      <UserAvatar user={u} />
                      <div>
                        <p className="font-medium text-card-foreground">
                          {u.first_name} {u.last_name}
                        </p>
                        <p className="text-xs text-muted-foreground">{u.email}</p>
                      </div>
                    </div>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${roleBadgeColors[u.role] ?? "bg-slate-100 text-slate-700"}`}
                    >
                      {roleLabels[u.role] ?? u.role}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    <span
                      className={`inline-flex rounded-full px-2.5 py-0.5 text-xs font-medium ${statusBadgeColors[u.status] ?? "bg-slate-100 text-slate-700"}`}
                    >
                      {statusLabels[u.status] ?? u.status}
                    </span>
                  </td>
                  <td className="px-6 py-4">
                    {u.email_verified ? (
                      <span className="text-green-600">Oui</span>
                    ) : (
                      <span className="text-muted-foreground">Non</span>
                    )}
                  </td>
                  <td className="px-6 py-4 text-muted-foreground">{formatDate(u.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}

        {/* Pagination */}
        {!isLoading && !isError && users.length > 0 && (
          <div className="flex items-center justify-between border-t border-border px-6 py-3">
            <p className="text-xs text-muted-foreground">
              Page {cursorStack.length + 1}
            </p>
            <div className="flex items-center gap-2">
              <button
                onClick={handlePrevPage}
                disabled={!hasPrevious}
                className="inline-flex h-8 items-center gap-1 rounded-lg border border-border px-3 text-xs font-medium transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-40"
              >
                <ChevronLeft className="h-3.5 w-3.5" />
                Pr&eacute;c&eacute;dent
              </button>
              <button
                onClick={handleNextPage}
                disabled={!hasMore}
                className="inline-flex h-8 items-center gap-1 rounded-lg border border-border px-3 text-xs font-medium transition-colors hover:bg-muted disabled:cursor-not-allowed disabled:opacity-40"
              >
                Suivant
                <ChevronRight className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}

function TableSkeleton() {
  return (
    <div className="space-y-0 divide-y divide-border">
      {Array.from({ length: 8 }).map((_, i) => (
        <div key={i} className="flex items-center gap-4 px-6 py-4">
          <div className="h-9 w-9 animate-pulse rounded-full bg-muted" />
          <div className="flex-1 space-y-2">
            <div className="h-4 w-40 animate-pulse rounded bg-muted" />
            <div className="h-3 w-56 animate-pulse rounded bg-muted" />
          </div>
          <div className="h-5 w-20 animate-pulse rounded-full bg-muted" />
          <div className="h-5 w-16 animate-pulse rounded-full bg-muted" />
          <div className="h-4 w-12 animate-pulse rounded bg-muted" />
          <div className="h-4 w-28 animate-pulse rounded bg-muted" />
        </div>
      ))}
    </div>
  )
}
