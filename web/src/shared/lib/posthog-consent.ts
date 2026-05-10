// RGPD-friendly consent persistence — thin glue between the
// vanilla-cookieconsent CMP and the analytics SDKs (PostHog + GA4).
//
// Design after Phase A.2:
//   - The CMP (`vanilla-cookieconsent`) is now the source of truth
//     for the user's choice. It persists its own cookie + localStorage
//     state and dispatches our `analytics:consent-changed` event on
//     every flip.
//   - `readConsent()` reflects the CMP analytics-category status into
//     a `"accepted" | "refused" | null` triplet that legacy callers
//     (GA4 provider, tests) consume. `null` means "no choice yet".
//   - `applyConsent()` is preserved for tests / programmatic flows
//     (settings page revoke, etc.). It writes to localStorage,
//     toggles the PostHog SDK flag, fires `analytics:consent-changed`,
//     and POSTs the consent receipt to the backend (Phase A.3 wiring).
//
// Why we still keep a localStorage flag in addition to the CMP cookie:
//   - It is the legacy contract for components that mounted before
//     A.2 (GA4 provider already reads it). Keeping it as a mirror of
//     the CMP state lets us swap the banner without touching every
//     analytics consumer in the same dispatch.
//   - The CMP cookie name (`cc_cookie`) is owned by the library and
//     could change between major versions; the legacy key shields
//     consumers from that churn.
//
// Why we don't tear out the legacy key entirely: scope discipline.
// The brief is "swap the banner, gate PostHog + GA4 on the new CMP" —
// not "rewrite every analytics consumer". The migration to the CMP
// API as primary will land in a follow-up.

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
 *
 * Also:
 *   - dispatches a same-tab `analytics:consent-changed` event so
 *     other analytics providers (e.g. GA4) can re-render their
 *     conditional mounts without a full reload.
 *   - fires-and-forgets a POST /api/v1/consent/log with the chosen
 *     categories so we keep server-side proof of consent (Phase A.3
 *     of gdpr-roadmap.md). Failure is silent: the localStorage flip
 *     is the source of truth in the browser; the server log is the
 *     audit trail that backs CNIL inquiries.
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
  // Notify any provider listening for consent flips in the same tab
  // (the standard `storage` event only fires across tabs).
  try {
    window.dispatchEvent(new CustomEvent("analytics:consent-changed"))
  } catch {
    // best-effort — old browsers without CustomEvent ctor still get
    // the persistence + posthog flip.
  }

  void recordConsentOnServer(choice, choice === "accepted" ? "accept_all" : "refuse_all")
}

/**
 * Persist a custom (per-category) choice. Used by the CMP "save
 * preferences" path where the user accepted a strict subset of
 * categories — analytics may be off or on independently of "accept
 * all" / "refuse all".
 *
 * `analyticsAccepted` drives the PostHog opt-in flag and the legacy
 * localStorage mirror; `categories` is forwarded as-is to the
 * consent_log audit trail so the server has the full picture.
 */
export function applyCustomConsent(
  analyticsAccepted: boolean,
  categories: readonly string[],
): void {
  if (typeof window === "undefined") return
  const legacy: ConsentChoice = analyticsAccepted ? "accepted" : "refused"
  try {
    window.localStorage.setItem(STORAGE_KEY, legacy)
  } catch {
    // best-effort
  }
  initPostHog()
  if (analyticsAccepted) {
    posthog.opt_in_capturing()
  } else {
    posthog.opt_out_capturing()
  }
  try {
    window.dispatchEvent(new CustomEvent("analytics:consent-changed"))
  } catch {
    // best-effort
  }
  void recordConsentOnServer(legacy, "custom", categories)
}

// recordConsentOnServer fires the POST to /api/v1/consent/log. Pulled
// to a private helper so applyConsent stays a side-effect choreographer
// and the network call can be tested / mocked independently.
function recordConsentOnServer(
  choice: ConsentChoice,
  action: "accept_all" | "refuse_all" | "custom",
  customCategories?: readonly string[],
): Promise<void> {
  // Default category map mirrors the CMP runtime. When the caller
  // passes its own list (custom save), we forward it unchanged.
  const categories =
    customCategories && customCategories.length > 0
      ? Array.from(customCategories)
      : choice === "accepted"
        ? ["necessary", "analytics"]
        : ["necessary"]
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"
  const body = JSON.stringify({ action, categories })

  return fetch(`${apiUrl}/api/v1/consent/log`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body,
  })
    .then(() => undefined)
    .catch(() => undefined)
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
