"use client"

import { create } from "zustand"
import { persist } from "zustand/middleware"
import { useEffect, useState } from "react"

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
  _hydrated: boolean
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
      _hydrated: false,
      setAuth: (user, accessToken, refreshToken) => {
        set({ user, accessToken, refreshToken })
        document.cookie = `access_token=${accessToken}; path=/; max-age=${60 * 60 * 24 * 7}; SameSite=Lax`
      },
      logout: () => {
        set({ user: null, accessToken: null, refreshToken: null })
        document.cookie = "access_token=; path=/; max-age=0"
      },
      isAuthenticated: () => !!get().accessToken,
    }),
    {
      name: "marketplace-auth",
      onRehydrateStorage: () => () => {
        useAuth.setState({ _hydrated: true })
      },
    },
  ),
)

/** Hook that waits for Zustand persist hydration before returning auth state */
export function useAuthReady() {
  const store = useAuth()
  // Subscribe to hydration changes — check both current state and future updates
  const [ready, setReady] = useState(() => useAuth.getState()._hydrated)

  useEffect(() => {
    // If already hydrated, set immediately
    if (useAuth.getState()._hydrated) {
      setReady(true)
      return
    }
    // Otherwise subscribe to changes
    const unsub = useAuth.subscribe((state) => {
      if (state._hydrated) {
        setReady(true)
        unsub()
      }
    })
    return unsub
  }, [])

  return { ...store, ready }
}
