"use client"

import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { fetchInvoices } from "../api/invoicing-api"
import type { InvoicesPage } from "../types"
import { invoicingQueryKey } from "./keys"

/**
 * Cursor-paginated invoices for the authenticated org.
 *
 * V1 keeps the implementation deliberately simple: a single page
 * query plus a `loadMore` trigger that swaps the cursor and
 * triggers a refetch. Switching to `useInfiniteQuery` is a
 * non-breaking refactor for V2 once the UI needs cross-page
 * scrolling — for now "Voir plus" and a single visible page is
 * what the spec asks for.
 */
export function useInvoices() {
  const [cursor, setCursor] = useState<string | null>(null)

  const query = useQuery<InvoicesPage>({
    queryKey: invoicingQueryKey.invoices(cursor),
    queryFn: () => fetchInvoices(cursor ?? undefined),
    staleTime: 30_000,
    retry: 1,
  })

  function loadMore() {
    const next = query.data?.next_cursor
    if (next) setCursor(next)
  }

  function reset() {
    setCursor(null)
  }

  return {
    ...query,
    cursor,
    loadMore,
    reset,
    hasMore: Boolean(query.data?.next_cursor),
  }
}
