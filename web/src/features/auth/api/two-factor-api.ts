// Auth-side two-factor API surface — limited to the unauthenticated
// login pipeline.
//
// `/api/v1/auth/login/verify-2fa` completes a login that was gated by
// `requires_2fa: true` in the response of /auth/login. Body shape:
// { user_id, challenge_id, code }. On success the backend issues the
// session cookie (web) or returns the bearer pair (mobile) just like
// a normal login response.
//
// The user-scoped enable / disable endpoints live in
// features/account/api/two-factor-api.ts because they are
// authenticated and pair with the Sécurité toggle.
import { API_BASE_URL } from "@/shared/lib/api-client"

const API_URL = API_BASE_URL

export type VerifyTwoFactorRequest = {
  user_id: string
  challenge_id: string
  code: string
}

/**
 * Completes a login that was gated by 2FA. We never read the body —
 * the caller invalidates the `["session"]` query and lets the
 * dashboard refetch /auth/me. Errors are re-thrown with `code` and
 * `reason` so the form can branch the same way it does for the
 * initial /auth/login call.
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
    throw Object.assign(new Error(err.message || "verification failed"), {
      code: err.error,
      reason: err.reason,
    })
  }
}
