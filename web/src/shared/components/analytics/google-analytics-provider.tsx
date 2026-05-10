"use client"

import { useEffect, useState } from "react"
import { GoogleAnalytics } from "@next/third-parties/google"

import { getGAConfig } from "@/shared/lib/ga"
import { readConsent } from "@/shared/lib/posthog-consent"

/**
 * GoogleAnalyticsProvider — mounts the official `<GoogleAnalytics>`
 * helper from `@next/third-parties/google` so the gtag.js script is
 * loaded with `next/script` strategy=`afterInteractive`.
 *
 * Two gates protect the render:
 *   1. `NEXT_PUBLIC_GA_MEASUREMENT_ID` must be a non-empty string —
 *      otherwise dev environments without the env var stay a no-op
 *      and never trigger a network call to googletagmanager.com.
 *   2. The user must have explicitly accepted analytics via the
 *      cookie banner (RGPD). This mirrors the PostHog gate at
 *      `shared/lib/posthog-consent.ts` so a single click toggles both
 *      SDKs in a single direction.
 *
 * The provider listens to the localStorage consent flag via a custom
 * `analytics:consent-changed` event so flipping the banner updates
 * the rendered tree without a hard reload. The cookie banner fires
 * that event from `applyConsent()`.
 */
export function GoogleAnalyticsProvider() {
  const { measurementId, isEnabled } = getGAConfig()
  const [hasConsent, setHasConsent] = useState(false)

  useEffect(() => {
    // Initial read on mount. SSR returns null so the banner is shown.
    setHasConsent(readConsent() === "accepted")

    function refresh() {
      setHasConsent(readConsent() === "accepted")
    }
    // Same-tab updates (cookie banner fires this).
    window.addEventListener("analytics:consent-changed", refresh)
    // Cross-tab updates via the standard storage event.
    window.addEventListener("storage", refresh)
    return () => {
      window.removeEventListener("analytics:consent-changed", refresh)
      window.removeEventListener("storage", refresh)
    }
  }, [])

  if (!isEnabled || !measurementId) return null
  if (!hasConsent) return null

  return <GoogleAnalytics gaId={measurementId} />
}
