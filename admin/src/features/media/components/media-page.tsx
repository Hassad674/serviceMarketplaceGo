import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { PageHeader } from "@/shared/components/layouts/page-header"
import { DataTable } from "@/shared/components/data-table/data-table"
import { DataTableToolbar } from "@/shared/components/data-table/data-table-toolbar"
import { DataTablePagination } from "@/shared/components/data-table/data-table-pagination"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { Select } from "@/shared/components/ui/select"
import { mediaColumns } from "./media-columns"
import { useAdminMedia } from "../hooks/use-media"
import type { MediaFilters, AdminMedia } from "../types"

const statusOptions = [
  { value: "pending", label: "En attente" },
  { value: "approved", label: "Approuv\u00E9" },
  { value: "flagged", label: "Signal\u00E9" },
  { value: "rejected", label: "Rejet\u00E9" },
]

const typeOptions = [
  { value: "image/", label: "Images" },
  { value: "video/", label: "Vid\u00E9os" },
  { value: "application/", label: "Documents" },
]

const contextOptions = [
  { value: "profile_photo", label: "Photo de profil" },
  { value: "profile_video", label: "Vid\u00E9o de profil" },
  { value: "message_attachment", label: "Pi\u00E8ce jointe" },
  { value: "review_video", label: "Vid\u00E9o d'avis" },
  { value: "job_video", label: "Vid\u00E9o d'offre" },
  { value: "referrer_video", label: "Vid\u00E9o parrain" },
  { value: "identity_document", label: "Pi\u00E8ce d'identit\u00E9" },
]

export function MediaPage() {
  const navigate = useNavigate()
  const [filters, setFilters] = useState<MediaFilters>({
    status: "",
    type: "",
    context: "",
    search: "",
    page: 1,
  })

  const { data, isLoading, error } = useAdminMedia(filters)

  function updateFilter(key: keyof MediaFilters, value: string | number) {
    setFilters((prev) => ({
      ...prev,
      [key]: value,
      page: key === "page" ? Number(value) : 1,
    }))
  }

  function handleRowClick(media: AdminMedia) {
    navigate(`/media/${media.id}`)
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="M\u00E9dias"
        description="Mod\u00E9ration des fichiers t\u00E9l\u00E9vers\u00E9s par les utilisateurs"
      />

      <DataTableToolbar
        searchValue={filters.search}
        onSearchChange={(v) => updateFilter("search", v)}
        searchPlaceholder="Rechercher un fichier ou utilisateur..."
      >
        <Select
          options={statusOptions}
          placeholder="Statut"
          value={filters.status}
          onChange={(e) => updateFilter("status", e.target.value)}
          className="w-[160px]"
        />
        <Select
          options={typeOptions}
          placeholder="Type"
          value={filters.type}
          onChange={(e) => updateFilter("type", e.target.value)}
          className="w-[140px]"
        />
        <Select
          options={contextOptions}
          placeholder="Contexte"
          value={filters.context}
          onChange={(e) => updateFilter("context", e.target.value)}
          className="w-[180px]"
        />
      </DataTableToolbar>

      {isLoading && <TableSkeleton rows={8} cols={7} />}

      {error && (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Erreur lors du chargement des m{"\u00E9"}dias
        </div>
      )}

      {!isLoading && !error && data && (
        <>
          <DataTable
            columns={mediaColumns}
            data={data.data}
            onRowClick={handleRowClick}
          />
          {data.total_pages > 0 && (
            <DataTablePagination
              currentPage={filters.page || 1}
              totalPages={data.total_pages}
              totalCount={data.total}
              onPageChange={(p) => updateFilter("page", p)}
            />
          )}
        </>
      )}
    </div>
  )
}
