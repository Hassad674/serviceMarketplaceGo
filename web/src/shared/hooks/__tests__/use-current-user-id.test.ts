import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook } from "@testing-library/react"
import { createElement } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

vi.mock("../use-user", () => ({
  useUser: vi.fn(),
}))

import { useUser } from "../use-user"
import { useCurrentUserId } from "../use-current-user-id"

const mockedUseUser = vi.mocked(useUser)

function createWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return createElement(QueryClientProvider, { client }, children)
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("useCurrentUserId", () => {
  it("returns user id when user data is available", () => {
    mockedUseUser.mockReturnValue({
      data: {
        id: "user-123",
        email: "test@example.com",
        first_name: "John",
        last_name: "Doe",
        display_name: "John Doe",
        role: "provider" as const,
        referrer_enabled: false,
        email_verified: true,
        created_at: "2026-01-01T00:00:00Z",
      },
    } as ReturnType<typeof useUser>)

    const { result } = renderHook(() => useCurrentUserId(), {
      wrapper: createWrapper(),
    })

    expect(result.current).toBe("user-123")
  })

  it("returns undefined when user data is not available", () => {
    mockedUseUser.mockReturnValue({
      data: undefined,
    } as ReturnType<typeof useUser>)

    const { result } = renderHook(() => useCurrentUserId(), {
      wrapper: createWrapper(),
    })

    expect(result.current).toBeUndefined()
  })

  it("returns undefined when query is loading", () => {
    mockedUseUser.mockReturnValue({
      data: undefined,
      isLoading: true,
    } as ReturnType<typeof useUser>)

    const { result } = renderHook(() => useCurrentUserId(), {
      wrapper: createWrapper(),
    })

    expect(result.current).toBeUndefined()
  })

  it("returns the correct id for different users", () => {
    mockedUseUser.mockReturnValue({
      data: {
        id: "agency-456",
        email: "agency@example.com",
        first_name: "Agency",
        last_name: "Test",
        display_name: "Agency Test",
        role: "agency" as const,
        referrer_enabled: false,
        email_verified: true,
        created_at: "2026-02-01T00:00:00Z",
      },
    } as ReturnType<typeof useUser>)

    const { result } = renderHook(() => useCurrentUserId(), {
      wrapper: createWrapper(),
    })

    expect(result.current).toBe("agency-456")
  })
})
