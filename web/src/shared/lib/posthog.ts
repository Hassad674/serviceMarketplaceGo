// PostHog browser SDK glue. Centralised so every component reaches a
// single, idempotent init helper instead of importing posthog-js
// directly in random places.
//
// Three responsibilities:
//   1. Read the env (NEXT_PUBLIC_POSTHOG_KEY / NEXT_PUBLIC_POSTHOG_HOST).
//   2. Provide an idempotent initPostHog() that no-ops on the server,
//      no-ops without a key, and silences re-init storms in dev hot
//      reloads.
//   3. Wrap the SDK's capture/identify/group/reset calls with type-
//      safe helpers so feature code never imports posthog-js directly.
//      This also makes the surface mockable from vitest without
//      rewiring globals.
//
// RGPD posture: capture is opt-IN by default in production. The cookie
// banner toggles posthog.opt_in_capturing() / opt_out_capturing();
// initPostHog() forces the SDK into "capturing disabled" mode until
// the user makes a choice so we never ship an uncovenanted hit.

import posthog from "posthog-js"

const PUBLIC_AUTH_PATHS = ["/login", "/register", "/forgot-password", "/reset-password"]

/**
 * Public env shape consumed by the SDK. `isEnabled` is the single
 * predicate every caller should branch on — both the env vars must be
 * present AND we must be running in the browser. SSR always returns
 * `isEnabled: false`.
 */
export type PostHogConfig = {
  apiKey: string | undefined
  apiHost: string
  isEnabled: boolean
}

const DEFAULT_HOST = "https://eu.posthog.com"

/** Read the env once and freeze the result for the rest of the session. */
export function getPostHogConfig(): PostHogConfig {
  const apiKey = process.env.NEXT_PUBLIC_POSTHOG_KEY
  const apiHost = process.env.NEXT_PUBLIC_POSTHOG_HOST || DEFAULT_HOST
  const isEnabled =
    typeof window !== "undefined" && Boolean(apiKey) && apiKey !== ""
  return { apiKey, apiHost, isEnabled }
}

let _initialized = false

/**
 * Idempotent initializer. Safe to call from a `useEffect` that may
 * fire twice in React 19 strict mode without buffering duplicate
 * SDKs. Returns the underlying posthog instance for callers that
 * want the raw object (typically tests).
 */
export function initPostHog(): typeof posthog | null {
  const cfg = getPostHogConfig()
  if (!cfg.isEnabled || !cfg.apiKey) return null
  if (_initialized) return posthog

  posthog.init(cfg.apiKey, {
    api_host: cfg.apiHost,
    // Pageviews are auto-captured by the SDK on every history-api
    // navigation. We disable the synthetic /pageleave because the
    // marketplace is a SPA — every Next.js navigation already fires
    // pageview, and pageleave double-counts when the user backs out.
    capture_pageview: true,
    capture_pageleave: false,
    // RGPD: never auto-capture until the user opts in via the
    // cookie banner. The banner persists the choice in localStorage
    // and toggles posthog.opt_in_capturing()/opt_out_capturing() —
    // see lib/posthog-consent.ts.
    opt_out_capturing_by_default: true,
    // Persist the distinct id only after consent. Until then the SDK
    // keeps state in memory only.
    persistence: "memory",
    // Disable the autocapture click-tracking — we use explicit
    // capture() calls at strategic points so dashboards have stable
    // event names instead of ".btn-primary [click]" strings.
    autocapture: false,
    // Session recording is disabled by default. Toggle in the
    // PostHog UI per environment if/when the team wants it; the
    // browser SDK will respect the server-side config.
    disable_session_recording: true,
    // Suppress the SDK's internal console.log spam in production.
    loaded: (instance) => {
      if (process.env.NODE_ENV === "development") {
        instance.debug(false)
      }
    },
    // RGPD-friendly: do not collect IP addresses (PostHog still
    // computes country from the request header but never persists
    // the raw IP).
    ip: false,
  })
  _initialized = true
  return posthog
}

/**
 * Capture an analytics event. No-ops gracefully when:
 *   1. The SDK has not been initialised (server, missing key, etc.).
 *   2. The user opted out of analytics.
 */
export function captureEvent(
  event: string,
  properties?: Record<string, unknown>,
): void {
  const ph = initPostHog()
  if (!ph) return
  // posthog.capture itself respects the opt-out flag, but checking
  // explicitly keeps the call site grep-able and avoids surprise
  // captures during dev when the banner hasn't been clicked yet.
  if (ph.has_opted_out_capturing()) return
  ph.capture(event, properties)
}

/**
 * Attach profile attributes to the current distinct id and switch
 * the SDK into "logged in" mode. Always safe to call multiple times
 * — PostHog dedupes by distinct id.
 */
export function identifyUser(
  distinctId: string,
  properties?: Record<string, unknown>,
): void {
  const ph = initPostHog()
  if (!ph) return
  // identify is allowed even when capture is opted out — we still
  // need the id when consent is granted later in the session.
  ph.identify(distinctId, properties)
}

/**
 * Attach attributes to a group (typically organization). Lets
 * PostHog dashboards filter by "events from organizations on plan
 * X" without us shipping the plan attribute on every event.
 */
export function setOrganizationGroup(
  organizationId: string,
  properties?: Record<string, unknown>,
): void {
  const ph = initPostHog()
  if (!ph) return
  ph.group("organization", organizationId, properties)
}

/**
 * Tear down the user dimension on logout. Resets the distinct id so
 * subsequent anonymous events do not pollute the previous user's
 * timeline.
 */
export function resetPostHog(): void {
  if (!_initialized) return
  posthog.reset()
}

/**
 * Public path predicate used by the providers tree to decide whether
 * to attempt an identify on a 401 from /auth/me — auth pages
 * legitimately return 401 for non-authenticated visitors.
 */
export function isOnPublicAuthPath(pathname: string): boolean {
  return PUBLIC_AUTH_PATHS.some((p) => pathname === p || pathname.startsWith(`${p}/`))
}
