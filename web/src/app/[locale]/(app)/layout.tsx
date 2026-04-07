"use client"

import { useSearchParams } from "next/navigation"
import { DashboardShell } from "@/shared/components/layouts/dashboard-shell"

export default function AppLayout({ children }: { children: React.ReactNode }) {
  const searchParams = useSearchParams()
  const isEmbedded = searchParams.get("embedded") === "true"

  if (isEmbedded) {
    return <div className="min-h-screen bg-background px-2">{children}</div>
  }

  return <DashboardShell>{children}</DashboardShell>
}
