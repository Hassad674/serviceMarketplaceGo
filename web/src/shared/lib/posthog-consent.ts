// RGPD-friendly consent persistence. The web app stores the user's
// analytics choice in localStorage so we never re-prompt across
// sessions, and translates that choice into the matching PostHog SDK
// flag (opt_in_capturing / opt_out_capturing).
//
// Why localStorage instead of a cookie: this signal is purely a
// browser-side preference — no server cares about it. Putting it in
// localStorage avoids the cookie-banner-about-cookies recursion and
// keeps the consent-tracking surface as small as possible.

import posthog from "posthog-js"

import { initPostHog } from "@/shared/lib/posthog"

const STORAGE_KEY = "marketplace.analytics.consent"

export type ConsentChoice = "accepted" | "refused"

/**
 * Read the persisted consent. Returns null when the user has not
 * made a choice yet — callers should show the banner in that case.
 */
export function readConsent(): ConsentChoice | null {
  if (typeof window === "undefined") return null
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY)
    if (raw === "accepted" || raw === "refused") return raw
    return null
  } catch {
    // Private browsing / disabled storage — assume "no choice".
    return null
  }
}

/**
 * Persist the user's choice and propagate it to PostHog.
 * Idempotent — calling it twice with the same choice is a no-op.
 */
export function applyConsent(choice: ConsentChoice): void {
  if (typeof window === "undefined") return
  try {
    window.localStorage.setItem(STORAGE_KEY, choice)
  } catch {
    // best-effort
  }
  // Make sure the SDK is up before toggling the flag.
  initPostHog()
  if (choice === "accepted") {
    posthog.opt_in_capturing()
  } else {
    posthog.opt_out_capturing()
  }
}

/** Helper for tests / settings page that lets a user revoke consent. */
export function clearConsent(): void {
  if (typeof window === "undefined") return
  try {
    window.localStorage.removeItem(STORAGE_KEY)
  } catch {
    // best-effort
  }
}
