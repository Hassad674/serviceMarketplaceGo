import { useState } from "react"
import { Shield } from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { DataTablePagination } from "@/shared/components/data-table/data-table-pagination"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { Badge } from "@/shared/components/ui/badge"
import { Skeleton } from "@/shared/components/ui/skeleton"
import { useModerationItems } from "../hooks/use-moderation"
import { useModerationCount } from "@/shared/hooks/use-moderation-count"
import { ModerationFiltersBar } from "./moderation-filters"
import { ModerationCard } from "./moderation-card"
import type { ModerationFilters } from "../types"

export function ModerationPage() {
  const [filters, setFilters] = useState<ModerationFilters>({
    source: "",
    type: "",
    status: "",
    sort: "newest",
    page: 1,
  })

  const { data, isLoading, error } = useModerationItems(filters)
  const { data: countData } = useModerationCount()

  function updateFilter(key: keyof ModerationFilters, value: string) {
    setFilters((prev) => ({
      ...prev,
      [key]: value,
      page: key === "page" ? Number(value) : 1,
    }))
  }

  const pendingCount = countData?.count ?? 0

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Mod\u00E9ration`}
        description="File unifi\u00E9e de mod\u00E9ration : signalements, d\u00E9tection m\u00E9dia et texte"
        actions={
          pendingCount > 0 ? (
            <Badge variant="destructive" className="text-sm">
              {pendingCount} en attente
            </Badge>
          ) : undefined
        }
      />

      <ModerationFiltersBar
        filters={filters}
        onFilterChange={updateFilter}
      />

      {isLoading && <ModerationSkeleton />}

      {error && (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          {`Erreur lors du chargement des \u00E9l\u00E9ments de mod\u00E9ration`}
        </div>
      )}

      {!isLoading && !error && data && (
        <>
          {data.data.length === 0 ? (
            <EmptyState
              icon={Shield}
              title={`Aucun \u00E9l\u00E9ment`}
              description={`Aucun \u00E9l\u00E9ment de mod\u00E9ration ne correspond \u00E0 vos filtres`}
            />
          ) : (
            <div className="space-y-4">
              {data.data.map((item) => (
                <ModerationCard key={`${item.source}-${item.id}`} item={item} />
              ))}
            </div>
          )}

          {data.total_pages > 1 && (
            <DataTablePagination
              currentPage={filters.page || 1}
              totalPages={data.total_pages}
              totalCount={data.total}
              onPageChange={(p) => updateFilter("page", String(p))}
            />
          )}
        </>
      )}
    </div>
  )
}

function ModerationSkeleton() {
  return (
    <div className="space-y-4">
      {Array.from({ length: 5 }).map((_, i) => (
        <div key={i} className="rounded-xl border border-gray-100 bg-white p-5 shadow-sm">
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Skeleton className="h-5 w-32" />
              <Skeleton className="h-4 w-20" />
            </div>
            <Skeleton className="h-4 w-3/4" />
            <div className="flex items-center justify-between">
              <Skeleton className="h-5 w-40" />
              <div className="flex gap-2">
                <Skeleton className="h-8 w-20" />
                <Skeleton className="h-8 w-20" />
              </div>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
