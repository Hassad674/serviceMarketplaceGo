import { cookies } from "next/headers"

export type SessionUser = {
  id: string
  email: string
  first_name: string
  last_name: string
  display_name: string
  role: "agency" | "enterprise" | "provider"
  referrer_enabled: boolean
}

export async function getSessionRole(): Promise<string | null> {
  const cookieStore = await cookies()
  return cookieStore.get("user_role")?.value ?? null
}

export async function isAuthenticated(): Promise<boolean> {
  const cookieStore = await cookies()
  return !!cookieStore.get("session_id")?.value
}

export async function isReferrerWorkspace(): Promise<boolean> {
  const cookieStore = await cookies()
  return cookieStore.get("workspace")?.value === "referrer"
}
