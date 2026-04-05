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
    filter: "",
    page: 1,
  })
  const [deleteTarget, setDeleteTarget] = useState<AdminJobApplication | null>(null)

  const { data, isLoading, error } = useAdminJobApplications(filters)
  const deleteMutation = useDeleteJobApplication()

  const applications = data?.data ?? []
  const total = data?.total ?? 0
  const totalPages = data?.total_pages ?? 0

  return (
    <div className="space-y-4">
      <DataTableToolbar
        searchValue={filters.search}
        onSearchChange={(search) => setFilters((f) => ({ ...f, search, page: 1 }))}
        searchPlaceholder="Rechercher par candidat..."
      >
        <Select
          options={sortOptions}
          placeholder="Tri"
          value={filters.sort}
          onChange={(e) => setFilters((f) => ({ ...f, sort: e.target.value, page: 1 }))}
          className="w-36"
        />
        <FilterCheckbox
          label="Candidatures signalees"
          checked={filters.filter === "reported"}
          onChange={(checked) =>
            setFilters((f) => ({ ...f, filter: checked ? "reported" : "", page: 1 }))
          }
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
            currentPage={filters.page}
            totalPages={totalPages}
            totalCount={total}
            onPageChange={(p) => setFilters((f) => ({ ...f, page: p }))}
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
