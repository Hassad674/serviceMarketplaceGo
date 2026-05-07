"use client"

import { useMutation } from "@tanstack/react-query"
import {
  changePassword,
  type ChangePasswordRequest,
} from "../api/account-api"

/**
 * useChangePassword — TanStack mutation wrapping the change-password
 * endpoint.
 *
 * UI-agnostic: the component is responsible for the toast, password
 * reset, and post-success redirect. Errors are bubbled up as ApiError.
 *
 * NB: backend bumps the session version on success → caller MUST
 * redirect to /login (or otherwise refresh the session) after a 200.
 */
export function useChangePassword() {
  return useMutation({
    mutationFn: (body: ChangePasswordRequest) => changePassword(body),
  })
}
