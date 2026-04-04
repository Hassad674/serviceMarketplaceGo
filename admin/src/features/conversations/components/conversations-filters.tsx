import { Select } from "@/shared/components/ui/select"

const SORT_OPTIONS = [
  { value: "recent", label: "Plus recentes" },
  { value: "oldest", label: "Plus anciennes" },
  { value: "most_messages", label: "Plus de messages" },
]

const FILTER_OPTIONS = [
  { value: "all", label: "Toutes" },
  { value: "reported", label: "Signalees" },
]

type ConversationsFiltersProps = {
  sort: string
  filter: string
  onSortChange: (sort: string) => void
  onFilterChange: (filter: string) => void
}

export function ConversationsFilters({
  sort,
  filter,
  onSortChange,
  onFilterChange,
}: ConversationsFiltersProps) {
  return (
    <div className="flex items-center gap-3">
      <Select
        options={SORT_OPTIONS}
        value={sort}
        onChange={(e) => onSortChange(e.target.value)}
        className="w-48"
      />
      <Select
        options={FILTER_OPTIONS}
        value={filter}
        onChange={(e) => onFilterChange(e.target.value)}
        className="w-40"
      />
    </div>
  )
}
