"use client"

/**
 * use-search-key.ts is the TanStack Query hook that fetches a
 * scoped Typesense API key from the backend and rotates it before
 * the 1h TTL expires.
 *
 * The backend issues keys with a 1h TTL; we cache for 55 minutes
 * (5 min safety margin) so the next request always lands a fresh
 * key before the old one is rejected by Typesense.
 */

import { useQuery } from "@tanstack/react-query"
import { apiClient } from "@/shared/lib/api-client"
import type { SearchDocumentPersona } from "./typesense-client"

/** SearchKeyResponse is the shape returned by GET /api/v1/search/key. */
export interface SearchKeyResponse {
  key: string
  host: string
  expires_at: number
  persona: SearchDocumentPersona
}

/** SCOPED_KEY_STALE_MS is 55 minutes — 5 min safety vs the 1h TTL. */
const SCOPED_KEY_STALE_MS = 55 * 60 * 1000

/** searchKeyQueryKey isolates the cache entry per persona. */
export function searchKeyQueryKey(persona: SearchDocumentPersona) {
  return ["search", "scoped-key", persona] as const
}

/**
 * useSearchKey fetches and caches a scoped Typesense API key for
 * the given persona. Returns null while the query is loading or
 * errored so callers can short-circuit before instantiating the
 * client.
 *
 * The hook is safe to call with a `null` persona — the underlying
 * query is disabled and no network request fires.
 */
export function useSearchKey(persona: SearchDocumentPersona | null): {
  key: SearchKeyResponse | null
  isLoading: boolean
  error: Error | null
} {
  const enabled = persona !== null
  const query = useQuery({
    queryKey: enabled ? searchKeyQueryKey(persona) : ["search", "scoped-key", "disabled"],
    queryFn: () => fetchScopedKey(persona as SearchDocumentPersona),
    enabled,
    staleTime: SCOPED_KEY_STALE_MS,
    gcTime: SCOPED_KEY_STALE_MS,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    retry: 1,
  })

  return {
    key: query.data ?? null,
    isLoading: query.isLoading,
    error: (query.error as Error | null) ?? null,
  }
}

/**
 * fetchScopedKey is the raw fetcher exported for tests + for any
 * server-side callers that need the key outside React.
 */
export async function fetchScopedKey(persona: SearchDocumentPersona): Promise<SearchKeyResponse> {
  return apiClient<SearchKeyResponse>(`/api/v1/search/key?persona=${encodeURIComponent(persona)}`)
}
