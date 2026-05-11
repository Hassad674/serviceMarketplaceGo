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
  endIntroAttribution,
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
    await listMyReferrals({ statuses: ["pending_provider", "active"] })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals/me?status=pending_provider&status=active",
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
      statuses: ["pending_provider"],
      cursor: "tok",
    })
    const call = mockApiClient.mock.calls[0][0] as string
    expect(call).toContain("status=pending_provider")
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
      rate_pct: 10,
      duration_months: 6,
      intro_message_provider: "intro",
      intro_message_client: "intro",
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
      action: "accept",
      message: "lgtm",
    })
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals/r-1/respond",
      {
        method: "POST",
        body: expect.objectContaining({ action: "accept" }),
      },
    )
  })

  it("supports reject action", () => {
    respondToReferral("r-1", {
      action: "reject",
      message: "no thanks",
    })
    const body = (mockApiClient.mock.calls[0][1] as { body: { action: string } }).body
    expect(body.action).toBe("reject")
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

describe("referral-api / endIntroAttribution (Run C)", () => {
  it("POSTs to /attributions/{id}/end", async () => {
    mockApiClient.mockResolvedValueOnce({
      id: "att-1",
      referral_id: "ref-1",
      proposal_id: "prop-1",
      ended_at: "2026-05-11T10:00:00Z",
    })
    const result = await endIntroAttribution("att-1")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/referrals/attributions/att-1/end",
      { method: "POST" },
    )
    expect(result.ended_at).toBe("2026-05-11T10:00:00Z")
  })

  it("returns the bare body without envelope unwrapping (backend ships unwrapped)", async () => {
    mockApiClient.mockResolvedValueOnce({
      id: "att-2",
      referral_id: "ref-2",
      proposal_id: "prop-2",
      ended_at: "2026-05-11T12:00:00Z",
    })
    const result = await endIntroAttribution("att-2")
    expect(result.id).toBe("att-2")
    expect(result.referral_id).toBe("ref-2")
  })

  it("propagates 403 errors from the backend", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("forbidden"))
    await expect(endIntroAttribution("att-x")).rejects.toThrow("forbidden")
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
