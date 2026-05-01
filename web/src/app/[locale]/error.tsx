"use client"

import { useEffect } from "react"
import { useTranslations } from "next-intl"
import { Link } from "@i18n/navigation"

import { Button } from "@/shared/components/ui/button"
// Locale-scoped error boundary (PERF-W-03 + QUAL-W-01). Catches any
// rendering or data-fetching error that escapes a route's own
// error.tsx. Server logs the error so the team is alerted; the user
// gets a localised message with a retry button.
//
// Next.js requires this file to be a Client Component because it
// receives the `reset()` callback to re-trigger the failed route
// segment.
export default function LocaleError({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  const t = useTranslations("boundary")

  useEffect(() => {
    // The error.digest is the only safe correlation handle to ship
    // to logging without leaking stack traces to the client. We log
    // a single line with the digest so server logs can join with
    // the original traceback.
    console.error("[error-boundary]", {
      message: error.message,
      digest: error.digest,
    })
  }, [error])

  return (
    <div
      role="alert"
      aria-live="assertive"
      className="mx-auto flex min-h-[60vh] max-w-md flex-col items-center justify-center gap-4 p-6 text-center"
    >
      <h1 className="text-2xl font-bold">{t("errorTitle")}</h1>
      <p className="text-sm text-muted-foreground">{t("errorDescription")}</p>
      <div className="mt-2 flex gap-3">
        <Button variant="ghost" size="auto"
          type="button"
          onClick={reset}
          className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white hover:opacity-90"
        >
          {t("errorRetry")}
        </Button>
        <Link
          href="/"
          className="rounded-lg border border-border px-4 py-2 text-sm font-medium hover:bg-muted"
        >
          {t("errorHome")}
        </Link>
      </div>
    </div>
  )
}
