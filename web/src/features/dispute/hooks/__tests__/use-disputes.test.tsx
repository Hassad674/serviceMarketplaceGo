import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import {
  useDispute,
  useOpenDispute,
  useCounterPropose,
  useRespondToCounter,
  useCancelDispute,
  useRespondToCancellation,
} from "../use-disputes"

const mockOpenDispute = vi.fn()
const mockGetDispute = vi.fn()
const mockCounterPropose = vi.fn()
const mockRespondToCounter = vi.fn()
const mockCancelDispute = vi.fn()
const mockRespondToCancellation = vi.fn()

vi.mock("../../api/dispute-api", () => ({
  openDispute: (...a: unknown[]) => mockOpenDispute(...a),
  getDispute: (...a: unknown[]) => mockGetDispute(...a),
  counterPropose: (...a: unknown[]) => mockCounterPropose(...a),
  respondToCounter: (...a: unknown[]) => mockRespondToCounter(...a),
  cancelDispute: (...a: unknown[]) => mockCancelDispute(...a),
  respondToCancellation: (...a: unknown[]) => mockRespondToCancellation(...a),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  return { queryClient, wrapper }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("useDispute", () => {
  it("queries when id is provided", async () => {
    mockGetDispute.mockResolvedValue({ id: "d-1" })
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useDispute("d-1"), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockGetDispute).toHaveBeenCalledWith("d-1")
  })

  it("does not query when id is undefined", () => {
    const { wrapper } = createWrapper()
    renderHook(() => useDispute(undefined), { wrapper })
    expect(mockGetDispute).not.toHaveBeenCalled()
  })
})

describe("useOpenDispute", () => {
  it("invalidates projects + proposals on success", async () => {
    mockOpenDispute.mockResolvedValue({ id: "d-1" })
    const { queryClient, wrapper } = createWrapper()
    const spy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHook(() => useOpenDispute(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({
        proposal_id: "p-1",
        reason: "harassment",
        description: "",
        message_to_party: "stop",
        requested_amount: 100,
      })
    })

    expect(spy).toHaveBeenCalledWith({ queryKey: ["projects"] })
    expect(spy).toHaveBeenCalledWith({ queryKey: ["proposals"] })
  })
})

describe("useCounterPropose", () => {
  it("invalidates dispute + projects on success", async () => {
    mockCounterPropose.mockResolvedValue({ id: "cp-1" })
    const { queryClient, wrapper } = createWrapper()
    const spy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHook(() => useCounterPropose("d-1"), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({ amount_client: 100, amount_provider: 200 })
    })

    expect(mockCounterPropose).toHaveBeenCalledWith("d-1", {
      amount_client: 100,
      amount_provider: 200,
    })
    expect(spy).toHaveBeenCalledWith({ queryKey: ["dispute", "d-1"] })
  })
})

describe("useRespondToCounter", () => {
  it("forwards both arguments and invalidates dispute", async () => {
    mockRespondToCounter.mockResolvedValue({ status: "accepted" })
    const { queryClient, wrapper } = createWrapper()
    const spy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHook(() => useRespondToCounter("d-1"), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({ cpId: "cp-1", accept: true })
    })

    expect(mockRespondToCounter).toHaveBeenCalledWith("d-1", "cp-1", true)
    expect(spy).toHaveBeenCalledWith({ queryKey: ["dispute", "d-1"] })
  })
})

describe("useCancelDispute", () => {
  it("invalidates dispute + projects + proposals", async () => {
    mockCancelDispute.mockResolvedValue({ status: "cancelled" })
    const { queryClient, wrapper } = createWrapper()
    const spy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHook(() => useCancelDispute(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync("d-1")
    })

    expect(mockCancelDispute).toHaveBeenCalledWith("d-1")
    expect(spy).toHaveBeenCalledWith({ queryKey: ["dispute"] })
    expect(spy).toHaveBeenCalledWith({ queryKey: ["projects"] })
  })
})

describe("useRespondToCancellation", () => {
  it("invalidates dispute by id and global keys", async () => {
    mockRespondToCancellation.mockResolvedValue({ status: "cancelled" })
    const { queryClient, wrapper } = createWrapper()
    const spy = vi.spyOn(queryClient, "invalidateQueries")
    const { result } = renderHook(() => useRespondToCancellation("d-9"), { wrapper })

    await act(async () => {
      await result.current.mutateAsync(true)
    })

    expect(mockRespondToCancellation).toHaveBeenCalledWith("d-9", true)
    expect(spy).toHaveBeenCalledWith({ queryKey: ["dispute", "d-9"] })
  })
})
