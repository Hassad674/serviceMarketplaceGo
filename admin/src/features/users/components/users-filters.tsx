import { DataTableToolbar } from "@/shared/components/data-table/data-table-toolbar"
import { Select } from "@/shared/components/ui/select"
import type { UserFilters } from "../types"

const roleOptions = [
  { value: "agency", label: "Prestataire" },
  { value: "enterprise", label: "Entreprise" },
  { value: "provider", label: "Freelance" },
]

const statusOptions = [
  { value: "active", label: "Actif" },
  { value: "suspended", label: "Suspendu" },
  { value: "banned", label: "Banni" },
]

type UsersFiltersProps = {
  filters: UserFilters
  onChange: (filters: UserFilters) => void
}

export function UsersFilters({ filters, onChange }: UsersFiltersProps) {
  return (
    <DataTableToolbar
      searchValue={filters.search}
      onSearchChange={(search) => onChange({ ...filters, search, cursor: "" })}
      searchPlaceholder="Rechercher par nom ou email..."
    >
      <Select
        options={roleOptions}
        placeholder="Tous les roles"
        value={filters.role}
        onChange={(e) => onChange({ ...filters, role: e.target.value, cursor: "" })}
        className="w-40"
      />
      <Select
        options={statusOptions}
        placeholder="Tous les statuts"
        value={filters.status}
        onChange={(e) => onChange({ ...filters, status: e.target.value, cursor: "" })}
        className="w-40"
      />
      <label className="flex cursor-pointer items-center gap-2 rounded-lg border border-border bg-white px-3 py-2 text-sm transition-all duration-200 ease-out select-none hover:border-rose-200 hover:bg-rose-50/50">
        <input
          type="checkbox"
          checked={filters.reported}
          onChange={(e) => onChange({ ...filters, reported: e.target.checked, cursor: "" })}
          className="h-4 w-4 rounded border-gray-300 text-rose-500 focus:ring-2 focus:ring-rose-500/20"
        />
        <span className="text-foreground">Signales uniquement</span>
      </label>
    </DataTableToolbar>
  )
}
