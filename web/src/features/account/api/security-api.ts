// Security activity API — read-only feed of the caller's recent
// authentication audit events.
//
// The endpoint is strictly user-scoped: a member of an organization
// only sees their OWN authentication history, never another member's.
// Cursor pagination matches the rest of the marketplace API.
//
// Errors are surfaced as ApiError. The hook layer catches and renders
// `errors.generic` for any non-401 failure — the page is purely
// informational so a fine-grained error map adds noise without value.
import { apiClient } from "@/shared/lib/api-client"

/**
 * One authentication-related audit row.
 *
 * `access_kind` is the device classification from the user-agent
 * parser (`desktop` / `mobile` / `tablet` / `unknown`). The UI uses
 * it to pick an icon, and falls back to a neutral "—" when the
 * `user_agent_summary` is empty.
 */
export type SecurityActivityEvent = {
  id: string
  action: string
  ip_address?: string
  user_agent_summary: string
  access_kind: "desktop" | "mobile" | "tablet" | "unknown"
  country_hint?: string
  created_at: string
}

export type SecurityActivityResponse = {
  data: SecurityActivityEvent[]
  next_cursor?: string
}

export type ListSecurityActivityParams = {
  cursor?: string
  limit?: number
}

const ENDPOINT = "/api/v1/me/security/activity"

/**
 * GET /api/v1/me/security/activity
 *
 * Returns the most recent authentication-related events attributable
 * to the calling user, newest-first. The action set is the
 * `auth.*` family (login_success, logout, token_refresh, password
 * reset request/complete) — feature-level audit (receipts, referrals,
 * ...) is intentionally filtered out.
 */
export function listSecurityActivity(
  params: ListSecurityActivityParams = {},
): Promise<SecurityActivityResponse> {
  const search = new URLSearchParams()
  if (params.cursor) search.set("cursor", params.cursor)
  if (params.limit) search.set("limit", String(params.limit))
  const qs = search.toString()
  const url = qs ? `${ENDPOINT}?${qs}` : ENDPOINT
  return apiClient<SecurityActivityResponse>(url)
}
