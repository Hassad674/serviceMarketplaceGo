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
        placeholder="Tous les rôles"
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
    </DataTableToolbar>
  )
}
