"use client"

import { useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { getReceipt, listReceipts } from "../api/receipt-api"
import type { Receipt, ReceiptsPage } from "../types"
import { receiptQueryKey } from "./keys"

/**
 * Cursor-paginated receipts for the authenticated org.
 *
 * V1 keeps the implementation deliberately simple: a single page
 * query plus a `loadMore` trigger that swaps the cursor and
 * triggers a refetch — same pattern as the sibling `useInvoices()`
 * hook so the two tabs feel identical to the user. Switching to
 * `useInfiniteQuery` is a non-breaking refactor for V2.
 */
export function useReceipts() {
  const [cursor, setCursor] = useState<string | null>(null)

  const query = useQuery<ReceiptsPage>({
    queryKey: receiptQueryKey.list(cursor),
    queryFn: () => listReceipts(cursor ?? undefined),
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

/** Single-receipt detail. `enabled` defaults to `Boolean(id)`. */
export function useReceipt(id: string | null) {
  return useQuery<Receipt>({
    queryKey: receiptQueryKey.detail(id ?? "__missing__"),
    queryFn: () => getReceipt(id as string),
    enabled: Boolean(id),
    staleTime: 60_000,
    retry: 1,
  })
}
