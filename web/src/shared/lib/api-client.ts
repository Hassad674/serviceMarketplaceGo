const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8083"

type RequestOptions = {
  method?: string
  body?: unknown
  headers?: Record<string, string>
}

export async function apiClient<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = "GET", body, headers = {} } = options

  const res = await fetch(`${API_URL}${path}`, {
    method,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...headers,
    },
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
