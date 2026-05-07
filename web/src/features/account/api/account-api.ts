// Account credentials API — change email + change password.
//
// Both endpoints bump the session version on success, which means the
// caller's access cookie will be rejected on the next request. The hooks
// in `../hooks/use-change-email.ts` / `use-change-password.ts` handle
// that by hard-redirecting to /login after the success toast.
//
// Errors are surfaced as ApiError. Callers map `.code` to a
// user-friendly i18n key (see `errorKeyForCode` in the components).
import { apiClient } from "@/shared/lib/api-client"

import type { Post } from "@/shared/lib/api-paths"

export type ChangeEmailRequest = {
  current_password: string
  new_email: string
}

export type ChangeEmailResponse = {
  data: {
    email: string
  }
  meta: {
    request_id: string
  }
}

export type ChangePasswordRequest = {
  current_password: string
  new_password: string
}

export type ChangePasswordResponse = {
  data: {
    ok: true
  }
  meta: {
    request_id: string
  }
}

/**
 * POST /api/v1/auth/change-email
 *
 * Verifies the user's current password and updates the account email.
 * Throws ApiError on non-2xx; callers inspect `.code`:
 *   - 400 invalid_email     → wrong format
 *   - 400 same_email        → new == current
 *   - 401 invalid_credentials → wrong current password
 *   - 409 email_already_exists → email taken
 */
export function changeEmail(
  body: ChangeEmailRequest,
): Promise<ChangeEmailResponse> {
  return apiClient<Post<"/api/v1/auth/change-email"> & ChangeEmailResponse>(
    "/api/v1/auth/change-email",
    {
      method: "POST",
      body,
    },
  )
}

/**
 * POST /api/v1/auth/change-password
 *
 * Verifies the user's current password and rotates it. The backend
 * bumps the session version, so the current access token becomes
 * invalid on the next request — caller MUST redirect to /login after
 * a successful response.
 *
 * Throws ApiError on non-2xx; callers inspect `.code`:
 *   - 400 weak_password   → fails server-side complexity check
 *   - 400 same_password   → new == current
 *   - 401 invalid_credentials → wrong current password
 */
export function changePassword(
  body: ChangePasswordRequest,
): Promise<ChangePasswordResponse> {
  return apiClient<
    Post<"/api/v1/auth/change-password"> & ChangePasswordResponse
  >("/api/v1/auth/change-password", {
    method: "POST",
    body,
  })
}
