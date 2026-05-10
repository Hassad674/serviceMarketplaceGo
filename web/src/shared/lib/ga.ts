// Google Analytics 4 (GA4) browser glue.
//
// PostHog covers product analytics ("what did the user do"); GA4 covers
// acquisition analytics ("where did the user come from") via the
// Google Search Console integration. The two SDKs coexist — every
// conversion event is fired into both, see
// `shared/lib/analytics-events.ts`.
//
// We rely on the official `@next/third-parties/google` helper for
// loading the gtag script with `next/script` strategy=`afterInteractive`
// (~3 KB). The helper is ONLY mounted when:
//   1. NEXT_PUBLIC_GA_MEASUREMENT_ID is non-empty, AND
//   2. the user opted in via the analytics cookie banner.
//
// This keeps the dev surface a no-op without an ID and respects the
// existing RGPD posture without requiring Google Consent Mode v2
// plumbing.
//
// `sendGAEvent` from `@next/third-parties/google` is a thin wrapper
// around `window.gtag('event', ...)` — it no-ops gracefully when the
// script has not been loaded (so calling it before consent is granted
// is safe, just useless).

import { sendGAEvent as nextSendGAEvent } from "@next/third-parties/google"

/** Public env shape consumed by the GA4 provider. */
export type GAConfig = {
  measurementId: string | undefined
  isEnabled: boolean
}

/** Read the env once and freeze the result for the rest of the call. */
export function getGAConfig(): GAConfig {
  const measurementId = process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID
  const isEnabled =
    typeof window !== "undefined" &&
    typeof measurementId === "string" &&
    measurementId.length > 0
  return { measurementId, isEnabled }
}

/**
 * Fire a GA4 event. No-ops gracefully when:
 *   1. The SDK has not been loaded (server, missing env, consent not granted).
 *   2. `window.gtag` is undefined (script not yet loaded).
 *
 * The wrapper exists so feature code never imports
 * `@next/third-parties/google` directly — keeps the surface mockable
 * from vitest without rewiring the third-party module.
 */
export function captureGAEvent(
  name: string,
  properties?: Record<string, unknown>,
): void {
  if (typeof window === "undefined") return
  // The helper accepts variadic args; we always pass `event` + name + props.
  // Wrap in try/catch because gtag's queue can throw on pages where the
  // dataLayer was never bootstrapped (e.g., SSR rehydration races).
  try {
    nextSendGAEvent("event", name, properties ?? {})
  } catch {
    // best-effort, never crash the host page on an analytics call
  }
}
