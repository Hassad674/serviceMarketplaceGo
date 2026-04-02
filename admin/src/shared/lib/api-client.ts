const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080"

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

export async function adminApi<T>(
  path: string,
  options: RequestOptions = {},
): Promise<T> {
  const token = localStorage.getItem("admin_token")
  const { method = "GET", body, headers = {} } = options

  const res = await fetch(`${API_URL}${path}`, {
    method,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
    ...(body ? { body: JSON.stringify(body) } : {}),
  })

  if (res.status === 401) {
    localStorage.removeItem("admin_token")
    window.location.href = "/login"
    throw new ApiError(401, "unauthorized", "Session expirée")
  }

  if (!res.ok) {
    const error = await res.json().catch(() => ({ code: "unknown", message: "Erreur" }))
    throw new ApiError(res.status, error.code || "unknown", error.message || "Erreur")
  }

  if (res.status === 204) return undefined as T
  return res.json()
}
