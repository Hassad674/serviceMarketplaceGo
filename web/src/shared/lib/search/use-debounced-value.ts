"use client"

import { useEffect, useState } from "react"

/**
 * useDebouncedValue is a shared debounce helper. It returns the
 * latest `value` after `delayMs` of quiescence — callers pass the
 * debounced value into query keys or fetchers so network traffic is
 * rate-limited without affecting the input's immediate visual state.
 *
 * Lives in `shared/lib/search` because the search page debounces the
 * query field on every keystroke. Using a shared helper prevents
 * features from creating their own drifting copies.
 *
 * Exported at the top level so cross-feature consumers can use it
 * without poking into internal paths. Pass `0` to disable
 * debouncing entirely — useful for tests that want to assert on the
 * immediate value.
 */
export function useDebouncedValue<T>(value: T, delayMs: number): T {
  const [debounced, setDebounced] = useState(value)

  useEffect(() => {
    // delayMs <= 0 disables debouncing entirely. Returning the current
    // `value` directly (below) means we don't need to update state in
    // the effect at all when debouncing is off, which keeps React happy
    // and avoids an extra render.
    if (delayMs <= 0) return
    const handle = window.setTimeout(() => setDebounced(value), delayMs)
    return () => window.clearTimeout(handle)
  }, [value, delayMs])

  return delayMs <= 0 ? value : debounced
}
