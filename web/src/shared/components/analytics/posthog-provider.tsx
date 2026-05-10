"use client"

import { useEffect, useState } from "react"

import { useSession } from "@/shared/hooks/use-user"
import {
  identifyUser,
  initPostHog,
  resetPostHog,
  setOrganizationGroup,
} from "@/shared/lib/posthog"
import { readConsent } from "@/shared/lib/posthog-consent"

/**
 * PostHogProvider wires the browser SDK into the React tree.
 *
 * Responsibilities (in order):
 *   1. WAIT for the user to grant the `analytics` category via the
 *      cookie-consent CMP. Until consent is granted, the SDK is NOT
 *      initialised — no /decide call, no pageview, no identify.
 *      This is the RGPD pause-before-consent gate (Phase A.2).
 *   2. When consent flips to "accepted" (first visit Accept, or after
 *      a programmatic toggle), boot the SDK once. Idempotent — safe
 *      under React 19 strict-mode double-invoke.
 *   3. When the auth `useSession()` query resolves to a logged-in
 *      user, identify the distinct id with profile attributes and
 *      attach the org group. When it resolves to "no session", reset.
 *   4. Render no UI — providers are pure side-effect components.
 *
 * Why we listen to a `analytics:consent-changed` window event: the
 * CMP fires this event from `applyCustomConsent()` whenever the user
 * flips the analytics toggle. Listening here lets us boot the SDK
 * lazily without forcing the host page to reload after Accept.
 */
export function PostHogProvider() {
  const { data: session } = useSession()
  const [hasConsent, setHasConsent] = useState(false)

  // Subscribe to consent changes (initial read + same-tab + cross-tab).
  useEffect(() => {
    function refresh() {
      setHasConsent(readConsent() === "accepted")
    }
    refresh()
    window.addEventListener("analytics:consent-changed", refresh)
    window.addEventListener("storage", refresh)
    return () => {
      window.removeEventListener("analytics:consent-changed", refresh)
      window.removeEventListener("storage", refresh)
    }
  }, [])

  // Boot the SDK only after consent. `initPostHog` is idempotent.
  useEffect(() => {
    if (!hasConsent) return
    initPostHog()
  }, [hasConsent])

  // Identify on login, reset on logout. Only run when consent is
  // granted — `identifyUser` would otherwise no-op (the SDK isn't
  // initialised yet), but we keep the guard explicit so the call
  // graph reads as "no telemetry leaves the browser before consent".
  useEffect(() => {
    if (!hasConsent) return
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
  }, [
    hasConsent,
    session?.user?.id,
    session?.user?.role,
    session?.user?.email_verified,
    session?.user?.referrer_enabled,
    session?.organization?.id,
    session?.organization?.type,
    session?.organization?.member_role,
  ])

  return null
}
