"use client"

import { useUser } from "@/shared/hooks/use-user"
import { useWorkspace } from "@/shared/hooks/use-workspace"
import type { DashboardLayout } from "../types"

// useDashboardLayout collapses the user role + referrer-mode toggle
// into a single layout token consumed by the dashboard shell. The
// derived value is referentially stable as long as the underlying
// session payload is — useUser() and useWorkspace() are TanStack
// Query selectors and a Zustand-backed store, both of which return
// the same reference when the slice has not changed.
//
// The function is exported as a hook (not a pure helper) because it
// reads two stores; consumers should not duplicate the role / mode
// resolution logic locally.

export interface DashboardLayoutResult {
  layout: DashboardLayout
  isReferrerMode: boolean
}

export function useDashboardLayout(): DashboardLayoutResult {
  const { data: user } = useUser()
  const { isReferrerMode } = useWorkspace()

  const role = user?.role ?? "enterprise"
  const referrerMode = role === "provider" && isReferrerMode

  const layout = resolveLayout(role, referrerMode)
  return { layout, isReferrerMode: referrerMode }
}

function resolveLayout(
  role: "agency" | "enterprise" | "provider",
  referrerMode: boolean,
): DashboardLayout {
  if (referrerMode) return "referrer"
  return role
}
