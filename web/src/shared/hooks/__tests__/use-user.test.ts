import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useUser, useLogout } from "../use-user"

const mockFetch = vi.fn()
vi.stubGlobal("fetch", mockFetch)

// Mock window.location
const originalLocation = window.location
beforeEach(() => {
  Object.defineProperty(window, "location", {
    writable: true,
    value: { ...originalLocation, href: "" },
  })
})

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe("useUser", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("fetches current user from /api/v1/auth/me", async () => {
    const user = {
      id: "user-1",
      email: "test@example.com",
      first_name: "Test",
      last_name: "User",
      display_name: "Test User",
      role: "provider",
      referrer_enabled: false,
      email_verified: true,
      created_at: "2026-03-20T10:00:00Z",
    }
    mockFetch.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve(user),
    })

    const { result } = renderHook(() => useUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(user)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/auth/me"),
      expect.objectContaining({ credentials: "include" }),
    )
  })

  it("handles not authenticated error", async () => {
    mockFetch.mockResolvedValue({ ok: false, status: 401 })

    const { result } = renderHook(() => useUser(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe("useLogout", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls logout endpoint and redirects to /login", async () => {
    mockFetch.mockResolvedValue({ ok: true })

    const { result } = renderHook(() => useLogout(), {
      wrapper: createWrapper(),
    })

    await result.current()

    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/auth/logout"),
      expect.objectContaining({ method: "POST", credentials: "include" }),
    )
    expect(window.location.href).toBe("/login")
  })
})
