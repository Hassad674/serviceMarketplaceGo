"use client"

import { create } from "zustand"
import { persist } from "zustand/middleware"

type User = {
  id: string
  email: string
  first_name: string
  last_name: string
  display_name: string
  role: "agency" | "enterprise" | "provider"
  referrer_enabled: boolean
}

type AuthState = {
  user: User | null
  accessToken: string | null
  refreshToken: string | null
  setAuth: (user: User, accessToken: string, refreshToken: string) => void
  logout: () => void
  isAuthenticated: () => boolean
}

export const useAuth = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      setAuth: (user, accessToken, refreshToken) => {
        set({ user, accessToken, refreshToken })
        // Set cookie for Next.js middleware (route protection)
        document.cookie = `access_token=${accessToken}; path=/; max-age=${60 * 60 * 24 * 7}; SameSite=Lax`
      },
      logout: () => {
        set({ user: null, accessToken: null, refreshToken: null })
        // Clear cookie
        document.cookie = "access_token=; path=/; max-age=0"
      },
      isAuthenticated: () => !!get().accessToken,
    }),
    { name: "marketplace-auth" },
  ),
)
