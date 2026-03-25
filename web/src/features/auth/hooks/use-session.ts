"use client"

import { useEffect } from "react"
import { useRouter } from "@i18n/navigation"
import { useAuth } from "@/shared/hooks/use-auth"

export function useSession() {
  const router = useRouter()
  const { user, accessToken } = useAuth()

  useEffect(() => {
    if (!accessToken) {
      router.push("/login")
    }
  }, [accessToken, router])

  return { user, accessToken }
}
