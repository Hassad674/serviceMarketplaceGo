"use client"

import { useOrganization } from "./use-user"

/**
 * Returns true if the current user's organization permissions include
 * the given permission string. Returns false when the organization is
 * not loaded yet or when the user has no organization (solo Provider).
 *
 * Note: this collapses the loading window into `false`, which is fine
 * for inline UI affordances (disabled buttons, hidden actions). For
 * page-level guards that render a full "Access restricted" fallback,
 * prefer `usePermissionStatus` to avoid flashing the denied card while
 * the session is still loading.
 */
export function useHasPermission(permission: string): boolean {
  const { data: org } = useOrganization()
  if (!org) return false
  return org.permissions.includes(permission)
}

/**
 * Tri-state permission check, suitable for page-level guards.
 *
 * `status` resolves to:
 *   - "loading" — the /auth/me query is still in flight or refetching
 *     and we genuinely don't know yet whether the user has the
 *     permission. Callers should render a skeleton/loader, NOT the
 *     "Access restricted" fallback.
 *   - "granted" — session loaded, organization present, permission in
 *     the list.
 *   - "denied"  — session loaded with a definitive answer that the
 *     user does NOT have the permission (no organization, or org
 *     loaded without the permission).
 *
 * `granted` is a convenience boolean equivalent to `status === "granted"`.
 */
export type PermissionStatus = "loading" | "granted" | "denied"

export type PermissionResult = {
  status: PermissionStatus
  granted: boolean
  isLoading: boolean
  isError: boolean
}

export function usePermissionStatus(permission: string): PermissionResult {
  const { data: org, isLoading, isError, isSuccess } = useOrganization()

  // Still fetching the session for the first time → don't commit to
  // an answer yet. We treat "no success and no error" as loading so
  // that background refetches don't spuriously flash a denied state.
  if (isLoading || (!isSuccess && !isError)) {
    return { status: "loading", granted: false, isLoading: true, isError: false }
  }

  if (isError) {
    return { status: "denied", granted: false, isLoading: false, isError: true }
  }

  if (org && org.permissions.includes(permission)) {
    return { status: "granted", granted: true, isLoading: false, isError: false }
  }

  return { status: "denied", granted: false, isLoading: false, isError: false }
}
