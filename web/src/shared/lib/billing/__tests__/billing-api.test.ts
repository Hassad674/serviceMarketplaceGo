/**
 * billing-api.test.ts
 *
 * Unit tests for the shared billing fee preview wrapper. Verifies the
 * URL is built correctly under all the input variations the proposal
 * creation flow exercises.
 */
import { describe, it, expect, vi, beforeEach } from "vitest"
import { getFeePreview } from "../billing-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("getFeePreview / amount handling", () => {
  it("rounds float amounts down (truncate, not round)", async () => {
    await getFeePreview(100.9)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/billing/fee-preview?amount=100",
    )
  })

  it("clamps negative amounts to 0 (no leak of bad input to backend)", async () => {
    await getFeePreview(-50)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/billing/fee-preview?amount=0",
    )
  })

  it("passes integer amounts as-is", async () => {
    await getFeePreview(50000)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/billing/fee-preview?amount=50000",
    )
  })

  it("does NOT clamp NaN to 0 — current behaviour leaks 'NaN' to backend", async () => {
    // FLAGGED-FOR-FOLLOWUP: getFeePreview should clamp NaN to 0
    // (Math.max(0, NaN) returns NaN, so the URL ends up with
    // ?amount=NaN). Locking the current behaviour so the F.3.2 sweep
    // does not silently change it. Track in a separate ticket.
    await getFeePreview(Number.NaN)
    const path = mockApiClient.mock.calls[0][0]
    expect(path).toBe("/api/v1/billing/fee-preview?amount=NaN")
  })
})

describe("getFeePreview / recipient_id propagation", () => {
  it("appends recipient_id as a query param when provided", async () => {
    await getFeePreview(50000, "user-2")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/billing/fee-preview?amount=50000&recipient_id=user-2",
    )
  })

  it("URL-encodes the recipient_id", async () => {
    await getFeePreview(1000, "uid/special")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/billing/fee-preview?amount=1000&recipient_id=uid%2Fspecial",
    )
  })

  it("omits recipient_id when not provided", async () => {
    await getFeePreview(1000)
    const path = mockApiClient.mock.calls[0][0]
    expect(path).not.toContain("recipient_id")
  })
})

describe("getFeePreview / propagation", () => {
  it("returns the parsed FeePreview body", async () => {
    const fixture = {
      amount_cents: 100000,
      fee_cents: 1000,
      net_cents: 99000,
      role: "freelance",
      active_tier_index: 0,
      tiers: [],
      viewer_is_provider: true,
      viewer_is_subscribed: false,
    }
    mockApiClient.mockResolvedValueOnce(fixture)
    const res = await getFeePreview(100000)
    expect(res).toEqual(fixture)
  })

  it("propagates apiClient errors (e.g. 401 unauthenticated)", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("unauthorized"))
    await expect(getFeePreview(1000)).rejects.toThrow("unauthorized")
  })
})
