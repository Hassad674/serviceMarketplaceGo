const API_URL = import.meta.env.VITE_API_URL || "http://localhost:8080"

export async function adminApi<T>(
  path: string,
  options: { method?: string; body?: unknown } = {},
): Promise<T> {
  const token = localStorage.getItem("admin_token")

  const res = await fetch(`${API_URL}${path}`, {
    method: options.method || "GET",
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    ...(options.body ? { body: JSON.stringify(options.body) } : {}),
  })

  if (res.status === 401) {
    localStorage.removeItem("admin_token")
    window.location.href = "/login"
    throw new Error("Unauthorized")
  }

  if (!res.ok) {
    const error = await res.json().catch(() => ({ message: "Erreur" }))
    throw new Error(error.message)
  }

  if (res.status === 204) return undefined as T
  return res.json()
}
