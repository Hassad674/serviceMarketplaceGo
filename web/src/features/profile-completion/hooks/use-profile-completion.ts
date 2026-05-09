"use client"

import { useQueryClient, useQuery } from "@tanstack/react-query"
import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import {
  getMyProfileCompletion,
  type CompletionPersona,
  type ProfileCompletionReport,
} from "../api/profile-completion-api"

// Query key convention follows the rest of the user-scoped surfaces
// — root namespace is ["user", uid, …] so a global "user logged out"
// invalidation can fan out without naming every feature explicitly.
//
// Persona is part of the cache key so the freelance and referrer
// reports can coexist in the cache for the same user — the apporteur
// surface can mount on /referral while /profile keeps showing the
// freelance bar without one query stomping the other.
export function profileCompletionQueryKey(
  uid: string | undefined,
  persona?: CompletionPersona,
) {
  return ["user", uid, "profile-completion", persona ?? "default"] as const
}

// useProfileCompletion reads the authenticated user's completion
// report. staleTime is 30 seconds — matches the backend's
// `Cache-Control: private, max-age=30`. After a write that affects
// completion (e.g. saving the profile expertise), callers SHOULD
// invalidate via useInvalidateProfileCompletion so the bar updates
// instantly instead of waiting for the staleTime window. As a
// belt-and-braces refresh path, the query also re-fetches when the
// user re-focuses the tab after navigating to an editor — that
// covers cross-tab edits and any mutation hook that has not been
// wired to the invalidator yet.
//
// The optional `persona` argument scopes the report to a specific
// persona (used on /referral to surface the apporteur checklist).
// When omitted, the backend auto-selects from the org type.
export function useProfileCompletion(persona?: CompletionPersona) {
  const uid = useCurrentUserId()
  return useQuery<ProfileCompletionReport>({
    queryKey: profileCompletionQueryKey(uid, persona),
    queryFn: () => getMyProfileCompletion(persona),
    staleTime: 30 * 1000,
    enabled: Boolean(uid),
    refetchOnWindowFocus: true,
  })
}

// useInvalidateProfileCompletion returns a stable invalidator the
// per-section save flows can call after a successful mutation. The
// hook lives here (not at every call site) so the query-key shape
// stays encapsulated.
//
// The invalidator drops every persona variant for the current user —
// a single section save (e.g. shared photo) often affects both the
// freelance and the referrer report, so we always refresh both.
export function useInvalidateProfileCompletion() {
  const queryClient = useQueryClient()
  const uid = useCurrentUserId()
  return () => {
    if (!uid) return
    // Match on the common prefix so every persona variant in the
    // cache is invalidated in one call.
    queryClient.invalidateQueries({
      queryKey: ["user", uid, "profile-completion"],
    })
  }
}
