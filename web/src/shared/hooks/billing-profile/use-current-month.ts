"use client"

import { useQuery } from "@tanstack/react-query"
import { fetchCurrentMonthAggregate } from "@/shared/lib/billing-profile/billing-profile-api"
import type { CurrentMonthAggregate } from "@/shared/types/billing-profile"
import { sharedInvoicingQueryKey } from "@/shared/lib/query-keys/invoicing"

/**
 * Shared current-month aggregate hook (P9 — `CurrentMonthAggregate`
 * is rendered above the wallet payout block).
 *
 * Live "current month" aggregate — running fee total since the
 * start of the calendar month. The backend recomputes on every
 * call, so the cache is intentionally short.
 */
export function useCurrentMonth() {
  return useQuery<CurrentMonthAggregate>({
    queryKey: sharedInvoicingQueryKey.currentMonth(),
    queryFn: fetchCurrentMonthAggregate,
    staleTime: 30_000,
    retry: 1,
  })
}
