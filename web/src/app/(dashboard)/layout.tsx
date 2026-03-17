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
  const { accessToken } = useAuth()
  const router = useRouter()

  useEffect(() => {
    if (!accessToken) {
      router.push("/login")
    }
  }, [accessToken, router])

  if (!accessToken) return null

  return <DashboardShell>{children}</DashboardShell>
}
