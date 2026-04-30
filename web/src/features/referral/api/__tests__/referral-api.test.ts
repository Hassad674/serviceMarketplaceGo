import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  listMyReferrals,
  listIncomingReferrals,
  getReferral,
  createReferral,
  respondToReferral,
  listNegotiations,
  listAttributions,
  listCommissions,
} from "../referral-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
}))

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue([])
})

describe("referral-api / listMyReferrals", () => {
  it("calls /me with no query string when filter is empty", async () => {
    await listMyReferrals()
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/referrals/me")
  })

  it("appends status filter as repeated params", async () => {
    await listMyReferrals({ statuses: ["pending", "accepted"] })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals/me?status=pending&status=accepted",
    )
  })

  it("appends cursor", async () => {
    await listMyReferrals({ cursor: "tok1" })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals/me?cursor=tok1",
    )
  })

  it("URL-encodes cursor", async () => {
    await listMyReferrals({ cursor: "a/b" })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals/me?cursor=a%2Fb",
    )
  })
})

describe("referral-api / listIncomingReferrals", () => {
  it("calls /incoming with no query string when filter is empty", async () => {
    await listIncomingReferrals()
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/referrals/incoming")
  })

  it("appends both statuses and cursor", async () => {
    await listIncomingReferrals({
      statuses: ["pending"],
      cursor: "tok",
    })
    const call = mockApiClient.mock.calls[0][0] as string
    expect(call).toContain("status=pending")
    expect(call).toContain("cursor=tok")
  })
})

describe("referral-api / getReferral", () => {
  it("GETs by id", () => {
    getReferral("r-1")
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/referrals/r-1")
  })
})

describe("referral-api / createReferral", () => {
  it("POSTs to /api/v1/referrals", () => {
    createReferral({
      provider_id: "u-2",
      client_id: "u-3",
      target_amount: 100000,
      commission_pct: 10,
      message: "intro",
    })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals",
      {
        method: "POST",
        body: expect.objectContaining({ provider_id: "u-2" }),
      },
    )
  })
})

describe("referral-api / respondToReferral", () => {
  it("POSTs to /respond with the body", () => {
    respondToReferral("r-1", {
      accepted: true,
      counter_amount: undefined,
      counter_commission_pct: undefined,
      message: "lgtm",
    })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals/r-1/respond",
      {
        method: "POST",
        body: expect.objectContaining({ accepted: true }),
      },
    )
  })

  it("supports decline", () => {
    respondToReferral("r-1", {
      accepted: false,
      counter_amount: undefined,
      counter_commission_pct: undefined,
      message: "no thanks",
    })
    const body = (mockApiClient.mock.calls[0][1] as { body: { accepted: boolean } }).body
    expect(body.accepted).toBe(false)
  })
})

describe("referral-api / nested resources", () => {
  it("listNegotiations GETs /:id/negotiations", () => {
    listNegotiations("r-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals/r-1/negotiations",
    )
  })

  it("listAttributions GETs /:id/attributions", () => {
    listAttributions("r-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals/r-1/attributions",
    )
  })

  it("listCommissions GETs /:id/commissions", () => {
    listCommissions("r-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals/r-1/commissions",
    )
  })
})

describe("referral-api / error propagation", () => {
  it("propagates errors from listMyReferrals", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("502"))
    await expect(listMyReferrals()).rejects.toThrow("502")
  })

  it("propagates errors from getReferral", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("404"))
    await expect(getReferral("missing")).rejects.toThrow("404")
  })
})
