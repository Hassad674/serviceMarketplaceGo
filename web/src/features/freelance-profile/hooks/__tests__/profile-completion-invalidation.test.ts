import { describe, expect, it, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { createElement } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: vi.fn(() => "user-1"),
}))

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: vi.fn(),
  API_BASE_URL: "http://localhost:8080",
}))

import { apiClient } from "@/shared/lib/api-client"
import { useUpsertFreelancePricing } from "../use-upsert-freelance-pricing"
import { useDeleteFreelancePricing } from "../use-delete-freelance-pricing"
import {
  useUploadFreelanceVideo,
  useDeleteFreelanceVideo,
} from "../use-freelance-video"
import {
  useUpsertFreelanceSocialLink,
  useDeleteFreelanceSocialLink,
} from "../use-freelance-social-links"
import { profileCompletionQueryKey } from "@/features/profile-completion/hooks/use-profile-completion"

const mockedApiClient = vi.mocked(apiClient)

// Each mutation hook must invalidate the profile-completion cache so
// the sidebar progress bar updates without a page reload. The contract
// is shared across every editor in the freelance feature — one
// regression here breaks the live-refresh promise everywhere.
function createWrapper() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  // Pre-fill the completion cache with a stale marker so we can detect
  // the invalidation by observing the state transition.
  client.setQueryData(profileCompletionQueryKey("user-1"), {
    role: "provider",
    persona: "freelance",
    percent: 0,
    total_sections: 11,
    filled_sections: 0,
    sections: [],
  })
  return {
    client,
    wrapper: function Wrapper({ children }: { children: React.ReactNode }) {
      return createElement(QueryClientProvider, { client }, children)
    },
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

async function expectInvalidated(client: QueryClient) {
  await waitFor(() => {
    const state = client.getQueryState(profileCompletionQueryKey("user-1"))
    // invalidateQueries marks the entry as `isInvalidated = true`
    // even when no observer is mounted to refetch it. That flag is
    // the contract we rely on.
    expect(state?.isInvalidated).toBe(true)
  })
}

describe("freelance-profile mutation hooks invalidate profile-completion", () => {
  it("useUpsertFreelancePricing fans out to the completion cache", async () => {
    const { client, wrapper } = createWrapper()
    mockedApiClient.mockResolvedValue({})

    const { result } = renderHook(() => useUpsertFreelancePricing(), { wrapper })
    await result.current.mutateAsync({
      type: "daily",
      min_amount: 500,
      max_amount: null,
      currency: "EUR",
      note: "",
      negotiable: false,
    })
    await expectInvalidated(client)
  })

  it("useDeleteFreelancePricing fans out to the completion cache", async () => {
    const { client, wrapper } = createWrapper()
    mockedApiClient.mockResolvedValue({})

    const { result } = renderHook(() => useDeleteFreelancePricing(), { wrapper })
    await result.current.mutateAsync()
    await expectInvalidated(client)
  })

  it("useUploadFreelanceVideo fans out to the completion cache", async () => {
    const { client, wrapper } = createWrapper()
    // Video API uses raw fetch (multipart form) — apiClient is not in
    // the call path. Stub fetch directly so the success branch fires.
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(JSON.stringify({ video_url: "u" }), {
        status: 200,
        headers: { "content-type": "application/json" },
      }),
    )

    const { result } = renderHook(() => useUploadFreelanceVideo(), { wrapper })
    await result.current.mutateAsync(new File([], "v.mp4"))
    await expectInvalidated(client)
    fetchSpy.mockRestore()
  })

  it("useDeleteFreelanceVideo fans out to the completion cache", async () => {
    const { client, wrapper } = createWrapper()
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockResolvedValue(
      new Response(null, { status: 204 }),
    )

    const { result } = renderHook(() => useDeleteFreelanceVideo(), { wrapper })
    await result.current.mutateAsync()
    await expectInvalidated(client)
    fetchSpy.mockRestore()
  })

  it("useUpsertFreelanceSocialLink fans out to the completion cache", async () => {
    const { client, wrapper } = createWrapper()
    mockedApiClient.mockResolvedValue({})

    const { result } = renderHook(() => useUpsertFreelanceSocialLink(), { wrapper })
    await result.current.mutateAsync({
      platform: "twitter",
      url: "https://twitter.com/me",
    })
    await expectInvalidated(client)
  })

  it("useDeleteFreelanceSocialLink fans out to the completion cache", async () => {
    const { client, wrapper } = createWrapper()
    mockedApiClient.mockResolvedValue({})

    const { result } = renderHook(() => useDeleteFreelanceSocialLink(), { wrapper })
    await result.current.mutateAsync("twitter")
    await expectInvalidated(client)
  })
})
