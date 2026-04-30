import { describe, it, expect, vi, beforeEach } from "vitest"
import { getWallet, requestPayout, retryFailedTransfer } from "../wallet-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("wallet-api", () => {
  it("getWallet GETs /api/v1/wallet", () => {
    getWallet()
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/wallet")
  })

  it("requestPayout POSTs to /api/v1/wallet/payout", () => {
    requestPayout()
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/wallet/payout",
      { method: "POST" },
    )
  })

  it("retryFailedTransfer POSTs to the record-scoped endpoint", () => {
    retryFailedTransfer("rec-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/wallet/transfers/rec-1/retry",
      { method: "POST" },
    )
  })

  it("propagates getWallet errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("403"))
    await expect(getWallet()).rejects.toThrow("403")
  })

  it("propagates payout errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("blocked"))
    await expect(requestPayout()).rejects.toThrow("blocked")
  })

  it("propagates retry errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("conflict"))
    await expect(retryFailedTransfer("r")).rejects.toThrow("conflict")
  })
})
