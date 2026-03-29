"use client"

import { useQuery, useQueryClient } from "@tanstack/react-query"

import { API_BASE_URL } from "@/shared/lib/api-client"

const API_URL = API_BASE_URL

export type CurrentUser = {
  id: string
  email: string
  first_name: string
  last_name: string
  display_name: string
  role: "agency" | "enterprise" | "provider"
  referrer_enabled: boolean
  email_verified: boolean
  created_at: string
}

async function fetchCurrentUser(): Promise<CurrentUser> {
  const res = await fetch(`${API_URL}/api/v1/auth/me`, {
    credentials: "include",
  })
  if (!res.ok) throw new Error("Not authenticated")
  return res.json()
}

export function useUser() {
  return useQuery({
    queryKey: ["current-user"],
    queryFn: fetchCurrentUser,
    staleTime: 5 * 60 * 1000,
    retry: false,
  })
}

export function useLogout() {
  const queryClient = useQueryClient()

  return async function logout() {
    await fetch(`${API_URL}/api/v1/auth/logout`, {
      method: "POST",
      credentials: "include",
    })
    queryClient.clear()
    // Hard redirect to destroy all in-memory state (React tree, query cache,
    // WebSocket connections). router.push would keep stale data in memory.
    window.location.href = "/login"
  }
}
