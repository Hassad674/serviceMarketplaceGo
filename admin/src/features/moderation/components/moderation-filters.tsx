import { Select } from "@/shared/components/ui/select"
import type { ModerationFilters } from "../types"

const sourceOptions = [
  { value: "human_report", label: "Signalements" },
  { value: "auto_media", label: "Détection média" },
  { value: "auto_text", label: "Détection texte" },
]

const typeOptions = [
  { value: "report", label: "Signalements" },
  { value: "message", label: "Messages" },
  { value: "review", label: "Avis" },
  { value: "media", label: "Médias" },
]

const statusOptions = [
  { value: "pending", label: "En attente" },
  { value: "resolved", label: "Résolu" },
  { value: "dismissed", label: "Rejeté" },
  { value: "hidden", label: "Masqué" },
  { value: "deleted", label: "Supprimé auto" },
  { value: "blocked", label: "Bloqué" },
  { value: "approved", label: "Approuvé" },
]

const sortOptions = [
  { value: "newest", label: "Plus récents" },
  { value: "oldest", label: "Plus anciens" },
  { value: "score", label: "Score le plus élevé" },
]

type ModerationFiltersBarProps = {
  filters: ModerationFilters
  onFilterChange: (key: keyof ModerationFilters, value: string) => void
}

export function ModerationFiltersBar({ filters, onFilterChange }: ModerationFiltersBarProps) {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <Select
        options={sourceOptions}
        placeholder="Source"
        value={filters.source}
        onChange={(e) => onFilterChange("source", e.target.value)}
        className="w-[180px]"
      />
      <Select
        options={typeOptions}
        placeholder="Type"
        value={filters.type}
        onChange={(e) => onFilterChange("type", e.target.value)}
        className="w-[160px]"
      />
      <Select
        options={statusOptions}
        placeholder="Statut"
        value={filters.status}
        onChange={(e) => onFilterChange("status", e.target.value)}
        className="w-[160px]"
      />
      <Select
        options={sortOptions}
        placeholder="Tri"
        value={filters.sort}
        onChange={(e) => onFilterChange("sort", e.target.value)}
        className="w-[180px]"
      />
    </div>
  )
}
