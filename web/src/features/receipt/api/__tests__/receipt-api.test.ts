import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  getReceipt,
  getReceiptPdfUrl,
  listReceipts,
} from "../receipt-api"

// Pin the contract that the receipt-api wrappers issue exactly the
// expected URLs and forward the cursor query parameter. Mocks
// `global.fetch` directly (the api-client tests already cover error
// envelope mapping — this file only verifies the path/query shape).

const realFetch = global.fetch

beforeEach(() => {
  global.fetch = vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    json: async () => ({ data: [] }),
  }) as unknown as typeof fetch
})

afterEach(() => {
  global.fetch = realFetch
  vi.restoreAllMocks()
})

describe("listReceipts", () => {
  it("hits /api/v1/receipts without query when no cursor is given", async () => {
    await listReceipts()
    const calls = (global.fetch as unknown as { mock: { calls: unknown[][] } })
      .mock.calls
    expect(calls).toHaveLength(1)
    const [path] = calls[0] as [string]
    expect(path).toMatch(/\/api\/v1\/receipts$/)
  })

  it("appends an encoded cursor query parameter when provided", async () => {
    await listReceipts("abc def/=+&")
    const calls = (global.fetch as unknown as { mock: { calls: unknown[][] } })
      .mock.calls
    const [path] = calls[0] as [string]
    expect(path).toMatch(/\?cursor=abc%20def%2F%3D%2B%26$/)
  })

  it("returns the parsed response body as ReceiptsPage", async () => {
    const body = {
      data: [
        {
          id: "rec-1",
          payment_record_id: "pay-1",
          amount_cents: 12000,
          currency: "EUR",
          created_at: "2026-04-15T10:00:00Z",
          client: null,
          provider: null,
          referrer: null,
          referrer_commission_amount_cents: 0,
          snapshot_available: true,
        },
      ],
      next_cursor: "next-abc",
    }
    global.fetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => body,
    }) as unknown as typeof fetch
    const result = await listReceipts()
    expect(result).toEqual(body)
  })
})

describe("getReceipt", () => {
  it("hits /api/v1/receipts/{id} with the URL-encoded id", async () => {
    await getReceipt("a/b c")
    const calls = (global.fetch as unknown as { mock: { calls: unknown[][] } })
      .mock.calls
    const [path] = calls[0] as [string]
    expect(path).toMatch(/\/api\/v1\/receipts\/a%2Fb%20c$/)
  })
})

describe("getReceiptPdfUrl", () => {
  it("builds the PDF URL with the default fr language", () => {
    const url = getReceiptPdfUrl("rec-123")
    expect(url).toMatch(/\/api\/v1\/receipts\/rec-123\/pdf\?lang=fr$/)
  })

  it("builds the PDF URL with the explicit en language", () => {
    const url = getReceiptPdfUrl("rec-123", "en")
    expect(url).toMatch(/\/api\/v1\/receipts\/rec-123\/pdf\?lang=en$/)
  })

  it("URL-encodes the id segment", () => {
    const url = getReceiptPdfUrl("a/b")
    expect(url).toContain("/api/v1/receipts/a%2Fb/pdf")
  })
})
