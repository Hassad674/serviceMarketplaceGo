import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Briefcase, Trash2 } from "lucide-react"
import { DataTable } from "@/shared/components/data-table/data-table"
import { DataTableToolbar } from "@/shared/components/data-table/data-table-toolbar"
import { DataTablePagination } from "@/shared/components/data-table/data-table-pagination"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { Select } from "@/shared/components/ui/select"
import { Button } from "@/shared/components/ui/button"
import { Dialog, DialogTitle, DialogDescription, DialogFooter } from "@/shared/components/ui/dialog"
import { useAdminJobs, useDeleteJob } from "../hooks/use-jobs"
import { jobsColumns } from "./jobs-columns"
import type { JobFilters, AdminJob } from "../types"

const statusOptions = [
  { value: "open", label: "Ouvert" },
  { value: "closed", label: "Ferme" },
]

const sortOptions = [
  { value: "newest", label: "Plus recent" },
  { value: "oldest", label: "Plus ancien" },
  { value: "budget", label: "Budget" },
]

export function JobsTab() {
  const navigate = useNavigate()
  const [filters, setFilters] = useState<JobFilters>({
    status: "",
    search: "",
    sort: "",
    filter: "",
    page: 1,
  })
  const [deleteTarget, setDeleteTarget] = useState<AdminJob | null>(null)

  const { data, isLoading, error } = useAdminJobs(filters)
  const deleteMutation = useDeleteJob()

  const jobs = data?.data ?? []
  const total = data?.total ?? 0
  const totalPages = data?.total_pages ?? 0

  function handleRowClick(job: AdminJob) {
    navigate(`/jobs/${job.id}`)
  }

  return (
    <div className="space-y-4">
      <DataTableToolbar
        searchValue={filters.search}
        onSearchChange={(search) => setFilters((f) => ({ ...f, search, page: 1 }))}
        searchPlaceholder="Rechercher par titre..."
      >
        <Select
          options={statusOptions}
          placeholder="Tous les statuts"
          value={filters.status}
          onChange={(e) => setFilters((f) => ({ ...f, status: e.target.value, page: 1 }))}
          className="w-36"
        />
        <Select
          options={sortOptions}
          placeholder="Tri"
          value={filters.sort}
          onChange={(e) => setFilters((f) => ({ ...f, sort: e.target.value, page: 1 }))}
          className="w-36"
        />
        <FilterCheckbox
          label="Offres signalees"
          checked={filters.filter === "reported"}
          onChange={(checked) =>
            setFilters((f) => ({ ...f, filter: checked ? "reported" : "", page: 1 }))
          }
        />
      </DataTableToolbar>

      {isLoading && <TableSkeleton rows={8} cols={6} />}

      {error && (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Erreur lors du chargement des offres
        </div>
      )}

      {!isLoading && !error && jobs.length === 0 && (
        <EmptyState
          icon={Briefcase}
          title="Aucune offre"
          description="Aucune offre ne correspond aux filtres selectionnes."
        />
      )}

      {!isLoading && !error && jobs.length > 0 && (
        <>
          <DataTable columns={jobsColumns} data={jobs} onRowClick={handleRowClick} />
          <DataTablePagination
            currentPage={filters.page}
            totalPages={totalPages}
            totalCount={total}
            onPageChange={(p) => setFilters((f) => ({ ...f, page: p }))}
          />
        </>
      )}

      <DeleteJobDialog
        job={deleteTarget}
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

function DeleteJobDialog({ job, onClose, onConfirm, isPending }: {
  job: AdminJob | null
  onClose: () => void
  onConfirm: () => void
  isPending: boolean
}) {
  return (
    <Dialog open={!!job} onClose={onClose}>
      <DialogTitle>Supprimer l&apos;offre</DialogTitle>
      <DialogDescription>
        Etes-vous sur de vouloir supprimer l&apos;offre &laquo; {job?.title} &raquo; ?
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
