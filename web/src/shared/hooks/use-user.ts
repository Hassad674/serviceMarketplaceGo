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
//
// Marketing / public listing routes (`/`, `/agencies`, `/freelancers`,
// `/referrers`, `/opportunities`, …) are also included: an incognito
// visitor browsing the catalogue MUST see the public surface, never a
// surprise hop to /login. The middleware already gates the truly
// protected paths (see PROTECTED_PATHS in src/middleware.ts), so the
// list here is the inverse — known public surfaces.
const AUTH_PUBLIC_PATHS = [
  "/",
  "/login",
  "/register",
  "/forgot-password",
  "/reset-password",
  "/agencies",
  "/freelancers",
  "/freelances",
  "/referrers",
  "/opportunities",
  "/clients",
  // `/invitation/<token>` is reached from the email link by an
  // unauthenticated visitor — `/auth/me` legitimately returns 401
  // here (the new member has no session yet) and a hard-redirect
  // to /login would prevent them from accepting the invitation.
  // The PostHogProvider mounted in the locale layout fires
  // `useSession()` on every page, so this whitelist is the only
  // thing standing between the visitor and a surprise hop to /login.
  "/invitation",
]

// Locale prefixes the next-intl router prepends to every URL. Keep
// in sync with `src/i18n/routing.ts`. We strip them before matching
// so the public-path test does not depend on the active locale.
const LOCALE_PREFIXES = ["/fr", "/en"]

function stripLocalePrefix(pathname: string): string {
  for (const prefix of LOCALE_PREFIXES) {
    if (pathname === prefix) return "/"
    if (pathname.startsWith(`${prefix}/`)) return pathname.slice(prefix.length)
  }
  return pathname
}

function isOnPublicAuthPath(): boolean {
  if (typeof window === "undefined") return true // SSR — never redirect
  const path = stripLocalePrefix(window.location.pathname)
  if (path === "/") return true
  return AUTH_PUBLIC_PATHS.some((p) => {
    if (p === "/") return false // exact-matched above
    return path === p || path.startsWith(`${p}/`)
  })
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
  /** Set when the user is in their 30-day GDPR cooldown (P5). */
  deleted_at?: string
  /** RFC3339 — when the cron will hard-purge if cancel does not land. */
  hard_delete_at?: string
  /**
   * FIX-2FA — true when email 2FA is opted-in. The backend now
   * surfaces this on every /auth/me payload (and on the login
   * response in web mode), so the Sécurité toggle can render the
   * correct initial state on first paint without keeping a parallel
   * local copy. Still marked optional in the type to stay
   * defensively forward-compatible with any handler that hasn't
   * been updated yet — the toggle treats `undefined` as `false`.
   */
  two_factor_email_enabled?: boolean
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

// Session lifetime is measured in hours (httpOnly cookie TTL). The
// payload only changes on explicit mutations (login, logout, profile
// edit, team transfer) which all call `queryClient.invalidateQueries`
// or `queryClient.clear` themselves — there is no value in TanStack
// Query firing /auth/me on its own beyond the very first mount.
//
// Hardening (PERF-FIX-W-AUTH-ME-FANOUT):
//   * staleTime: 30 min — tolerates page-navigation re-renders + dev
//     hot-reloads without re-fetching. Mutations that change the
//     session (logout, role change, transfer) explicitly invalidate
//     ["session"] so cache freshness is owned by the writer, not by
//     a wall-clock timer.
//   * gcTime: 30 min — keep the unmounted cache around long enough
//     to survive App Router transitions (which briefly unmount every
//     consumer between layouts).
//   * refetchOnWindowFocus / Reconnect / Mount: explicit `false` so
//     the per-hook contract does not depend on the global
//     QueryClient defaults (those are correct today, but a future
//     edit must not silently re-enable a refetch storm here).
//   * retry: false — a 401 means logout, not a transient error. The
//     fetcher already hard-redirects to /login.
//   * retryOnMount: false — ROOT-CAUSE FIX for the /auth/me fan-out.
//     `refetchOnMount: false` only prevents re-fetches when the cache
//     already holds `data !== undefined`. When the very first /auth/me
//     fails (401 on a public page, network error, …) the cache stays
//     in `{ data: undefined, status: "error" }` forever, and TanStack
//     Query's `shouldLoadOnMount` then triggers a NEW fetch for every
//     observer that subscribes (see query-core/queryObserver.js
//     `shouldLoadOnMount`). A public page like /freelancers/[id] mounts
//     ~30 distinct session consumers (PublicLayout, PostHogProvider,
//     SendMessageButton, sidebar/header on logged-in branches, …), so
//     a single 401 turned into a 30-200+ request storm in <1 s,
//     tripping the global IP rate limit and bricking the page.
//     `retryOnMount: false` makes that gate close once the first
//     attempt has resolved (success OR error), so subsequent observers
//     read the cached verdict instead of re-fetching. Login / register
//     flows must explicitly `invalidateQueries({ queryKey: ["session"] })`
//     to force a refetch when the session legitimately changes.
const SESSION_STALE_TIME_MS = 30 * 60 * 1000
const SESSION_GC_TIME_MS = 30 * 60 * 1000

const sessionQueryOptions = {
  queryKey: SESSION_QUERY_KEY,
  queryFn: fetchSession,
  staleTime: SESSION_STALE_TIME_MS,
  gcTime: SESSION_GC_TIME_MS,
  refetchOnWindowFocus: false,
  refetchOnReconnect: false,
  refetchOnMount: false,
  retry: false,
  retryOnMount: false,
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
