// Account-side two-factor authentication API (B.6.1).
//
// Lives under /me/two-factor/* — pairing the toggle (Sécurité tab)
// with the protected user-scoped endpoints. The login-side companion
// (/auth/login/verify-2fa) lives in features/auth/api/two-factor-api.ts
// because it belongs to the unauthenticated login pipeline.
//
// All errors come back as ApiError with the codes mapped in the
// backend's handleTwoFactorError (no_challenge, challenge_expired,
// invalid_code, too_many_attempts, invalid_credentials, …).
import { apiClient } from "@/shared/lib/api-client"

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
