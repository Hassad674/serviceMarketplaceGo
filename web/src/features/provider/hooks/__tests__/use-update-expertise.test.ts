import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useUpdateExpertiseDomains } from "../use-update-expertise"
import { profileQueryKey } from "../use-profile"

const mockUpdateExpertiseDomains = vi.fn()

vi.mock("../../api/expertise-api", () => ({
  updateExpertiseDomains: (...args: unknown[]) =>
    mockUpdateExpertiseDomains(...args),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "test-user-id",
}))

function createWrapperAndClient() {
  // Use a non-zero gcTime so that data seeded with `setQueryData` for an
  // unobserved key (only written to, never read via `useQuery`) is not
  // garbage-collected before the mutation hook reads it back in
  // `onMutate` / `onSuccess`.
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 5 * 60 * 1000 },
      mutations: { retry: false },
    },
  })
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  return { queryClient, wrapper }
}

const baseProfile = {
  organization_id: "org-1",
  title: "Test",
  photo_url: "",
  presentation_video_url: "",
  referrer_video_url: "",
  about: "",
  referrer_about: "",
  expertise_domains: ["development"],
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T00:00:00Z",
}

describe("useUpdateExpertiseDomains", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("optimistically updates the profile cache before the request resolves", async () => {
    const { queryClient, wrapper } = createWrapperAndClient()
    queryClient.setQueryData(profileQueryKey("test-user-id"), baseProfile)

    let resolveMutation: (value: { expertise_domains: string[] }) => void =
      () => {}
    mockUpdateExpertiseDomains.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveMutation = resolve
        }),
    )

    const { result } = renderHook(() => useUpdateExpertiseDomains(), {
      wrapper,
    })

    act(() => {
      result.current.mutate(["development", "design_ui_ux"])
    })

    await waitFor(() => {
      const cached = queryClient.getQueryData<typeof baseProfile>(
        profileQueryKey("test-user-id"),
      )
      expect(cached?.expertise_domains).toEqual([
        "development",
        "design_ui_ux",
      ])
    })

    resolveMutation({ expertise_domains: ["development", "design_ui_ux"] })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
  })

  it("rolls back the cache to the previous value when the API fails", async () => {
    const { queryClient, wrapper } = createWrapperAndClient()
    queryClient.setQueryData(profileQueryKey("test-user-id"), baseProfile)

    mockUpdateExpertiseDomains.mockRejectedValue(new Error("boom"))

    const { result } = renderHook(() => useUpdateExpertiseDomains(), {
      wrapper,
    })

    await act(async () => {
      result.current.mutate(["development", "design_ui_ux"])
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    const cached = queryClient.getQueryData<typeof baseProfile>(
      profileQueryKey("test-user-id"),
    )
    expect(cached?.expertise_domains).toEqual(["development"])
  })

  it("writes the server-returned list into the cache on success", async () => {
    const { queryClient, wrapper } = createWrapperAndClient()
    queryClient.setQueryData(profileQueryKey("test-user-id"), baseProfile)

    mockUpdateExpertiseDomains.mockResolvedValue({
      expertise_domains: ["design_ui_ux", "development"],
    })

    const { result } = renderHook(() => useUpdateExpertiseDomains(), {
      wrapper,
    })

    await act(async () => {
      result.current.mutate(["development", "design_ui_ux"])
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const cached = queryClient.getQueryData<typeof baseProfile>(
      profileQueryKey("test-user-id"),
    )
    expect(cached?.expertise_domains).toEqual([
      "design_ui_ux",
      "development",
    ])
  })
})
