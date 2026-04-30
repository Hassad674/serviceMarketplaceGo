import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  subscribe,
  getMySubscription,
  toggleAutoRenew,
  changeCycle,
  getStats,
  getCyclePreview,
  getPortalURL,
} from "../subscription-api"
import { ApiError } from "@/shared/lib/api-client"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", async () => {
  const actual = await vi.importActual<{
    ApiError: typeof ApiError
  }>("@/shared/lib/api-client")
  return {
    apiClient: (...args: unknown[]) => mockApiClient(...args),
    ApiError: actual.ApiError,
  }
})

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({})
})

describe("subscription-api / subscribe", () => {
  it("POSTs to /api/v1/subscriptions with the input", async () => {
    await subscribe({ billing_cycle: "monthly", return_url: "https://x" })
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/subscriptions", {
      method: "POST",
      body: { billing_cycle: "monthly", return_url: "https://x" },
    })
  })
})

describe("subscription-api / getMySubscription", () => {
  it("returns the subscription on 200", async () => {
    const sub = { id: "s-1", status: "active" }
    mockApiClient.mockResolvedValueOnce(sub)
    await expect(getMySubscription()).resolves.toEqual(sub)
  })

  it("returns null on 404", async () => {
    mockApiClient.mockRejectedValueOnce(
      new ApiError(404, "not_found", "no sub", null),
    )
    await expect(getMySubscription()).resolves.toBeNull()
  })

  it("propagates other errors", async () => {
    mockApiClient.mockRejectedValueOnce(
      new ApiError(500, "server_error", "boom", null),
    )
    await expect(getMySubscription()).rejects.toThrow("boom")
  })

  it("propagates non-ApiError exceptions", async () => {
    mockApiClient.mockRejectedValueOnce(new TypeError("network"))
    await expect(getMySubscription()).rejects.toThrow("network")
  })
})

describe("subscription-api / toggleAutoRenew", () => {
  it("PATCHes auto-renew with true", () => {
    toggleAutoRenew(true)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/subscriptions/me/auto-renew",
      { method: "PATCH", body: { auto_renew: true } },
    )
  })

  it("PATCHes auto-renew with false", () => {
    toggleAutoRenew(false)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/subscriptions/me/auto-renew",
      { method: "PATCH", body: { auto_renew: false } },
    )
  })
})

describe("subscription-api / changeCycle", () => {
  it("PATCHes the cycle endpoint with monthly", () => {
    changeCycle("monthly")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/subscriptions/me/billing-cycle",
      { method: "PATCH", body: { billing_cycle: "monthly" } },
    )
  })

  it("PATCHes the cycle endpoint with annual", () => {
    changeCycle("annual")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/subscriptions/me/billing-cycle",
      { method: "PATCH", body: { billing_cycle: "annual" } },
    )
  })
})

describe("subscription-api / getStats", () => {
  it("returns the stats on 200", async () => {
    const stats = { savings_cents: 5000, missions: 10 }
    mockApiClient.mockResolvedValueOnce(stats)
    await expect(getStats()).resolves.toEqual(stats)
  })

  it("returns null on 404", async () => {
    mockApiClient.mockRejectedValueOnce(
      new ApiError(404, "not_found", "", null),
    )
    await expect(getStats()).resolves.toBeNull()
  })

  it("propagates non-404 errors", async () => {
    mockApiClient.mockRejectedValueOnce(
      new ApiError(500, "x", "x", null),
    )
    await expect(getStats()).rejects.toThrow()
  })
})

describe("subscription-api / getCyclePreview", () => {
  it("GETs with the billing_cycle as a query param", () => {
    getCyclePreview("annual")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/subscriptions/me/cycle-preview?billing_cycle=annual",
    )
  })

  it("supports monthly", () => {
    getCyclePreview("monthly")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/subscriptions/me/cycle-preview?billing_cycle=monthly",
    )
  })
})

describe("subscription-api / getPortalURL", () => {
  it("returns the URL string from the JSON body", async () => {
    mockApiClient.mockResolvedValueOnce({ url: "https://stripe.com/portal/x" })
    await expect(getPortalURL()).resolves.toBe("https://stripe.com/portal/x")
  })

  it("propagates errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("nope"))
    await expect(getPortalURL()).rejects.toThrow("nope")
  })
})
