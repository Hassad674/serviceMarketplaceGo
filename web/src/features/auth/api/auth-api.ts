import { apiClient } from "@/shared/lib/api-client"

export type AuthUser = {
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

export type AuthResponse = {
  user: AuthUser
  access_token: string
  refresh_token: string
}

export async function login(
  email: string,
  password: string,
): Promise<AuthResponse> {
  return apiClient<AuthResponse>("/api/v1/auth/login", {
    method: "POST",
    body: { email, password },
  })
}

export async function register(data: {
  email: string
  password: string
  first_name: string
  last_name: string
  display_name: string
  role: string
}): Promise<AuthResponse> {
  return apiClient<AuthResponse>("/api/v1/auth/register", {
    method: "POST",
    body: data,
  })
}

export async function refreshToken(
  refresh_token: string,
): Promise<AuthResponse> {
  return apiClient<AuthResponse>("/api/v1/auth/refresh", {
    method: "POST",
    body: { refresh_token },
  })
}

export async function getMe(token: string): Promise<AuthUser> {
  return apiClient<AuthUser>("/api/v1/auth/me", { token })
}
