"use client"

import { useAuth } from "@/shared/hooks/use-auth"
import { useRouter } from "next/navigation"
import { useEffect } from "react"
import { DashboardShell } from "@/shared/components/layouts/dashboard-shell"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  const { isAuthenticated } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (!isAuthenticated()) {
      router.push("/login")
    }
  }, [isAuthenticated, router])

  if (!isAuthenticated()) return null

  return <DashboardShell>{children}</DashboardShell>
}
