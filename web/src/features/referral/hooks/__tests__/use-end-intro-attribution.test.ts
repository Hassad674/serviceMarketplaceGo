import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"

import { useEndIntroAttribution, referralKeys } from "../use-referrals"

const mockEndIntroAttribution = vi.fn()

// Mock JUST the endIntroAttribution export — the rest of the API
// module is wired through normally so the hook's surrounding
// re-exports stay intact.
vi.mock("../../api/referral-api", async () => {
  const actual = await vi.importActual<
    typeof import("../../api/referral-api")
  >("../../api/referral-api")
  return {
    ...actual,
    endIntroAttribution: (...args: unknown[]) =>
      mockEndIntroAttribution(...args),
  }
})

// `useRespondToReferral` from `@/shared/hooks/referral/use-referral`
// is re-exported by use-referrals.ts; the import side-effect resolves
// fine in vitest. Avoid mocking it here.

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
  return { Wrapper, queryClient }
}

describe("useEndIntroAttribution (Run C)", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls endIntroAttribution with the supplied attribution id", async () => {
    mockEndIntroAttribution.mockResolvedValue({
      id: "att-1",
      referral_id: "ref-1",
      proposal_id: "prop-1",
      ended_at: "2026-05-11T10:00:00Z",
    })

    const { Wrapper } = createWrapper()
    const { result } = renderHook(() => useEndIntroAttribution(), {
      wrapper: Wrapper,
    })

    await act(async () => {
      result.current.mutate("att-1")
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockEndIntroAttribution).toHaveBeenCalledWith("att-1")
    expect(result.current.data?.ended_at).toBe("2026-05-11T10:00:00Z")
  })

  it("invalidates the referrals tree and the wallet summary on success", async () => {
    mockEndIntroAttribution.mockResolvedValue({
      id: "att-2",
      referral_id: "ref-2",
      proposal_id: "prop-2",
      ended_at: "2026-05-11T11:00:00Z",
    })

    const { Wrapper, queryClient } = createWrapper()
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    const { result } = renderHook(() => useEndIntroAttribution(), {
      wrapper: Wrapper,
    })

    await act(async () => {
      result.current.mutate("att-2")
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    // Both invalidations fire on success — referrer tree refresh +
    // unified wallet summary refresh (commission projections drop the
    // ended attribution's milestones from "to be paid").
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: referralKeys.all,
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["wallet"],
    })
  })

  it("propagates a 403 error to the caller", async () => {
    mockEndIntroAttribution.mockRejectedValue(new Error("forbidden"))

    const { Wrapper } = createWrapper()
    const { result } = renderHook(() => useEndIntroAttribution(), {
      wrapper: Wrapper,
    })

    await act(async () => {
      result.current.mutate("att-x")
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("forbidden")
  })

  it("propagates a 404 error to the caller", async () => {
    mockEndIntroAttribution.mockRejectedValue(new Error("not_found"))

    const { Wrapper } = createWrapper()
    const { result } = renderHook(() => useEndIntroAttribution(), {
      wrapper: Wrapper,
    })

    await act(async () => {
      result.current.mutate("missing")
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("not_found")
  })
})
