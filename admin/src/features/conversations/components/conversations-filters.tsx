import { Select } from "@/shared/components/ui/select"

const SORT_OPTIONS = [
  { value: "recent", label: "Plus recentes" },
  { value: "oldest", label: "Plus anciennes" },
  { value: "most_messages", label: "Plus de messages" },
]

type ConversationsFiltersProps = {
  sort: string
  reportedConversations: boolean
  reportedMessages: boolean
  onSortChange: (sort: string) => void
  onReportedConversationsChange: (checked: boolean) => void
  onReportedMessagesChange: (checked: boolean) => void
}

export function ConversationsFilters({
  sort,
  reportedConversations,
  reportedMessages,
  onSortChange,
  onReportedConversationsChange,
  onReportedMessagesChange,
}: ConversationsFiltersProps) {
  return (
    <div className="flex items-center gap-4">
      <Select
        options={SORT_OPTIONS}
        value={sort}
        onChange={(e) => onSortChange(e.target.value)}
        className="w-48"
      />
      <FilterCheckbox
        label="Conversations signalees"
        checked={reportedConversations}
        onChange={onReportedConversationsChange}
      />
      <FilterCheckbox
        label="Messages signales"
        checked={reportedMessages}
        onChange={onReportedMessagesChange}
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
