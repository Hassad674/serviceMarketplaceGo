/**
 * invoicing-api.test.ts
 *
 * Unit tests for the invoicing feature's HTTP wrapper. Verifies the
 * cursor-pagination URL, encoding, and the PDF link helper.
 */
import { describe, it, expect, vi, beforeEach } from "vitest"
import { fetchInvoices, getInvoicePDFURL } from "../invoicing-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", async () => {
  const actual = await vi.importActual<typeof import("@/shared/lib/api-client")>(
    "@/shared/lib/api-client",
  )
  return {
    ...actual,
    apiClient: (...a: unknown[]) => mockApiClient(...a),
  }
})

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({ data: [] })
})

describe("invoicing-api / fetchInvoices", () => {
  it("calls /api/v1/me/invoices without a cursor", async () => {
    await fetchInvoices()
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/me/invoices")
  })

  it("URL-encodes the cursor", async () => {
    await fetchInvoices("abc/def")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/me/invoices?cursor=abc%2Fdef",
    )
  })

  it("handles + and = in the cursor", async () => {
    await fetchInvoices("a+b=c")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/me/invoices?cursor=a%2Bb%3Dc",
    )
  })

  it("returns the parsed invoices page", async () => {
    const page = { data: [{ id: "i1", number: "001" }] }
    mockApiClient.mockResolvedValueOnce(page)
    const result = await fetchInvoices()
    expect(result).toEqual(page)
  })

  it("propagates apiClient errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("boom"))
    await expect(fetchInvoices()).rejects.toThrow("boom")
  })
})

describe("invoicing-api / getInvoicePDFURL", () => {
  it("returns the relative path when API_BASE_URL is empty", () => {
    // In tests NEXT_PUBLIC_API_URL is not set, so API_BASE_URL is "".
    expect(getInvoicePDFURL("inv-1")).toBe("/api/v1/me/invoices/inv-1/pdf")
  })

  it("interpolates the id into the path", () => {
    expect(getInvoicePDFURL("abc-123")).toContain("/me/invoices/abc-123/pdf")
  })

  it("does not URL-encode the id (caller's responsibility)", () => {
    // The handler accepts UUIDs and short slugs only; we lock the
    // current behaviour so the F.3.2 sweep does not silently change it.
    expect(getInvoicePDFURL("abc/123")).toBe("/api/v1/me/invoices/abc/123/pdf")
  })
})
