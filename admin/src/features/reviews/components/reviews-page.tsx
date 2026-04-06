import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Star } from "lucide-react"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { DataTable } from "@/shared/components/data-table/data-table"
import { DataTableToolbar } from "@/shared/components/data-table/data-table-toolbar"
import { DataTablePagination } from "@/shared/components/data-table/data-table-pagination"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { Select } from "@/shared/components/ui/select"
import { useAdminReviews } from "../hooks/use-reviews"
import { reviewsColumns } from "./reviews-columns"
import type { ReviewFilters, AdminReview } from "../types"

const ratingOptions = [
  { value: "1", label: "1 etoile" },
  { value: "2", label: "2 etoiles" },
  { value: "3", label: "3 etoiles" },
  { value: "4", label: "4 etoiles" },
  { value: "5", label: "5 etoiles" },
]

const sortOptions = [
  { value: "newest", label: "Plus recent" },
  { value: "oldest", label: "Plus ancien" },
  { value: "rating_high", label: "Note (haute)" },
  { value: "rating_low", label: "Note (basse)" },
]

export function ReviewsPage() {
  const navigate = useNavigate()
  const [filters, setFilters] = useState<ReviewFilters>({
    search: "",
    rating: "",
    sort: "",
    filter: "",
    page: 1,
  })

  const { data, isLoading, error } = useAdminReviews(filters)

  const reviews = data?.data ?? []
  const total = data?.total ?? 0
  const totalPages = data?.total_pages ?? 0

  function handleRowClick(review: AdminReview) {
    navigate(`/reviews/${review.id}`)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Avis"
        description="Gestion des avis et evaluations de la marketplace"
      />

      <DataTableToolbar
        searchValue={filters.search}
        onSearchChange={(search) => setFilters((f) => ({ ...f, search, page: 1 }))}
        searchPlaceholder="Rechercher par utilisateur..."
      >
        <Select
          options={ratingOptions}
          placeholder="Toutes les notes"
          value={filters.rating}
          onChange={(e) => setFilters((f) => ({ ...f, rating: e.target.value, page: 1 }))}
          className="w-36"
        />
        <Select
          options={sortOptions}
          placeholder="Tri"
          value={filters.sort}
          onChange={(e) => setFilters((f) => ({ ...f, sort: e.target.value, page: 1 }))}
          className="w-40"
        />
        <FilterCheckbox
          label="Avis signales"
          checked={filters.filter === "reported"}
          onChange={(checked) =>
            setFilters((f) => ({ ...f, filter: checked ? "reported" : "", page: 1 }))
          }
        />
      </DataTableToolbar>

      {isLoading && <TableSkeleton rows={8} cols={7} />}

      {error && (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Erreur lors du chargement des avis
        </div>
      )}

      {!isLoading && !error && reviews.length === 0 && (
        <EmptyState
          icon={Star}
          title="Aucun avis"
          description="Aucun avis ne correspond aux filtres selectionnes."
        />
      )}

      {!isLoading && !error && reviews.length > 0 && (
        <>
          <DataTable columns={reviewsColumns} data={reviews} onRowClick={handleRowClick} />
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

function FilterCheckbox({ label, checked, onChange }: {
  label: string
  checked: boolean
  onChange: (checked: boolean) => void
}) {
  return (
    <label className="flex cursor-pointer items-center gap-2 rounded-lg border border-border bg-white px-3 py-2 text-sm transition-all duration-200 ease-out select-none hover:border-rose-200 hover:bg-rose-50/50">
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="h-4 w-4 rounded border-gray-300 text-rose-500 focus:ring-2 focus:ring-rose-500/20"
      />
      <span className="text-foreground">{label}</span>
    </label>
  )
}
