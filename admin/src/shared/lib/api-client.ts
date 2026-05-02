import { getAuthToken, clearAuthToken } from "@/shared/stores/auth-store"

const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8083"

type RequestOptions = {
  method?: string
  body?: unknown
  headers?: Record<string, string>
}

export class ApiError extends Error {
  status: number
  code: string

  constructor(status: number, code: string, message: string) {
    super(message)
    this.name = "ApiError"
    this.status = status
    this.code = code
  }
}

// adminApi is the centralized fetch wrapper for the admin SPA. The
// bearer token is read from the in-memory Zustand store (NOT from
// localStorage — see `auth-store.ts` for the SEC-FINAL-07 rationale).
//
// Cookies are forwarded via `credentials: "include"` so the cookie-based
// session restore path on app boot (`/auth/me`) works without an
// Authorization header.
//
// On 401 we clear the token AND redirect to /login. The token clear
// runs BEFORE the navigation so that any in-flight retry from a parent
// query observer cannot replay the dead bearer.
export async function adminApi<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const token = getAuthToken()
  const { method = "GET", body, headers = {} } = options

  const res = await fetch(`${API_URL}${path}`, {
    method,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
    ...(body ? { body: JSON.stringify(body) } : {}),
  })

  if (res.status === 401) {
    clearAuthToken()
    if (typeof window !== "undefined" && window.location.pathname !== "/login") {
      window.location.href = "/login"
    }
    throw new ApiError(401, "unauthorized", "Session expirée")
  }

  if (!res.ok) {
    const error = await res.json().catch(() => ({ code: "unknown", message: "Erreur" }))
    throw new ApiError(res.status, error.code || "unknown", error.message || "Erreur")
  }

  if (res.status === 204) return undefined as T
  return res.json()
}
