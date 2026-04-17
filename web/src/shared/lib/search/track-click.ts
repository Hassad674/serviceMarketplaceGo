"use client"

/**
 * track-click.ts fires the click-through beacon against the
 * `/api/v1/search/track` endpoint. Exposed as a plain function so
 * any search-result renderer can emit the event on `onClick`
 * without dragging TanStack Query state through the tree.
 *
 * The request is fire-and-forget:
 *  - Prefer `navigator.sendBeacon` so the call survives navigation
 *    triggered by the click itself.
 *  - Fall back to a `keepalive: true` fetch when Beacon is
 *    unavailable (older browsers, mobile WebKit with strict
 *    privacy settings).
 *  - Errors are swallowed — analytics must never interfere with
 *    the user-facing action.
 */

import { API_BASE_URL } from "../api-client"

/**
 * trackSearchClick fires the beacon. Safe to call from any event
 * handler — returns synchronously.
 */
export function trackSearchClick(
  searchId: string,
  docId: string,
  position: number,
): void {
  if (!searchId || !docId || position < 0) return
  if (typeof window === "undefined") return

  const url = buildTrackURL(searchId, docId, position)
  try {
    if (
      typeof navigator !== "undefined" &&
      typeof navigator.sendBeacon === "function"
    ) {
      navigator.sendBeacon(url)
      return
    }
    // Fallback for browsers without Beacon.
    void fetch(url, { method: "GET", credentials: "include", keepalive: true })
  } catch {
    // Swallow — analytics must never break the user-facing action.
  }
}

/**
 * buildTrackURL composes the full endpoint with query params. Exposed
 * for unit tests; consumers should call `trackSearchClick` instead.
 */
export function buildTrackURL(
  searchId: string,
  docId: string,
  position: number,
): string {
  const base = API_BASE_URL || ""
  const params = new URLSearchParams({
    search_id: searchId,
    doc_id: docId,
    position: String(position),
  })
  return `${base}/api/v1/search/track?${params.toString()}`
}
