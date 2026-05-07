"use client"

import { useMutation } from "@tanstack/react-query"
import { changeEmail, type ChangeEmailRequest } from "../api/account-api"

/**
 * useChangeEmail — TanStack mutation wrapping the change-email endpoint.
 *
 * The hook stays UI-agnostic on purpose: the component owns the toast,
 * the form reset, and the post-success redirect. Errors are bubbled
 * up as ApiError so the caller can map `.code` → i18n key.
 *
 * NB: backend bumps the session version on success → caller MUST
 * redirect to /login (or otherwise refresh the session) after a 200.
 */
export function useChangeEmail() {
  return useMutation({
    mutationFn: (body: ChangeEmailRequest) => changeEmail(body),
  })
}
