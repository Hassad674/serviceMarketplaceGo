import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useFeePreview } from "../use-fee-preview"

const mockGetFeePreview = vi.fn()

vi.mock("../../api/billing-api", () => ({
  getFeePreview: (...args: unknown[]) => mockGetFeePreview(...args),
}))

// `getFeePreview` moved to shared (P9). The hook re-exported via
// `../use-fee-preview` calls the shared module, so we mock that path
// too.
vi.mock("@/shared/lib/billing/billing-api", () => ({
  getFeePreview: (...args: unknown[]) => mockGetFeePreview(...args),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  return { queryClient, wrapper }
}

const RESPONSE = {
  amount_cents: 100,
  fee_cents: 10,
  net_cents: 90,
  role: "freelance",
  active_tier_index: 0,
  tiers: [{ label: "0-200", max_cents: 20000, fee_cents: 1000 }],
  viewer_is_provider: true,
  viewer_is_subscribed: false,
}

beforeEach(() => {
  vi.clearAllMocks()
  mockGetFeePreview.mockResolvedValue(RESPONSE)
})

describe("useFeePreview", () => {
  it("eventually fires the API call (after the 300ms debounce)", async () => {
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useFeePreview(500), { wrapper })

    await waitFor(
      () => expect(result.current.isSuccess).toBe(true),
      { timeout: 2000 },
    )
    expect(mockGetFeePreview).toHaveBeenCalledWith(500, undefined)
  })

  it("clamps negative amounts to zero", async () => {
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useFeePreview(-100), { wrapper })
    await waitFor(
      () => expect(result.current.isSuccess).toBe(true),
      { timeout: 2000 },
    )
    expect(mockGetFeePreview).toHaveBeenCalledWith(0, undefined)
  })

  it("includes recipientId in the API call when provided", async () => {
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useFeePreview(1000, "rec-x"), {
      wrapper,
    })
    await waitFor(
      () => expect(result.current.isSuccess).toBe(true),
      { timeout: 2000 },
    )
    expect(mockGetFeePreview).toHaveBeenCalledWith(1000, "rec-x")
  })

  it("fires even at amount=0 so viewer_is_provider is known before typing", async () => {
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useFeePreview(0), { wrapper })
    await waitFor(
      () => expect(result.current.isSuccess).toBe(true),
      { timeout: 2000 },
    )
    expect(mockGetFeePreview).toHaveBeenCalledWith(0, undefined)
  })

  it("returns the data from the API once resolved", async () => {
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useFeePreview(100), { wrapper })
    await waitFor(
      () => expect(result.current.isSuccess).toBe(true),
      { timeout: 2000 },
    )
    expect(result.current.data).toEqual(RESPONSE)
  })

  it("scopes the query key by recipientId so two recipients don't share cache", async () => {
    mockGetFeePreview
      .mockResolvedValueOnce({ ...RESPONSE, viewer_is_provider: true })
      .mockResolvedValueOnce({ ...RESPONSE, viewer_is_provider: false })

    const { wrapper } = createWrapper()
    const { result: r1 } = renderHook(() => useFeePreview(100, "rec-A"), {
      wrapper,
    })
    const { result: r2 } = renderHook(() => useFeePreview(100, "rec-B"), {
      wrapper,
    })

    await waitFor(
      () => expect(r1.current.isSuccess).toBe(true),
      { timeout: 2000 },
    )
    await waitFor(
      () => expect(r2.current.isSuccess).toBe(true),
      { timeout: 2000 },
    )

    expect(mockGetFeePreview).toHaveBeenCalledWith(100, "rec-A")
    expect(mockGetFeePreview).toHaveBeenCalledWith(100, "rec-B")
  })

  it("propagates errors", async () => {
    mockGetFeePreview.mockRejectedValueOnce(new Error("boom"))
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useFeePreview(100), { wrapper })
    await waitFor(
      () => expect(result.current.isError).toBe(true),
      { timeout: 2000 },
    )
  })
})
