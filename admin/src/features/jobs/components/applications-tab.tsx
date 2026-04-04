import { useState } from "react"
import { FileText, Trash2 } from "lucide-react"
import { DataTable } from "@/shared/components/data-table/data-table"
import { DataTableToolbar } from "@/shared/components/data-table/data-table-toolbar"
import { DataTablePagination } from "@/shared/components/data-table/data-table-pagination"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { Select } from "@/shared/components/ui/select"
import { Button } from "@/shared/components/ui/button"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "@/shared/components/ui/dialog"
import { useAdminJobApplications, useDeleteJobApplication } from "../hooks/use-jobs"
import { applicationsColumns } from "./applications-columns"
import type { ApplicationFilters, AdminJobApplication } from "../types"

const sortOptions = [
  { value: "newest", label: "Plus recent" },
  { value: "oldest", label: "Plus ancien" },
]

export function ApplicationsTab() {
  const [filters, setFilters] = useState<ApplicationFilters>({
    job_id: "",
    search: "",
    sort: "",
    cursor: "",
  })
  const [cursors, setCursors] = useState<string[]>([])
  const [deleteTarget, setDeleteTarget] = useState<AdminJobApplication | null>(null)

  const { data, isLoading, error } = useAdminJobApplications(filters)
  const deleteMutation = useDeleteJobApplication()

  const applications = data?.data ?? []
  const hasMore = data?.has_more ?? false
  const total = data?.total ?? 0

  function handleNextPage() {
    if (data?.next_cursor) {
      setCursors((prev) => [...prev, filters.cursor])
      setFilters((prev) => ({ ...prev, cursor: data.next_cursor }))
    }
  }

  function handlePreviousPage() {
    const prev = cursors[cursors.length - 1] ?? ""
    setCursors((c) => c.slice(0, -1))
    setFilters((f) => ({ ...f, cursor: prev }))
  }

  return (
    <div className="space-y-4">
      <DataTableToolbar
        searchValue={filters.search}
        onSearchChange={(search) => setFilters((f) => ({ ...f, search, cursor: "" }))}
        searchPlaceholder="Rechercher par candidat..."
      >
        <Select
          options={sortOptions}
          placeholder="Tri"
          value={filters.sort}
          onChange={(e) => setFilters((f) => ({ ...f, sort: e.target.value, cursor: "" }))}
          className="w-36"
        />
      </DataTableToolbar>

      {isLoading && <TableSkeleton rows={8} cols={4} />}

      {error && (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Erreur lors du chargement des candidatures
        </div>
      )}

      {!isLoading && !error && applications.length === 0 && (
        <EmptyState
          icon={FileText}
          title="Aucune candidature"
          description="Aucune candidature ne correspond aux filtres selectionnes."
        />
      )}

      {!isLoading && !error && applications.length > 0 && (
        <>
          <DataTable columns={applicationsColumns} data={applications} />
          <DataTablePagination
            hasMore={hasMore}
            onNextPage={handleNextPage}
            onPreviousPage={handlePreviousPage}
            hasPreviousPage={cursors.length > 0}
            totalCount={total}
          />
        </>
      )}

      <DeleteApplicationDialog
        application={deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={() => {
          if (deleteTarget) {
            deleteMutation.mutate(deleteTarget.id, {
              onSuccess: () => setDeleteTarget(null),
            })
          }
        }}
        isPending={deleteMutation.isPending}
      />
    </div>
  )
}

function DeleteApplicationDialog({ application, onClose, onConfirm, isPending }: {
  application: AdminJobApplication | null
  onClose: () => void
  onConfirm: () => void
  isPending: boolean
}) {
  return (
    <Dialog open={!!application} onClose={onClose}>
      <DialogTitle>Supprimer la candidature</DialogTitle>
      <DialogDescription>
        Etes-vous sur de vouloir supprimer cette candidature ?
        Cette action est irreversible.
      </DialogDescription>
      <DialogFooter>
        <Button variant="outline" onClick={onClose}>Annuler</Button>
        <Button variant="destructive" onClick={onConfirm} disabled={isPending}>
          <Trash2 className="h-4 w-4" />
          {isPending ? "Suppression..." : "Supprimer"}
        </Button>
      </DialogFooter>
    </Dialog>
  )
}
