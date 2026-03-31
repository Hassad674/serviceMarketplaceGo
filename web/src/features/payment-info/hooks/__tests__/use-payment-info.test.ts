import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import {
  usePaymentInfo,
  usePaymentInfoStatus,
  useCreateAccountSession,
} from "../use-payment-info"

const mockGetPaymentInfo = vi.fn()
const mockGetPaymentInfoStatus = vi.fn()
const mockCreateAccountSession = vi.fn()

vi.mock("../../api/payment-info-api", () => ({
  getPaymentInfo: (...args: unknown[]) => mockGetPaymentInfo(...args),
  getPaymentInfoStatus: (...args: unknown[]) =>
    mockGetPaymentInfoStatus(...args),
  createAccountSession: (...args: unknown[]) =>
    mockCreateAccountSession(...args),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "test-user-id",
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe("usePaymentInfo", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls getPaymentInfo API on mount", async () => {
    mockGetPaymentInfo.mockResolvedValue({
      id: "pi-1",
      user_id: "test-user-id",
      stripe_account_id: "acct_abc",
      stripe_verified: false,
      charges_enabled: false,
      payouts_enabled: false,
      created_at: "2026-03-20T10:00:00Z",
      updated_at: "2026-03-20T10:00:00Z",
    })

    const { result } = renderHook(() => usePaymentInfo(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockGetPaymentInfo).toHaveBeenCalledOnce()
  })

  it("returns payment info data from API", async () => {
    mockGetPaymentInfo.mockResolvedValue({
      id: "pi-1",
      user_id: "test-user-id",
      stripe_account_id: "acct_abc",
      stripe_verified: true,
      charges_enabled: true,
      payouts_enabled: true,
      stripe_business_type: "individual",
      stripe_country: "FR",
      stripe_display_name: "Alice Dupont",
      created_at: "2026-03-20T10:00:00Z",
      updated_at: "2026-03-20T10:00:00Z",
    })

    const { result } = renderHook(() => usePaymentInfo(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.stripe_verified).toBe(true)
    expect(result.current.data?.stripe_display_name).toBe("Alice Dupont")
  })

  it("handles null response for user with no payment info", async () => {
    mockGetPaymentInfo.mockResolvedValue(null)

    const { result } = renderHook(() => usePaymentInfo(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toBeNull()
  })
})

describe("usePaymentInfoStatus", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls getPaymentInfoStatus API on mount", async () => {
    mockGetPaymentInfoStatus.mockResolvedValue({ complete: true })

    const { result } = renderHook(() => usePaymentInfoStatus(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockGetPaymentInfoStatus).toHaveBeenCalledOnce()
  })

  it("returns complete status", async () => {
    mockGetPaymentInfoStatus.mockResolvedValue({ complete: true })

    const { result } = renderHook(() => usePaymentInfoStatus(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.complete).toBe(true)
  })
})

describe("useCreateAccountSession", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls createAccountSession API on mutate", async () => {
    mockCreateAccountSession.mockResolvedValue({
      client_secret: "cas_secret_123",
      stripe_account_id: "acct_abc",
    })

    const { result } = renderHook(() => useCreateAccountSession(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("alice@example.com")
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockCreateAccountSession).toHaveBeenCalledWith("alice@example.com")
    expect(result.current.data?.client_secret).toBe("cas_secret_123")
  })

  it("handles session creation failure", async () => {
    mockCreateAccountSession.mockRejectedValue(
      new Error("Stripe not configured"),
    )

    const { result } = renderHook(() => useCreateAccountSession(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("alice@example.com")
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("Stripe not configured")
  })
})
