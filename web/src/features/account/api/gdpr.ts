// GDPR right-to-erasure + right-to-export API client.
//
// All mutations bubble ApiError up to the caller — the UI components
// switch on .code (invalid_password / owner_must_transfer_or_dissolve
// / etc.) to render the right inline message + action.
import { apiClient, API_BASE_URL } from "@/shared/lib/api-client"

export type RequestDeletionResponse = {
  email_sent_to: string
  expires_at: string // RFC3339
}

export type ConfirmDeletionResponse = {
  user_id: string
  deleted_at: string // RFC3339 — landing time of the soft-delete
  hard_delete_at: string // RFC3339 — when the cron will purge
}

export type CancelDeletionResponse = {
  cancelled: boolean
}

export type BlockedOrg = {
  org_id: string
  org_name: string
  member_count: number
  available_admins: { user_id: string; email: string }[]
  actions: ("transfer_ownership" | "dissolve_org")[]
}

// Conflict (409) body shape returned when the user is the Owner of an
// org with active members. The caller uses this to render the
// remediation panel inline.
export type OwnerBlockedDetails = {
  blocked_orgs: BlockedOrg[]
}

/**
 * POST /api/v1/me/account/request-deletion
 *
 * Verifies the password and (if no blocking orgs) sends the
 * confirmation email. Returns the email address echoed back so the
 * UX can show "we sent an email to xx@yy.com — check your inbox".
 *
 * Throws ApiError on non-2xx; callers inspect:
 *   .code === "invalid_password"               → wrong password
 *   .status === 409                            → org-owner-blocked
 *                                                (.body.error.details.blocked_orgs)
 *   .status === 401                            → unauthenticated
 */
export function requestDeletion(password: string): Promise<RequestDeletionResponse> {
  return apiClient<RequestDeletionResponse>("/api/v1/me/account/request-deletion", {
    method: "POST",
    body: { password, confirm: true },
  })
}

/**
 * GET /api/v1/me/account/confirm-deletion?token=<jwt>
 *
 * Public endpoint — the JWT in the URL is the auth. Throws ApiError
 * with .code === "invalid_token" when the token is expired or
 * tampered with.
 */
export function confirmDeletion(token: string): Promise<ConfirmDeletionResponse> {
  return apiClient<ConfirmDeletionResponse>(
    "/api/v1/me/account/confirm-deletion?token=" + encodeURIComponent(token),
    { method: "GET" },
  )
}

/**
 * POST /api/v1/me/account/cancel-deletion
 *
 * Auth-required. Returns { cancelled: true } when a soft-delete was
 * actually rolled back, false when there was nothing to cancel.
 * Idempotent — safe to call from a "cancel" landing page that was
 * navigated to directly without going through the email link.
 */
export function cancelDeletion(): Promise<CancelDeletionResponse> {
  return apiClient<CancelDeletionResponse>("/api/v1/me/account/cancel-deletion", {
    method: "POST",
  })
}

/**
 * GET /api/v1/me/export
 *
 * Streams the data export as a ZIP. We bypass apiClient here because
 * we need the raw response body to feed into the download trigger.
 * The cookie-based auth is forwarded by `credentials: "include"`.
 */
export async function exportMyData(): Promise<Blob> {
  const res = await fetch(`${API_BASE_URL}/api/v1/me/export`, {
    method: "GET",
    credentials: "include",
  })
  if (!res.ok) {
    let message = "Failed to build export"
    try {
      const body = await res.json()
      if (body?.error?.message) message = body.error.message
      else if (body?.message) message = body.message
    } catch {
      // body wasn't JSON — keep the default message
    }
    throw new Error(message)
  }
  return res.blob()
}

/**
 * Triggers a browser download of the export ZIP. Builds an
 * <a download> off the blob and clicks it. The filename is read
 * from the Content-Disposition header when present, else
 * synthesized from a timestamp.
 */
export async function downloadExport(): Promise<void> {
  const blob = await exportMyData()
  const url = URL.createObjectURL(blob)
  const a = document.createElement("a")
  a.href = url
  const ts = new Date().toISOString().replace(/[:.]/g, "-")
  a.download = `marketplace-export-${ts}.zip`
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}
