import { describe, it, expect, vi, beforeEach } from "vitest"
import { getFeePreview } from "../billing-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...args: unknown[]) => mockApiClient(...args),
}))

describe("billing-api / getFeePreview", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockApiClient.mockResolvedValue({
      amount_cents: 1000,
      fee_cents: 100,
      net_cents: 900,
      role: "freelance",
      active_tier_index: 0,
      tiers: [],
      viewer_is_provider: true,
      viewer_is_subscribed: false,
    })
  })

  it("calls apiClient with the amount as a query param", () => {
    getFeePreview(1000)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/billing/fee-preview?amount=1000",
    )
  })

  it("clamps negative amounts to zero", () => {
    getFeePreview(-50)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/billing/fee-preview?amount=0",
    )
  })

  it("truncates fractional amounts (centimes are integers)", () => {
    getFeePreview(150.6)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/billing/fee-preview?amount=150",
    )
  })

  it("appends recipient_id when provided", () => {
    getFeePreview(2000, "rec-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/billing/fee-preview?amount=2000&recipient_id=rec-1",
    )
  })

  it("does not include recipient_id when undefined", () => {
    getFeePreview(2000)
    const call = mockApiClient.mock.calls[0][0] as string
    expect(call.includes("recipient_id")).toBe(false)
  })

  it("returns the resolved FeePreview payload as-is", async () => {
    const promise = getFeePreview(500)
    await expect(promise).resolves.toMatchObject({
      role: "freelance",
      viewer_is_provider: true,
    })
  })

  it("propagates apiClient rejections", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("network down"))
    await expect(getFeePreview(100)).rejects.toThrow("network down")
  })

  it("handles zero amounts", () => {
    getFeePreview(0)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/billing/fee-preview?amount=0",
    )
  })
})
