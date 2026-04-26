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
    const parsed = await res.json().catch(() => null) as ApiErrorBody | null
    // Backwards-compatible code/message extraction:
    //   - canonical envelope: { error: { code, message } }
    //   - legacy flat shape:  { error: "code", message: "…" }
    // The full parsed body is preserved in `body` so callers (e.g.
    // the billing-profile gate) can read sibling fields like
    // `missing_fields` straight off the 403 envelope without
    // round-tripping the API.
    let code = "unknown_error"
    let message = "An error occurred"
    if (parsed) {
      if (parsed.error && typeof parsed.error === "object") {
        code = parsed.error.code ?? code
        message = parsed.error.message ?? message
      } else if (typeof parsed.error === "string") {
        code = parsed.error
        if (parsed.message) message = parsed.message
      } else if (parsed.message) {
        message = parsed.message
      }
    }
    throw new ApiError(res.status, code, message, parsed)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}

export type ApiErrorBody = {
  error?: string | { code?: string; message?: string }
  message?: string
  missing_fields?: unknown
  [key: string]: unknown
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: string,
    message: string,
    /** Parsed JSON envelope returned by the backend, when available. */
    public body: ApiErrorBody | null = null,
  ) {
    super(message)
    this.name = "ApiError"
  }
}
