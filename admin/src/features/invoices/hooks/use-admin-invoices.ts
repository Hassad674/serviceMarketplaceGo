import { useQuery } from "@tanstack/react-query"
import { fetchAdminInvoices } from "../api/invoicing-api"
import type { AdminInvoiceFilters } from "../types"

export function adminInvoicesQueryKey(filters: AdminInvoiceFilters) {
  return ["admin", "invoices", filters] as const
}

// useAdminInvoices is the read-only query hook for the admin "Toutes
// les factures emises" page. The filter struct doubles as the query
// key, so a filter change triggers a refetch automatically.
export function useAdminInvoices(filters: AdminInvoiceFilters) {
  return useQuery({
    queryKey: adminInvoicesQueryKey(filters),
    queryFn: () => fetchAdminInvoices(filters),
    staleTime: 30_000,
  })
}
