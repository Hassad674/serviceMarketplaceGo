"use client"

import { useEffect, useState } from "react"

// Tiny debounce helper kept local to the skill feature so we don't
// pull in a cross-feature utility. Returns the latest `value` after
// `delayMs` of quiescence — callers pass the debounced value into
// query keys or fetchers so network traffic is rate-limited without
// affecting the input's immediate visual state.
export function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    const handle = window.setTimeout(() => setDebounced(value), delayMs)
    return () => window.clearTimeout(handle)
  }, [value, delayMs])

  return debounced
}
