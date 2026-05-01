"use client"

import { useEffect } from "react"

// global-error.tsx is the html-level error boundary (PERF-W-03).
// Triggered only when the locale-scoped error.tsx itself throws or
// the root layout fails to render — at that point next-intl's
// provider may not be available, so this page falls back to plain
// English copy and renders its own <html>/<body> wrapper.
//
// We deliberately keep this minimal: a pure HTML5 page with a single
// CTA. No theming, no fonts, no client libraries — the goal is to
// guarantee a graceful failure mode when everything else is broken.
export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string }
  reset: () => void
}) {
  useEffect(() => {
    console.error("[global-error-boundary]", {
      message: error.message,
      digest: error.digest,
    })
  }, [error])

  return (
    <html lang="en">
      <body
        style={{
          fontFamily:
            "system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
          background: "#f8fafc",
          color: "#0f172a",
          margin: 0,
          minHeight: "100vh",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          padding: "2rem",
        }}
      >
        <main
          role="alert"
          aria-live="assertive"
          style={{
            maxWidth: "28rem",
            textAlign: "center",
          }}
        >
          <h1 style={{ fontSize: "1.5rem", fontWeight: 700 }}>We hit a snag</h1>
          <p style={{ fontSize: "0.95rem", color: "#475569", marginTop: "0.5rem" }}>
            Refreshing the page usually fixes it. If it persists, please contact support.
          </p>
          <button
            type="button"
            onClick={reset}
            style={{
              marginTop: "1.5rem",
              background: "#f43f5e",
              color: "white",
              border: 0,
              borderRadius: "0.5rem",
              padding: "0.5rem 1rem",
              fontSize: "0.95rem",
              cursor: "pointer",
            }}
          >
            Try again
          </button>
        </main>
      </body>
    </html>
  )
}
