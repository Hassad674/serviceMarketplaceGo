// Empty NEXT_PUBLIC_API_URL = use proxy (production on Vercel).
// Set NEXT_PUBLIC_API_URL = "http://localhost:8083" for local development.
const rawApiUrl = process.env.NEXT_PUBLIC_API_URL || ""

/** HTTP base URL — empty in production (proxy), full URL in dev. */
export const API_BASE_URL = rawApiUrl

/** WebSocket base URL — always the real backend (no proxy for WS). */
export const WS_BASE_URL = rawApiUrl ? rawApiUrl.replace(/^http/, "ws") : ""

const API_URL = API_BASE_URL

type RequestOptions = {
  method?: string
  body?: unknown
  headers?: Record<string, string>
  signal?: AbortSignal
}

export async function apiClient<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, headers = {}, signal } = options

  const res = await fetch(`${API_URL}${path}`, {
    method,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...headers,
    },
    signal,
    ...(body ? { body: JSON.stringify(body) } : {}),
  })

  if (!res.ok) {
    const error = await res.json().catch(() => ({ message: "An error occurred" }))
    throw new ApiError(res.status, error.error || "unknown_error", error.message || "An error occurred")
  }

  if (res.status === 204) return undefined as T
  return res.json()
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
  ) {
    super(message)
    this.name = "ApiError"
  }
}
