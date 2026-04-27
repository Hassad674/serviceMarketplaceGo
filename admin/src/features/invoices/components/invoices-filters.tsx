import { Search } from "lucide-react"
import { Select } from "@/shared/components/ui/select"
import type {
  AdminInvoiceFilters,
  AdminInvoiceTypeFilter,
} from "../types"

type InvoicesFiltersProps = {
  filters: AdminInvoiceFilters
  // onChange replaces the whole filter struct except the cursor (which
  // is reset to "" automatically when any filter mutates so the user
  // doesn't accidentally hold a stale cursor against new criteria).
  onChange: (next: AdminInvoiceFilters) => void
}

const TYPE_OPTIONS: { value: AdminInvoiceTypeFilter; label: string }[] = [
  { value: "", label: "Tous les types" },
  { value: "subscription", label: "Abonnement" },
  { value: "monthly_commission", label: "Commission mensuelle" },
  { value: "credit_note", label: "Avoir" },
]

// InvoicesFilters renders the filter bar for the admin "Toutes les
// factures emises" page. Every change resets the cursor so a filter
// flip never lands on a paginated middle of a different result set.
export function InvoicesFilters({ filters, onChange }: InvoicesFiltersProps) {
  function patch<K extends keyof AdminInvoiceFilters>(
    key: K,
    value: AdminInvoiceFilters[K],
  ) {
    onChange({ ...filters, [key]: value, cursor: "" })
  }

  return (
    <div className="space-y-3">
      {/* Top row: search + type */}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-sm">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <input
            type="text"
            value={filters.search}
            onChange={(e) => patch("search", e.target.value)}
            placeholder="Numero ou raison sociale..."
            className="w-full rounded-lg border border-border bg-background py-2 pl-9 pr-3 text-sm transition-all duration-200 ease-out placeholder:text-muted-foreground focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-500/20"
          />
        </div>

        <Select
          options={TYPE_OPTIONS}
          value={filters.status}
          onChange={(e) =>
            patch("status", e.target.value as AdminInvoiceTypeFilter)
          }
          label=""
        />

        <input
          type="text"
          value={filters.recipient_org_id}
          onChange={(e) => patch("recipient_org_id", e.target.value)}
          placeholder="ID organisation destinataire (UUID)"
          className="w-[280px] rounded-lg border border-border bg-background px-3 py-2 text-sm font-mono transition-all duration-200 ease-out placeholder:text-muted-foreground focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-500/20"
          aria-label="ID organisation destinataire"
        />
      </div>

      {/* Bottom row: date + amount range */}
      <div className="flex flex-wrap items-center gap-3">
        <DateInput
          label="Du"
          value={filters.date_from}
          onChange={(v) => patch("date_from", v)}
        />
        <DateInput
          label="Au"
          value={filters.date_to}
          onChange={(v) => patch("date_to", v)}
        />
        <AmountInput
          label="Min (centimes)"
          value={filters.min_amount_cents}
          onChange={(v) => patch("min_amount_cents", v)}
        />
        <AmountInput
          label="Max (centimes)"
          value={filters.max_amount_cents}
          onChange={(v) => patch("max_amount_cents", v)}
        />
      </div>
    </div>
  )
}

function DateInput({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (next: string) => void
}) {
  // The backend wants RFC3339; the native <input type="date"> emits
  // YYYY-MM-DD. We append the time-and-timezone parts on the way out
  // so the operator sees a familiar date picker.
  const dateValue = value ? value.slice(0, 10) : ""
  return (
    <label className="flex items-center gap-2 text-xs text-muted-foreground">
      <span>{label}</span>
      <input
        type="date"
        value={dateValue}
        onChange={(e) => {
          const raw = e.target.value
          onChange(raw ? `${raw}T00:00:00Z` : "")
        }}
        className="rounded-lg border border-border bg-background px-3 py-2 text-sm transition-all duration-200 ease-out focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-500/20"
      />
    </label>
  )
}

function AmountInput({
  label,
  value,
  onChange,
}: {
  label: string
  value: string
  onChange: (next: string) => void
}) {
  return (
    <label className="flex items-center gap-2 text-xs text-muted-foreground">
      <span>{label}</span>
      <input
        type="number"
        inputMode="numeric"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder="0"
        className="w-32 rounded-lg border border-border bg-background px-3 py-2 text-sm font-mono transition-all duration-200 ease-out placeholder:text-muted-foreground focus:border-rose-500 focus:outline-none focus:ring-2 focus:ring-rose-500/20"
      />
    </label>
  )
}
