"use client"

import { useCallback, useSyncExternalStore } from "react"

/**
 * SSR-safe hook that tracks a CSS media query.
 * Returns false on the server and during hydration,
 * then updates once the browser evaluates the query.
 *
 * Uses useSyncExternalStore so the initial value comes straight from
 * window.matchMedia on the client (no setState-in-effect cascade) and
 * re-renders only when the match state actually flips.
 */
export function useMediaQuery(query: string): boolean {
  const subscribe = useCallback(
    (onStoreChange: () => void) => {
      const mql = window.matchMedia(query)
      mql.addEventListener("change", onStoreChange)
      return () => mql.removeEventListener("change", onStoreChange)
    },
    [query],
  )

  const getSnapshot = useCallback(
    () => window.matchMedia(query).matches,
    [query],
  )

  // SSR fallback: media queries can't be evaluated on the server.
  const getServerSnapshot = useCallback(() => false, [])

  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
}
