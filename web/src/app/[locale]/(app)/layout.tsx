"use client"

import { usePathname, useSearchParams } from "next/navigation"
import { DashboardShell } from "@/shared/components/layouts/dashboard-shell"

// Paths under the (app) group that must render WITHOUT the dashboard
// chrome (sidebar + top header). The /projects/pay checkout flow is
// the canonical example: a focused, single-task page where the
// dashboard navigation would be visual noise ("deux navbar" bug). The
// nested segment layout (`projects/pay/layout.tsx`) supplies its own
// minimal shell — this list lets the parent layout step out of the
// way so the nested layout is the sole chrome on screen.
const CHROMELESS_PATH_SEGMENTS = ["/projects/pay"] as const

function isChromelessPath(pathname: string): boolean {
  // Strip the locale prefix (e.g. /fr, /en) before comparing. The
  // (app) group is itself locale-namespaced via the [locale] dynamic
  // segment, so pathnames always carry the prefix at runtime.
  const withoutLocale = pathname.replace(/^\/[a-z]{2}(?=\/|$)/, "")
  return CHROMELESS_PATH_SEGMENTS.some(
    (segment) =>
      withoutLocale === segment || withoutLocale.startsWith(`${segment}/`),
  )
}

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const searchParams = useSearchParams()
  const pathname = usePathname()
  const isEmbedded = searchParams.get("embedded") === "true"

  if (isEmbedded) {
    return <div className="min-h-screen bg-background px-2">{children}</div>
  }

  // Path-scoped opt-out: chromeless segments (checkout, etc.) render
  // their own minimal shell from a nested segment layout. We render
  // children directly here so the nested layout is the only chrome.
  if (isChromelessPath(pathname)) {
    return <>{children}</>
  }

  return <DashboardShell>{children}</DashboardShell>
}
