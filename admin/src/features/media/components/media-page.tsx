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
  { value: "approved", label: "Approuvé" },
  { value: "flagged", label: "Signalé" },
  { value: "rejected", label: "Rejeté" },
]

const typeOptions = [
  { value: "image/", label: "Images" },
  { value: "video/", label: "Vidéos" },
  { value: "application/", label: "Documents" },
]

const contextOptions = [
  { value: "profile_photo", label: "Photo de profil" },
  { value: "profile_video", label: "Vidéo de profil" },
  { value: "message_attachment", label: "Pièce jointe" },
  { value: "review_video", label: "Vidéo d'avis" },
  { value: "job_video", label: "Vidéo d'offre" },
  { value: "referrer_video", label: "Vidéo parrain" },
  { value: "identity_document", label: "Pièce d'identité" },
]

const sortOptions = [
  { value: "newest", label: "Plus récents" },
  { value: "oldest", label: "Plus anciens" },
  { value: "score", label: "Score le plus élevé" },
]

export function MediaPage() {
  const navigate = useNavigate()
  const [filters, setFilters] = useState<MediaFilters>({
    status: "",
    type: "",
    context: "",
    search: "",
    sort: "newest",
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
        title="Médias"
        description="Modération des fichiers téléversés par les utilisateurs"
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
        <Select
          options={sortOptions}
          placeholder="Tri"
          value={filters.sort}
          onChange={(e) => updateFilter("sort", e.target.value)}
          className="w-[180px]"
        />
      </DataTableToolbar>

      {isLoading && <TableSkeleton rows={8} cols={7} />}

      {error && (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Erreur lors du chargement des m{"é"}dias
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
