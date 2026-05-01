import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import {
  useWallet,
  useRequestPayout,
  useRetryTransfer,
} from "../use-wallet"

const mockGetWallet = vi.fn()
const mockRequestPayout = vi.fn()
const mockRetryFailedTransfer = vi.fn()

vi.mock("../../api/wallet-api", () => ({
  getWallet: (...args: unknown[]) => mockGetWallet(...args),
  requestPayout: (...args: unknown[]) => mockRequestPayout(...args),
  retryFailedTransfer: (...args: unknown[]) =>
    mockRetryFailedTransfer(...args),
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

describe("useWallet", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls getWallet API on mount", async () => {
    mockGetWallet.mockResolvedValue({
      stripe_account_id: "acct_123",
      charges_enabled: true,
      payouts_enabled: true,
      escrow_amount: 0,
      available_amount: 0,
      transferred_amount: 0,
      records: null,
    })

    const { result } = renderHook(() => useWallet(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockGetWallet).toHaveBeenCalledOnce()
  })

  it("returns wallet balance and records", async () => {
    mockGetWallet.mockResolvedValue({
      stripe_account_id: "acct_456",
      charges_enabled: true,
      payouts_enabled: true,
      escrow_amount: 5000,
      available_amount: 12000,
      transferred_amount: 8000,
      records: [
        {
          proposal_id: "prop-1",
          proposal_amount: 10000,
          platform_fee: 1000,
          provider_payout: 9000,
          payment_status: "paid",
          transfer_status: "completed",
          mission_status: "completed",
          created_at: "2026-03-15T10:00:00Z",
        },
        {
          proposal_id: "prop-2",
          proposal_amount: 5000,
          platform_fee: 500,
          provider_payout: 4500,
          payment_status: "paid",
          transfer_status: "pending",
          mission_status: "active",
          created_at: "2026-03-20T14:00:00Z",
        },
      ],
    })

    const { result } = renderHook(() => useWallet(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const wallet = result.current.data
    expect(wallet?.available_amount).toBe(12000)
    expect(wallet?.escrow_amount).toBe(5000)
    expect(wallet?.transferred_amount).toBe(8000)
    expect(wallet?.records).toHaveLength(2)
    expect(wallet?.records?.[0].provider_payout).toBe(9000)
  })

  it("handles wallet with no records", async () => {
    mockGetWallet.mockResolvedValue({
      stripe_account_id: "acct_new",
      charges_enabled: false,
      payouts_enabled: false,
      escrow_amount: 0,
      available_amount: 0,
      transferred_amount: 0,
      records: null,
    })

    const { result } = renderHook(() => useWallet(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data?.records).toBeNull()
    expect(result.current.data?.charges_enabled).toBe(false)
  })
})

describe("useRequestPayout", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls requestPayout API on mutate", async () => {
    mockRequestPayout.mockResolvedValue({
      status: "success",
      message: "Payout initiated",
    })

    const { result } = renderHook(() => useRequestPayout(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate()
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockRequestPayout).toHaveBeenCalledOnce()
    expect(result.current.data?.status).toBe("success")
  })

  it("handles payout failure", async () => {
    mockRequestPayout.mockRejectedValue(new Error("Insufficient balance"))

    const { result } = renderHook(() => useRequestPayout(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate()
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("Insufficient balance")
  })
})

describe("useRetryTransfer", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls retryFailedTransfer with the supplied record id", async () => {
    mockRetryFailedTransfer.mockResolvedValue({
      status: "transferred",
      message: "Transferred 21188 EUR to your account",
    })

    const { result } = renderHook(() => useRetryTransfer(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("rec-123")
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    // The hook MUST forward the record id verbatim — the backend uses
    // it to target the exact failed payment_record. Forwarding the
    // proposal id (legacy bug) over-released sibling milestones.
    expect(mockRetryFailedTransfer).toHaveBeenCalledWith("rec-123")
    expect(result.current.data?.status).toBe("transferred")
  })

  it("propagates the api error to the caller", async () => {
    mockRetryFailedTransfer.mockRejectedValue(
      new Error("provider_kyc_incomplete"),
    )

    const { result } = renderHook(() => useRetryTransfer(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("rec-123")
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("provider_kyc_incomplete")
  })
})
