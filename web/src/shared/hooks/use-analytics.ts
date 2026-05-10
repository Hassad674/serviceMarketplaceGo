"use client"

import { useCallback } from "react"

import { captureEvent } from "@/shared/lib/posthog"

/**
 * Thin React wrapper around the lib/posthog `captureEvent` helper.
 *
 * Why a hook rather than calling `captureEvent` directly: components
 * are heavy clients, and stable callback identity matters for
 * `useEffect` deps + memoised handlers. A hook returns a stable
 * `useCallback` so re-renders never re-run effects that watch the
 * capture function.
 *
 * Usage:
 *   const { capture } = useAnalytics()
 *   capture("landing.search_submitted", { query: "react" })
 */
export function useAnalytics() {
  const capture = useCallback(
    (event: string, properties?: Record<string, unknown>) => {
      captureEvent(event, properties)
    },
    [],
  )
  return { capture }
}
