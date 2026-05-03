import { apiClient } from "@/shared/lib/api-client"

import type { Post } from "@/shared/lib/api-paths"
import { API_BASE_URL } from "@/shared/lib/api-client"

const API_URL = API_BASE_URL

type AuthUser = {
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

/**
 * Web auth responses return the flat user object (session cookie is set via Set-Cookie header).
 * Mobile clients using X-Auth-Mode: token receive { user, access_token, refresh_token }.
 */
export type LoginError = {
  error: string
  message: string
  reason?: string
}

export class AuthApiError extends Error {
  code: string
  reason?: string

  constructor(code: string, message: string, reason?: string) {
    super(message)
    this.code = code
    this.reason = reason
  }
}

export async function login(email: string, password: string): Promise<AuthUser> {
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  })
  if (!res.ok) {
    const err: LoginError = await res.json().catch(() => ({ error: "unknown", message: "An error occurred" }))
    throw new AuthApiError(err.error, err.message || "Login failed", err.reason)
  }
  return res.json()
}

export async function register(data: {
  email: string
  password: string
  first_name?: string
  last_name?: string
  display_name?: string
  role: string
}): Promise<AuthUser> {
  const res = await fetch(`${API_URL}/api/v1/auth/register`, {
    method: "POST",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ message: "An error occurred" }))
    throw new Error(err.message || "Registration failed")
  }
  return res.json()
}

export async function forgotPassword(email: string): Promise<{ message: string }> {
  return apiClient<Post<"/api/v1/auth/forgot-password"> & { message: string }>("/api/v1/auth/forgot-password", {
    method: "POST",
    body: { email },
  })
}

export async function resetPassword(token: string, newPassword: string): Promise<{ message: string }> {
  return apiClient<Post<"/api/v1/auth/reset-password"> & { message: string }>("/api/v1/auth/reset-password", {
    method: "POST",
    body: { token, new_password: newPassword },
  })
}
