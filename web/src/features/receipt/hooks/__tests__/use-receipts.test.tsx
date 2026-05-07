import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement, type ReactNode } from "react"
import { useReceipt, useReceipts } from "../use-receipts"

const mockListReceipts = vi.fn()
const mockGetReceipt = vi.fn()

vi.mock("../../api/receipt-api", () => ({
  listReceipts: (...args: unknown[]) => mockListReceipts(...args),
  getReceipt: (...args: unknown[]) => mockGetReceipt(...args),
  // getReceiptPdfUrl is not used in hooks but kept for completeness
  getReceiptPdfUrl: vi.fn(),
}))

function wrapper({ children }: { children: ReactNode }) {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return createElement(QueryClientProvider, { client }, children)
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("useReceipts", () => {
  it("fetches the first page with no cursor on mount", async () => {
    mockListReceipts.mockResolvedValue({ data: [] })
    const { result } = renderHook(() => useReceipts(), { wrapper })
    await waitFor(() => expect(result.current.isLoading).toBe(false))
    expect(mockListReceipts).toHaveBeenCalledWith(undefined)
    expect(result.current.hasMore).toBe(false)
  })

  it("flags hasMore=true when next_cursor is returned", async () => {
    mockListReceipts.mockResolvedValue({ data: [], next_cursor: "next-1" })
    const { result } = renderHook(() => useReceipts(), { wrapper })
    await waitFor(() => expect(result.current.hasMore).toBe(true))
  })

  it("loadMore swaps the cursor and refetches the next page", async () => {
    mockListReceipts.mockResolvedValueOnce({ data: [], next_cursor: "next-1" })
    mockListReceipts.mockResolvedValueOnce({ data: [], next_cursor: undefined })
    const { result } = renderHook(() => useReceipts(), { wrapper })
    await waitFor(() => expect(result.current.hasMore).toBe(true))

    act(() => {
      result.current.loadMore()
    })

    await waitFor(() =>
      expect(mockListReceipts).toHaveBeenCalledWith("next-1"),
    )
    await waitFor(() => expect(result.current.hasMore).toBe(false))
  })

  it("reset clears the cursor", async () => {
    mockListReceipts.mockResolvedValue({ data: [], next_cursor: "next-1" })
    const { result } = renderHook(() => useReceipts(), { wrapper })
    await waitFor(() => expect(result.current.hasMore).toBe(true))

    act(() => result.current.loadMore())
    await waitFor(() =>
      expect(mockListReceipts).toHaveBeenCalledWith("next-1"),
    )

    act(() => result.current.reset())
    expect(result.current.cursor).toBeNull()
  })

  it("surfaces errors via isError", async () => {
    mockListReceipts.mockRejectedValue(new Error("boom"))
    const { result } = renderHook(() => useReceipts(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true), {
      timeout: 5_000,
    })
  })
})

describe("useReceipt", () => {
  it("does not fire when id is null", () => {
    renderHook(() => useReceipt(null), { wrapper })
    expect(mockGetReceipt).not.toHaveBeenCalled()
  })

  it("fetches the single receipt when id is provided", async () => {
    mockGetReceipt.mockResolvedValue({
      id: "rec-1",
      payment_record_id: "pay-1",
      amount_cents: 0,
      currency: "EUR",
      created_at: "2026-04-15T10:00:00Z",
      client: null,
      provider: null,
      referrer: null,
      referrer_commission_amount_cents: 0,
      snapshot_available: true,
    })
    const { result } = renderHook(() => useReceipt("rec-1"), { wrapper })
    await waitFor(() => expect(result.current.data?.id).toBe("rec-1"))
    expect(mockGetReceipt).toHaveBeenCalledWith("rec-1")
  })
})
