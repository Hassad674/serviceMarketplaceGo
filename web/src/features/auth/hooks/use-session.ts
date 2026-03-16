"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"
import { useAuth } from "@/shared/hooks/use-auth"

export function useSession() {
  const router = useRouter()
  const { user, accessToken, isAuthenticated } = useAuth()

  useEffect(() => {
    if (!isAuthenticated()) {
      router.push("/login")
    }
  }, [isAuthenticated, router])

  return { user, accessToken }
}
