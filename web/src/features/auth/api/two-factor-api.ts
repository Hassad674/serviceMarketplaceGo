// Two-factor authentication API surface.
//
// Three endpoints are exposed by the backend (B.6.1, commit b930824a):
//   1. POST /api/v1/auth/login/verify-2fa
//      body: { user_id, challenge_id, code }
//      → completes a login that was gated by `requires_2fa: true`
//   2. POST /api/v1/me/two-factor/enable
//      body: empty | { code }
//      → two-step opt-in: first call issues an email challenge (202 +
//        challenge_id), second call (with code) flips the flag and
//        returns 200 + { enabled: true }.
//   3. POST /api/v1/me/two-factor/disable
//      body: { current_password }
//      → flips the flag off after re-auth, returns 200 + { enabled: false }.
//
// All errors come back as ApiError with the codes mapped in
// handleTwoFactorError (no_challenge, challenge_expired, invalid_code,
// too_many_attempts, invalid_credentials, session_invalid, …).
import { apiClient, API_BASE_URL } from "@/shared/lib/api-client"

const API_URL = API_BASE_URL

/**
 * Login response shape extension when the user has opted into 2FA.
 *
 * The `requires_2fa` branch returns ONLY these three fields — no user
 * payload, no tokens. The client stores `user_id` + `challenge_id`
 * locally and POSTs them back to /auth/login/verify-2fa together with
 * the 6-digit code the user typed in.
 */
export type LoginTwoFactorChallenge = {
  requires_2fa: true
  user_id: string
  challenge_id: string
}

export type VerifyTwoFactorRequest = {
  user_id: string
  challenge_id: string
  code: string
}

/**
 * Completes a login that was gated by 2FA. On success the backend
 * issues the session cookie (web) or returns the bearer pair (mobile)
 * just like a normal login response. We never read the body here — the
 * caller invalidates the `["session"]` query and lets the dashboard
 * refetch /auth/me.
 */
export async function verifyTwoFactor(req: VerifyTwoFactorRequest): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/auth/login/verify-2fa`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({
      error: "unknown",
      message: "verification failed",
    }))
    // Re-throw using the same shape /auth/login uses so the calling
    // form can branch on `err.code` exactly like it does for the
    // initial login error path.
    throw Object.assign(new Error(err.message || "verification failed"), {
      code: err.error,
      reason: err.reason,
    })
  }
}

/**
 * Step 1 of the enable flow — empty body. The backend issues a
 * confirmation challenge by email and returns
 * `{ requires_confirmation: true, challenge_id }` with HTTP 202.
 */
export type EnableTwoFactorChallenge = {
  requires_confirmation: true
  challenge_id: string
}

export async function requestEnableTwoFactor(): Promise<EnableTwoFactorChallenge> {
  return apiClient<EnableTwoFactorChallenge>("/api/v1/me/two-factor/enable", {
    method: "POST",
  })
}

/**
 * Step 2 of the enable flow — submit the 6-digit code. On success the
 * backend flips the flag and returns `{ enabled: true }`.
 */
export type EnableTwoFactorConfirmation = { enabled: true }

export async function confirmEnableTwoFactor(code: string): Promise<EnableTwoFactorConfirmation> {
  return apiClient<EnableTwoFactorConfirmation>("/api/v1/me/two-factor/enable", {
    method: "POST",
    body: { code },
  })
}

/**
 * Disable flow — re-auth with the caller's CURRENT password, then the
 * flag flips off. The next login skips the 2FA gate.
 */
export type DisableTwoFactorRequest = { current_password: string }
export type DisableTwoFactorResponse = { enabled: false }

export async function disableTwoFactor(
  req: DisableTwoFactorRequest,
): Promise<DisableTwoFactorResponse> {
  return apiClient<DisableTwoFactorResponse>("/api/v1/me/two-factor/disable", {
    method: "POST",
    body: req,
  })
}
