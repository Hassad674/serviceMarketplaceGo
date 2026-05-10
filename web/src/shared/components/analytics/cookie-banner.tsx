"use client"

import { useEffect, useState } from "react"
import { useTranslations } from "next-intl"

import { Button } from "@/shared/components/ui/button"
import { applyConsent, readConsent } from "@/shared/lib/posthog-consent"

/**
 * RGPD-compliant analytics consent banner.
 *
 * - Renders a sticky bottom strip on first visit only.
 * - "Accepter" calls `applyConsent("accepted")` which flips PostHog
 *   into capturing-on mode and persists the choice in localStorage.
 * - "Refuser" does the opposite — captures stay disabled for the
 *   rest of the session and across reloads.
 * - When `readConsent()` already returns a value, the banner is
 *   never mounted, so no flash on subsequent visits.
 *
 * Tutoiement FR is mandatory per project lang policy. All strings are
 * keyed in `messages/{fr,en}.json` under `analyticsConsent`.
 */
export function CookieBanner() {
  const t = useTranslations("analyticsConsent")
  const [shouldShow, setShouldShow] = useState(false)

  // Read persisted consent on mount (avoid SSR mismatch).
  useEffect(() => {
    setShouldShow(readConsent() === null)
  }, [])

  function handle(choice: "accepted" | "refused") {
    applyConsent(choice)
    setShouldShow(false)
  }

  if (!shouldShow) return null

  return (
    <div
      role="dialog"
      aria-live="polite"
      aria-label={t("ariaLabel")}
      className="fixed bottom-4 left-4 right-4 z-50 mx-auto max-w-3xl rounded-2xl border border-border bg-card p-4 shadow-lg sm:p-5"
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p className="text-sm text-muted-foreground sm:max-w-xl">{t("description")}</p>
        <div className="flex shrink-0 items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => handle("refused")}>
            {t("refuse")}
          </Button>
          <Button variant="primary" size="sm" onClick={() => handle("accepted")}>
            {t("accept")}
          </Button>
        </div>
      </div>
    </div>
  )
}
