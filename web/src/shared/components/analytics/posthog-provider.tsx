"use client"

import { useEffect } from "react"

import { useSession } from "@/shared/hooks/use-user"
import {
  identifyUser,
  initPostHog,
  resetPostHog,
  setOrganizationGroup,
} from "@/shared/lib/posthog"

/**
 * PostHogProvider wires the browser SDK into the React tree.
 *
 * Responsibilities (in order):
 *   1. Initialise the SDK on first client render. Idempotent — safe
 *      under React 19 strict-mode double-invoke.
 *   2. When the auth `useSession()` query resolves to a logged-in
 *      user, identify the distinct id with profile attributes and
 *      attach the org group. When it resolves to "no session", reset.
 *   3. Render no UI — providers are pure side-effect components.
 *
 * Why the consumer of `useSession()` lives here: this is the single
 * place in the tree where we have BOTH the auth query AND the
 * PostHog SDK boot lifecycle. Putting the identify call inside the
 * auth feature itself would couple the auth feature to PostHog; the
 * provider keeps analytics a top-level concern.
 */
export function PostHogProvider() {
  const { data: session } = useSession()

  // Initialise once on mount.
  useEffect(() => {
    initPostHog()
  }, [])

  // Identify on login, reset on logout. Watch session.user.id rather
  // than session itself so we don't re-identify on every cache
  // refresh that returns the same user.
  useEffect(() => {
    if (!session?.user?.id) {
      resetPostHog()
      return
    }
    identifyUser(session.user.id, {
      role: session.user.role,
      email_verified: session.user.email_verified,
      referrer_enabled: session.user.referrer_enabled,
    })
    if (session.organization?.id) {
      setOrganizationGroup(session.organization.id, {
        type: session.organization.type,
        member_role: session.organization.member_role,
      })
    }
  }, [session?.user?.id, session?.user?.role, session?.user?.email_verified, session?.user?.referrer_enabled, session?.organization?.id, session?.organization?.type, session?.organization?.member_role])

  return null
}
