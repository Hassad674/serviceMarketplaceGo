import { Select } from "@/shared/components/ui/select"
import type { ModerationFilters } from "../types"

const sourceOptions = [
  { value: "human_report", label: "Signalements" },
  { value: "auto_media", label: "D\u00E9tection m\u00E9dia" },
  { value: "auto_text", label: "D\u00E9tection texte" },
]

const typeOptions = [
  { value: "report", label: "Signalements" },
  { value: "message", label: "Messages" },
  { value: "review", label: "Avis" },
  { value: "media", label: "M\u00E9dias" },
]

const statusOptions = [
  { value: "pending", label: "En attente" },
  { value: "resolved", label: "R\u00E9solu" },
  { value: "dismissed", label: "Rejet\u00E9" },
  { value: "hidden", label: "Masqu\u00E9" },
  { value: "approved", label: "Approuv\u00E9" },
]

const sortOptions = [
  { value: "newest", label: "Plus r\u00E9cents" },
  { value: "oldest", label: "Plus anciens" },
  { value: "score", label: "Score le plus \u00E9lev\u00E9" },
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
