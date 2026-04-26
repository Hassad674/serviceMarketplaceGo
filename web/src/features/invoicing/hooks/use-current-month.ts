"use client"

import { useQuery } from "@tanstack/react-query"
import { fetchCurrentMonthAggregate } from "../api/invoicing-api"
import type { CurrentMonthAggregate } from "../types"
import { invoicingQueryKey } from "./keys"

/**
 * Live "current month" aggregate — running fee total since the
 * start of the calendar month. The backend recomputes on every
 * call, so the cache is intentionally short.
 */
export function useCurrentMonth() {
  return useQuery<CurrentMonthAggregate>({
    queryKey: invoicingQueryKey.currentMonth(),
    queryFn: fetchCurrentMonthAggregate,
    staleTime: 30_000,
    retry: 1,
  })
}
