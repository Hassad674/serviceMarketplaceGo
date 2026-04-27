import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { fetchAdminInvoices, openInvoicePDF } from "../api/invoicing-api"
import { EMPTY_ADMIN_INVOICE_FILTERS } from "../types"

// makeResponseWithURL builds a Response object and overrides its
// `url` getter so we can simulate fetch following a redirect to the
// presigned R2 URL. Plain Response objects expose `url` only via a
// getter, so direct assignment fails — we use Object.defineProperty.
function makeResponseWithURL(url: string, status = 200): Response {
  const r = new Response("", { status })
  Object.defineProperty(r, "url", { value: url, configurable: true })
  return r
}

describe("fetchAdminInvoices", () => {
  beforeEach(() => {
    localStorage.setItem("admin_token", "test-token")
    vi.stubGlobal("fetch", vi.fn())
  })
  afterEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
  })

  it("hits the listing endpoint with no filters", async () => {
    const mockFetch = globalThis.fetch as unknown as ReturnType<typeof vi.fn>
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ data: [], has_more: false }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    await fetchAdminInvoices(EMPTY_ADMIN_INVOICE_FILTERS)
    const url = mockFetch.mock.calls[0][0] as string
    expect(url).toContain("/api/v1/admin/invoices")
    expect(url).toContain("limit=20")
    // No filter params should be present
    expect(url).not.toContain("status=")
    expect(url).not.toContain("recipient_org_id=")
  })

  it("serializes every populated filter into the query string", async () => {
    const mockFetch = globalThis.fetch as unknown as ReturnType<typeof vi.fn>
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ data: [], has_more: false }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    await fetchAdminInvoices({
      recipient_org_id: "00000000-0000-0000-0000-000000000001",
      status: "subscription",
      date_from: "2026-04-01T00:00:00Z",
      date_to: "2026-04-30T23:59:59Z",
      min_amount_cents: "1000",
      max_amount_cents: "50000",
      search: "Acme",
      cursor: "cur1",
    })
    const url = mockFetch.mock.calls[0][0] as string
    expect(url).toContain("recipient_org_id=00000000-0000-0000-0000-000000000001")
    expect(url).toContain("status=subscription")
    expect(url).toContain("date_from=2026-04-01T00%3A00%3A00Z")
    expect(url).toContain("date_to=2026-04-30T23%3A59%3A59Z")
    expect(url).toContain("min_amount_cents=1000")
    expect(url).toContain("max_amount_cents=50000")
    expect(url).toContain("search=Acme")
    expect(url).toContain("cursor=cur1")
  })

  it("sends the bearer token from localStorage", async () => {
    const mockFetch = globalThis.fetch as unknown as ReturnType<typeof vi.fn>
    mockFetch.mockResolvedValueOnce(
      new Response(JSON.stringify({ data: [], has_more: false }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    )
    await fetchAdminInvoices(EMPTY_ADMIN_INVOICE_FILTERS)
    const init = mockFetch.mock.calls[0][1] as RequestInit
    const headers = new Headers(init.headers)
    expect(headers.get("Authorization")).toBe("Bearer test-token")
  })
})

describe("openInvoicePDF", () => {
  beforeEach(() => {
    localStorage.setItem("admin_token", "test-token")
    vi.stubGlobal("fetch", vi.fn())
  })
  afterEach(() => {
    localStorage.clear()
    vi.unstubAllGlobals()
  })

  it("uses ?type=invoice for non-credit-note rows", async () => {
    const mockFetch = globalThis.fetch as unknown as ReturnType<typeof vi.fn>
    mockFetch.mockResolvedValueOnce(
      makeResponseWithURL("https://r2.test/abc.pdf"),
    )
    await openInvoicePDF("11111111-1111-1111-1111-111111111111", false)
    const url = mockFetch.mock.calls[0][0] as string
    expect(url).toContain("/api/v1/admin/invoices/11111111-1111-1111-1111-111111111111/pdf")
    expect(url).toContain("type=invoice")
  })

  it("uses ?type=credit_note for credit-note rows", async () => {
    const mockFetch = globalThis.fetch as unknown as ReturnType<typeof vi.fn>
    mockFetch.mockResolvedValueOnce(
      makeResponseWithURL("https://r2.test/cn.pdf"),
    )
    await openInvoicePDF("22222222-2222-2222-2222-222222222222", true)
    const url = mockFetch.mock.calls[0][0] as string
    expect(url).toContain("type=credit_note")
  })

  it("throws when the redirect endpoint replies non-OK", async () => {
    const mockFetch = globalThis.fetch as unknown as ReturnType<typeof vi.fn>
    mockFetch.mockResolvedValueOnce(new Response("", { status: 404 }))
    await expect(
      openInvoicePDF("33333333-3333-3333-3333-333333333333", false),
    ).rejects.toThrow(/404/)
  })
})
