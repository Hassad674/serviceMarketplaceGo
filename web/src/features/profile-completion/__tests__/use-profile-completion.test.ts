import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { createElement } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: vi.fn(),
}))

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: vi.fn(),
  API_BASE_URL: "http://localhost:8080",
}))

import { useCurrentUserId } from "@/shared/hooks/use-current-user-id"
import { apiClient } from "@/shared/lib/api-client"
import {
  profileCompletionQueryKey,
  useProfileCompletion,
} from "../hooks/use-profile-completion"

const mockedUseCurrentUserId = vi.mocked(useCurrentUserId)
const mockedApiClient = vi.mocked(apiClient)

function createWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return createElement(QueryClientProvider, { client }, children)
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("profileCompletionQueryKey", () => {
  it("returns a user-scoped key with the default persona suffix", () => {
    expect(profileCompletionQueryKey("user-1")).toEqual([
      "user",
      "user-1",
      "profile-completion",
      "default",
    ])
  })

  it("appends the persona to the key when provided", () => {
    expect(profileCompletionQueryKey("user-1", "referrer")).toEqual([
      "user",
      "user-1",
      "profile-completion",
      "referrer",
    ])
    expect(profileCompletionQueryKey("user-1", "freelance")).toEqual([
      "user",
      "user-1",
      "profile-completion",
      "freelance",
    ])
  })
})

describe("useProfileCompletion", () => {
  it("fetches the report and calls /api/v1/me/profile/completion", async () => {
    mockedUseCurrentUserId.mockReturnValue("user-1")
    mockedApiClient.mockResolvedValue({
      role: "provider",
      persona: "freelance",
      percent: 60,
      total_sections: 10,
      filled_sections: 6,
      sections: [],
    })

    const { result } = renderHook(() => useProfileCompletion(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockedApiClient).toHaveBeenCalledWith("/api/v1/me/profile/completion")
    expect(result.current.data?.percent).toBe(60)
  })

  it("forwards the persona override on the query string", async () => {
    mockedUseCurrentUserId.mockReturnValue("user-3")
    mockedApiClient.mockResolvedValue({
      role: "provider",
      persona: "referrer",
      percent: 25,
      total_sections: 8,
      filled_sections: 2,
      sections: [],
    })

    const { result } = renderHook(() => useProfileCompletion("referrer"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    expect(mockedApiClient).toHaveBeenCalledWith(
      "/api/v1/me/profile/completion?persona=referrer",
    )
    expect(result.current.data?.persona).toBe("referrer")
  })

  it("caches freelance and referrer reports under different keys", async () => {
    mockedUseCurrentUserId.mockReturnValue("user-4")
    mockedApiClient.mockImplementation(async (path: string) => ({
      role: "provider",
      persona: path.includes("referrer") ? "referrer" : "freelance",
      percent: 0,
      total_sections: 0,
      filled_sections: 0,
      sections: [],
    }))

    const wrapper = createWrapper()
    const both = renderHook(
      () => ({
        free: useProfileCompletion(),
        ref: useProfileCompletion("referrer"),
      }),
      { wrapper },
    )

    await waitFor(() => {
      expect(both.result.current.free.isSuccess).toBe(true)
      expect(both.result.current.ref.isSuccess).toBe(true)
    })

    // Both queries fire — they cache under different keys.
    expect(mockedApiClient).toHaveBeenCalledTimes(2)
    expect(both.result.current.free.data?.persona).toBe("freelance")
    expect(both.result.current.ref.data?.persona).toBe("referrer")
  })

  it("disables the query when no user id is available", async () => {
    mockedUseCurrentUserId.mockReturnValue(undefined)

    const { result } = renderHook(() => useProfileCompletion(), {
      wrapper: createWrapper(),
    })

    // Query is disabled — fetch is never called and isLoading stays false.
    expect(result.current.isLoading).toBe(false)
    expect(mockedApiClient).not.toHaveBeenCalled()
  })

  it("propagates API errors", async () => {
    mockedUseCurrentUserId.mockReturnValue("user-2")
    mockedApiClient.mockRejectedValue(new Error("kaboom"))

    const { result } = renderHook(() => useProfileCompletion(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("kaboom")
  })
})
