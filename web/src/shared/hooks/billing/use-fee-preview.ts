"use client"

import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { getFeePreview } from "@/shared/lib/billing/billing-api"
import type { FeePreview } from "@/shared/types/billing"

const DEBOUNCE_MS = 300

/**
 * Shared `useFeePreview` (P9). Debounces the raw amount (typed by the
 * prestataire) before issuing the fee-preview request. Returns a
 * TanStack Query result so the caller can render loading / error / data
 * states with the same patterns used elsewhere in the app.
 *
 * The query fires even at amount === 0 because the response carries
 * `viewer_is_provider` — the consumer uses that flag to hide the whole
 * section for client-side viewers BEFORE they start typing. Skipping
 * the call at 0 would keep the placeholder "Renseignez un montant…"
 * visible to enterprises until they hit a key, leaking the existence
 * of the fee preview to parties who must never see it.
 */
export function useFeePreview(amountCents: number, recipientId?: string) {
  const debouncedAmount = useDebouncedValue(amountCents, DEBOUNCE_MS)
  const safeAmount = debouncedAmount < 0 ? 0 : debouncedAmount
  // recipientId is part of the cache key so two form instances
  // targeting different recipients never share a cached response —
  // in particular, the `viewer_is_provider` flag depends on the
  // pair and must not leak across recipients.
  const recipientKey = recipientId ?? null

  return useQuery<FeePreview>({
    queryKey: ["billing", "fee-preview", safeAmount, recipientKey],
    queryFn: () => getFeePreview(safeAmount, recipientId),
    staleTime: 60_000,
  })
}

function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const handle = setTimeout(() => setDebounced(value), delayMs)
    return () => clearTimeout(handle)
  }, [value, delayMs])

  return debounced
}
