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
    if (delayMs <= 0) {
      setDebounced(value)
      return
    }
    const handle = window.setTimeout(() => setDebounced(value), delayMs)
    return () => window.clearTimeout(handle)
  }, [value, delayMs])

  return debounced
}
