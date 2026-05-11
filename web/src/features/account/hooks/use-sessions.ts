"use client"

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  listSessions,
  revokeOtherSessions,
  revokeSession,
  type ListSessionsResponse,
} from "../api/sessions-api"

const SESSIONS_QUERY_KEY = ["account", "sessions"] as const

/**
 * useSessions — TanStack query over /api/v1/me/sessions for the Malt-
 * style Sécurité-page session list.
 *
 * staleTime = 30s so the table refreshes when the tab is reopened
 * without bombarding the API; refetchOnWindowFocus stays on by default
 * because a stale session list is worse UX than a brief network
 * round-trip.
 */
export function useSessions() {
  return useQuery<ListSessionsResponse>({
    queryKey: SESSIONS_QUERY_KEY,
    queryFn: listSessions,
    staleTime: 30_000,
  })
}

/**
 * useRevokeSession — mutation wrapper for DELETE /me/sessions/{id}.
 *
 * Optimistically removes the row from the cached list so the UI
 * updates immediately; on error the cache is restored from the
 * pre-mutation snapshot and the parent component should surface a
 * toast.
 */
export function useRevokeSession() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => revokeSession(id),
    onMutate: async (id) => {
      await qc.cancelQueries({ queryKey: SESSIONS_QUERY_KEY })
      const previous = qc.getQueryData<ListSessionsResponse>(SESSIONS_QUERY_KEY)
      if (previous) {
        qc.setQueryData<ListSessionsResponse>(SESSIONS_QUERY_KEY, {
          ...previous,
          data: previous.data.filter((row) => row.id !== id),
        })
      }
      return { previous }
    },
    onError: (_err, _id, ctx) => {
      if (ctx?.previous) {
        qc.setQueryData(SESSIONS_QUERY_KEY, ctx.previous)
      }
    },
    onSettled: () => {
      qc.invalidateQueries({ queryKey: SESSIONS_QUERY_KEY })
    },
  })
}

/**
 * useRevokeOtherSessions — mutation wrapper for
 * POST /me/sessions/revoke-others. After success the cache is
 * invalidated so the list re-fetches and the freshly-revoked rows
 * disappear; we do NOT pre-compute the optimistic shape because the
 * "current" detection lives server-side.
 */
export function useRevokeOtherSessions() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => revokeOtherSessions(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: SESSIONS_QUERY_KEY })
    },
  })
}
