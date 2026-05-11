// Sessions API — the user-facing surface that drives the Malt-style
// session list on the Sécurité page (SEC-SESSIONS).
//
// Three endpoints under /api/v1/me/sessions, all strictly user-scoped:
//   GET    /me/sessions                — list the caller's active sessions
//   DELETE /me/sessions/{id}           — revoke one session
//   POST   /me/sessions/revoke-others  — revoke everything except the current chain
//
// The wire shape mirrors the column names so the contract is trivial
// to diff against the backend schema (migration 150).
import { apiClient } from "@/shared/lib/api-client"

/**
 * One row in the Sécurité-page session list. Maps 1:1 with the
 * Postgres user_sessions row + a server-computed `is_current` flag.
 */
export type Session = {
  id: string
  device_label: string
  browser?: string
  os?: string
  city?: string
  country_code?: string
  ip_anonymized?: string
  login_method: string
  created_at: string
  last_used_at: string
  expires_at: string
  is_current: boolean
}

export type ListSessionsResponse = {
  data: Session[]
}

const ENDPOINT = "/api/v1/me/sessions"

/**
 * GET /api/v1/me/sessions — every still-active session for the caller.
 */
export function listSessions(): Promise<ListSessionsResponse> {
  return apiClient<ListSessionsResponse>(ENDPOINT)
}

/**
 * DELETE /api/v1/me/sessions/{id} — revoke one of the caller's
 * sessions. Returns a 204 No Content envelope (the apiClient resolves
 * to undefined).
 */
export async function revokeSession(id: string): Promise<void> {
  await apiClient<void>(`${ENDPOINT}/${id}`, { method: "DELETE" })
}

/**
 * POST /api/v1/me/sessions/revoke-others — revoke every active session
 * except the current one. The server discovers the current session via
 * the request cookies.
 */
export async function revokeOtherSessions(): Promise<void> {
  await apiClient<void>(`${ENDPOINT}/revoke-others`, { method: "POST" })
}
