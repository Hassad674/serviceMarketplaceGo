"use client"

import { useOrganization } from "./use-user"

/**
 * Returns true if the current user's organization permissions include
 * the given permission string. Returns false when the organization is
 * not loaded yet or when the user has no organization (solo Provider).
 */
export function useHasPermission(permission: string): boolean {
  const { data: org } = useOrganization()
  if (!org) return false
  return org.permissions.includes(permission)
}
