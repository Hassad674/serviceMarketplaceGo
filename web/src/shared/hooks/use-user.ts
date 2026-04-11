"use client"

import { useQuery, useQueryClient } from "@tanstack/react-query"

import { API_BASE_URL } from "@/shared/lib/api-client"

// Three hooks share a single underlying query to /api/v1/auth/me.
// - useSession() returns the full { user, organization } payload.
// - useUser() and useOrganization() are selectors on the same query.
// TanStack Query deduplicates by queryKey, so mounting any combination
// of the three hooks results in exactly one network request.

const API_URL = API_BASE_URL

const SESSION_QUERY_KEY = ["session"] as const

// Paths that should never trigger the "session died — redirect to login"
// behaviour. On unauthenticated pages we legitimately expect /auth/me to
// return 401 (the user isn't logged in yet), and forcing a redirect
// would break the /login and /register flows.
const AUTH_PUBLIC_PATHS = ["/login", "/register", "/forgot-password", "/reset-password"]

function isOnPublicAuthPath(): boolean {
  if (typeof window === "undefined") return true // SSR — never redirect
  const path = window.location.pathname
  return AUTH_PUBLIC_PATHS.some((p) => path === p || path.startsWith(`${p}/`) || path.includes(p))
}

export type CurrentUser = {
  id: string
  email: string
  first_name: string
  last_name: string
  display_name: string
  role: "agency" | "enterprise" | "provider"
  referrer_enabled: boolean
  email_verified: boolean
  kyc_status: "none" | "pending" | "restricted" | "completed"
  kyc_deadline?: string
  created_at: string
}

export type CurrentOrganization = {
  id: string
  type: string
  owner_user_id: string
  member_role: string
  member_title: string
  permissions: string[]
  // Populated only while an ownership transfer is in flight. Surfaced
  // on /me so the team page can render the "transfer in progress"
  // banner for both the initiator (current Owner) and the target.
  pending_transfer_to_user_id?: string
  pending_transfer_initiated_at?: string
  pending_transfer_expires_at?: string
}

export type SessionResponse = {
  user: CurrentUser
  // null for Providers who are not part of any organization.
  organization: CurrentOrganization | null
}

async function fetchSession(): Promise<SessionResponse> {
  const res = await fetch(`${API_URL}/api/v1/auth/me`, {
    credentials: "include",
  })
  if (!res.ok) {
    // R16 zombie-session fix: when the backend tells us the session is
    // no longer valid (401), hard-redirect to /login. This catches two
    // cases the old "throw new Error" handling swallowed silently:
    //   1. access cookie expired / not present (normal sign-out flow)
    //   2. the user account has been deleted (e.g. operator who left
    //      their org) — backend now returns 401 "session_invalid"
    //      instead of 404 so the frontend knows to log out instead of
    //      retrying forever.
    // We also redirect on 404 as belt-and-braces for older backends
    // that might still return 404 for this case.
    if ((res.status === 401 || res.status === 404) && !isOnPublicAuthPath()) {
      // Hard-redirect destroys the in-memory React tree and the
      // TanStack Query cache — cheaper and safer than manually
      // invalidating every query. Mirrors useLogout().
      window.location.href = "/login"
    }
    throw new Error("Not authenticated")
  }
  return res.json()
}

const sessionQueryOptions = {
  queryKey: SESSION_QUERY_KEY,
  queryFn: fetchSession,
  staleTime: 5 * 60 * 1000,
  retry: false,
} as const

/**
 * Returns the full session payload (user + organization).
 *
 * Prefer `useUser()` or `useOrganization()` when you only need one of the
 * two — they return referentially stable slices and skip re-renders when
 * the other slice changes. Use `useSession()` only when a single component
 * legitimately needs both objects in the same render (e.g. a dashboard
 * banner "Hi {user.first_name}, {member_role} of {organization.name}").
 */
export function useSession() {
  return useQuery(sessionQueryOptions)
}

export function useUser() {
  return useQuery({
    ...sessionQueryOptions,
    select: (data) => data.user,
  })
}

export function useOrganization() {
  return useQuery({
    ...sessionQueryOptions,
    select: (data) => data.organization,
  })
}

export function useLogout() {
  const queryClient = useQueryClient()

  return async function logout() {
    await fetch(`${API_URL}/api/v1/auth/logout`, {
      method: "POST",
      credentials: "include",
    })
    queryClient.clear()
    // Hard redirect to destroy all in-memory state (React tree, query cache,
    // WebSocket connections). router.push would keep stale data in memory.
    window.location.href = "/login"
  }
}
