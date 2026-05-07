"use client"

import { useEffect, useRef } from "react"
import { useQuery, useQueryClient } from "@tanstack/react-query"
import { useTranslations } from "next-intl"
import { toast } from "sonner"

import { endCall, getMyActiveCall, type MyActiveCall } from "../api/call-api"

const RECONCILE_QUERY_KEY = ["call", "me", "active"] as const
const RECONCILE_TOAST_ID = "call-orphan-reconcile"

/**
 * useReconcileCallOnMount asks the backend whether the current user
 * still has an active call entry in Redis at mount time. If the
 * answer is non-null the user is most likely a victim of orphaned
 * state — a tab closed during a call, a network blackout, a hangup
 * race. We surface a Sonner toast offering a manual "Raccrocher"
 * action that POSTs `/calls/{id}/end` and clears the state.
 *
 * The hook NEVER auto-rejoins the LiveKit room — that path is owned
 * exclusively by `useCall.connectToRoom` and would require the SDK
 * which we explicitly avoid touching in this fix. A manual button
 * keeps the UX honest: the user always knows whether they are
 * actually in a call.
 *
 * The query is gated by `enabled` to avoid firing during SSR /
 * pre-auth states. Pass `enabled=false` while no user is signed in.
 */
export function useReconcileCallOnMount(options: { enabled: boolean }) {
  const t = useTranslations("call")
  const queryClient = useQueryClient()
  const dismissedRef = useRef(false)

  const query = useQuery<MyActiveCall | null>({
    queryKey: RECONCILE_QUERY_KEY,
    queryFn: ({ signal }) => getMyActiveCall(signal),
    enabled: options.enabled,
    // Only run once per session — refetch on window focus would re-show
    // the toast every time the tab comes back, which is hostile.
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
    refetchOnMount: false,
    staleTime: Infinity,
    gcTime: Infinity,
    retry: false,
  })

  useEffect(() => {
    if (!query.data || dismissedRef.current) {
      return
    }
    const orphan = query.data
    dismissedRef.current = true

    toast(t("orphanCallTitle"), {
      id: RECONCILE_TOAST_ID,
      description: t("orphanCallDescription"),
      duration: Infinity,
      action: {
        label: t("hangup"),
        onClick: () => {
          // Best-effort: even if the API rejects (call already gone)
          // we clear the cache so the user can move on.
          endCall(orphan.call_id, 0)
            .catch(() => {
              /* swallow — orphan state may already have been GC'd */
            })
            .finally(() => {
              queryClient.setQueryData(RECONCILE_QUERY_KEY, null)
              toast.dismiss(RECONCILE_TOAST_ID)
            })
        },
      },
    })
  }, [query.data, queryClient, t])

  return query
}

export const reconcileCallQueryKey = RECONCILE_QUERY_KEY
