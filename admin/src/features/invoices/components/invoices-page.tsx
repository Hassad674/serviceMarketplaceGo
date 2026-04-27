import { useState } from "react"
import { ChevronLeft, ChevronRight, FileText } from "lucide-react"

import { PageHeader } from "@/shared/components/layouts/page-header"
import { TableSkeleton } from "@/shared/components/ui/skeleton"
import { EmptyState } from "@/shared/components/ui/empty-state"
import { cn } from "@/shared/lib/utils"
import { useAdminInvoices } from "../hooks/use-admin-invoices"
import {
  EMPTY_ADMIN_INVOICE_FILTERS,
  type AdminInvoiceFilters,
} from "../types"
import { InvoicesFilters } from "./invoices-filters"
import { InvoicesTable } from "./invoices-table"

// InvoicesPage is the admin "Toutes les factures emises" page. It
// holds the filter state, drives the cursor stack, and composes the
// filters bar, table, and pagination buttons. No business logic —
// the page is a thin shell over the feature's hooks + components.
export function InvoicesPage() {
  const [filters, setFilters] = useState<AdminInvoiceFilters>(
    EMPTY_ADMIN_INVOICE_FILTERS,
  )
  // cursorStack holds the cursors used to reach previous pages. We
  // push the cursor that produced the current page when moving forward
  // and pop when moving back. Cursor "" represents the first page.
  const [cursorStack, setCursorStack] = useState<string[]>([])

  const { data, isLoading, isError } = useAdminInvoices(filters)

  function handleFiltersChange(next: AdminInvoiceFilters) {
    setFilters(next)
    setCursorStack([])
  }

  function handleNextPage() {
    if (!data?.next_cursor) return
    setCursorStack((stack) => [...stack, filters.cursor])
    setFilters((f) => ({ ...f, cursor: data.next_cursor ?? "" }))
  }

  function handlePreviousPage() {
    setCursorStack((stack) => {
      if (stack.length === 0) return stack
      const next = stack.slice(0, -1)
      const previousCursor = stack[stack.length - 1] ?? ""
      setFilters((f) => ({ ...f, cursor: previousCursor }))
      return next
    })
  }

  const rows = data?.data ?? []
  const hasMore = data?.has_more ?? false
  const hasPrevious = cursorStack.length > 0

  return (
    <div className="space-y-6">
      <PageHeader
        title="Toutes les factures emises"
        description="Factures et avoirs emis par la marketplace, toutes organisations confondues."
      />

      <InvoicesFilters filters={filters} onChange={handleFiltersChange} />

      {isLoading ? (
        <TableSkeleton rows={8} cols={7} />
      ) : isError ? (
        <div className="rounded-xl border border-destructive/20 bg-destructive/5 p-6 text-center text-sm text-destructive">
          Erreur lors du chargement des factures
        </div>
      ) : rows.length === 0 && !hasPrevious ? (
        <EmptyState
          icon={FileText}
          title="Aucune facture"
          description="Aucune facture ne correspond aux filtres actuels."
        />
      ) : (
        <>
          <InvoicesTable rows={rows} />
          <div className="flex items-center justify-between px-1">
            <div className="text-sm text-muted-foreground">
              {rows.length} resultat{rows.length > 1 ? "s" : ""}
              {hasMore ? " (plus de resultats disponibles)" : ""}
            </div>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={handlePreviousPage}
                disabled={!hasPrevious}
                className={cn(
                  "inline-flex items-center gap-1 rounded-lg px-3 py-1.5 text-sm font-medium transition-all duration-200 ease-out",
                  !hasPrevious
                    ? "cursor-not-allowed text-muted-foreground/50"
                    : "text-muted-foreground hover:bg-muted",
                )}
              >
                <ChevronLeft className="h-4 w-4" />
                Precedent
              </button>
              <button
                type="button"
                onClick={handleNextPage}
                disabled={!hasMore}
                className={cn(
                  "inline-flex items-center gap-1 rounded-lg px-3 py-1.5 text-sm font-medium transition-all duration-200 ease-out",
                  !hasMore
                    ? "cursor-not-allowed text-muted-foreground/50"
                    : "text-muted-foreground hover:bg-muted",
                )}
              >
                Suivant
                <ChevronRight className="h-4 w-4" />
              </button>
            </div>
          </div>
        </>
      )}
    </div>
  )
}
