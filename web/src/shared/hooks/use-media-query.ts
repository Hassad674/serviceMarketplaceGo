"use client"

import { useState, useEffect } from "react"

/**
 * SSR-safe hook that tracks a CSS media query.
 * Returns false on the server and during hydration,
 * then updates once the browser evaluates the query.
 */
export function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(false)

  useEffect(() => {
    const mql = window.matchMedia(query)
    setMatches(mql.matches)

    function onChange(e: MediaQueryListEvent) {
      setMatches(e.matches)
    }

    mql.addEventListener("change", onChange)
    return () => mql.removeEventListener("change", onChange)
  }, [query])

  return matches
}
