import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useSecurityActivity } from "../use-security-activity"

const listMock = vi.fn()

vi.mock("../../api/security-api", () => ({
  listSecurityActivity: (...args: unknown[]) => listMock(...args),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  Wrapper.displayName = "TestWrapper"
  return Wrapper
}

describe("useSecurityActivity", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("loads the first page on mount with no cursor", async () => {
    listMock.mockResolvedValue({
      data: [
        {
          id: "evt-1",
          action: "auth.login_success",
          ip_address: "203.0.113.4",
          user_agent_summary: "Ordinateur (Chrome 120)",
          access_kind: "desktop",
          created_at: "2026-05-08T12:00:00Z",
        },
      ],
      next_cursor: "next-token",
    })

    const { result } = renderHook(() => useSecurityActivity(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(listMock).toHaveBeenCalledTimes(1)
    expect(listMock.mock.calls[0][0]).toEqual({ cursor: undefined, limit: 20 })
    expect(result.current.data?.pages).toHaveLength(1)
    expect(result.current.hasNextPage).toBe(true)
  })

  it("fetches next page using the cursor returned by the first page", async () => {
    listMock
      .mockResolvedValueOnce({
        data: [],
        next_cursor: "page-2",
      })
      .mockResolvedValueOnce({
        data: [],
        next_cursor: "",
      })

    const { result } = renderHook(() => useSecurityActivity(), {
      wrapper: createWrapper(),
    })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    await act(async () => {
      await result.current.fetchNextPage()
    })

    expect(listMock).toHaveBeenCalledTimes(2)
    expect(listMock.mock.calls[1][0]).toEqual({ cursor: "page-2", limit: 20 })
  })
})
