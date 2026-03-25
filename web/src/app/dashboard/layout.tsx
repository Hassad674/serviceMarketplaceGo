"use client"

import { useAuthReady } from "@/shared/hooks/use-auth"
import { useRouter } from "next/navigation"
import { useEffect } from "react"
import { DashboardShell } from "@/shared/components/layouts/dashboard-shell"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { accessToken, ready } = useAuthReady()
  const router = useRouter()

  useEffect(() => {
    if (ready && !accessToken) {
      router.push("/login")
    }
  }, [accessToken, ready, router])

  // Wait for hydration before deciding
  if (!ready) {
    return (
      <div className="flex h-screen items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-primary border-t-transparent" />
      </div>
    )
  }

  if (!accessToken) return null

  return <DashboardShell>{children}</DashboardShell>
}
