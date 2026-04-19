"use client"

import { useEffect, useState } from "react"
import { useQuery } from "@tanstack/react-query"
import { getFeePreview } from "../api/billing-api"
import type { FeePreview } from "../types"

const DEBOUNCE_MS = 300

/**
 * Debounces the raw amount (typed by the prestataire) before issuing
 * the fee-preview request. Returns a TanStack Query result so the
 * caller can render loading / error / data states with the same
 * patterns used elsewhere in the app.
 *
 * The query is skipped for non-positive amounts: the endpoint would
 * return a zero-fee response, but showing the grid without an active
 * row is more helpful than hitting the network on every keystroke.
 */
export function useFeePreview(amountCents: number) {
  const debouncedAmount = useDebouncedValue(amountCents, DEBOUNCE_MS)
  const enabled = debouncedAmount > 0

  return useQuery<FeePreview>({
    queryKey: ["billing", "fee-preview", debouncedAmount],
    queryFn: () => getFeePreview(debouncedAmount),
    enabled,
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
