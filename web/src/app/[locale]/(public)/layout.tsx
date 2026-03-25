"use client"

import { useAuthReady } from "@/shared/hooks/use-auth"
import { DashboardShell } from "@/shared/components/layouts/dashboard-shell"
import { PublicNavbar } from "@/shared/components/layouts/public-navbar"

export default function PublicLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { accessToken, ready } = useAuthReady()

  if (!ready) {
    return <PublicLayoutSkeleton />
  }

  if (accessToken) {
    return <DashboardShell>{children}</DashboardShell>
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <PublicNavbar />
      <main className="mx-auto max-w-7xl px-6 py-10">{children}</main>
    </div>
  )
}

function PublicLayoutSkeleton() {
  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-950">
      <header className="border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-gray-900">
        <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6">
          <div className="h-5 w-40 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
          <div className="flex items-center gap-4">
            <div className="h-4 w-16 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
            <div className="h-4 w-16 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
            <div className="h-4 w-16 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
          </div>
        </div>
      </header>
      <main className="mx-auto max-w-7xl px-6 py-10">
        <div className="space-y-4">
          <div className="h-8 w-64 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
          <div className="h-4 w-96 animate-pulse rounded bg-gray-200 dark:bg-gray-700" />
        </div>
      </main>
    </div>
  )
}
