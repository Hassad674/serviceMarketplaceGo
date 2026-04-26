import { describe, it, expect, vi, afterEach } from "vitest"
import { apiClient, ApiError } from "@/shared/lib/api-client"

// These tests pin the contract that the api-client preserves the
// raw JSON envelope on `ApiError.body`. The invoicing completion
// modal depends on this — without it the wallet/subscribe gates
// could not surface `missing_fields` from a 403 response.

const realFetch = global.fetch

afterEach(() => {
  global.fetch = realFetch
  vi.restoreAllMocks()
})

function mockFetchResponse(status: number, body: unknown) {
  global.fetch = vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
  }) as unknown as typeof fetch
}

describe("api-client error mapping", () => {
  it("preserves the parsed body on the ApiError", async () => {
    mockFetchResponse(403, {
      error: { code: "billing_profile_incomplete", message: "incomplete" },
      missing_fields: [{ field: "legal_name", reason: "required" }],
    })
    let caught: unknown
    try {
      await apiClient<unknown>("/test")
    } catch (err) {
      caught = err
    }
    expect(caught).toBeInstanceOf(ApiError)
    const apiErr = caught as ApiError
    expect(apiErr.status).toBe(403)
    expect(apiErr.code).toBe("billing_profile_incomplete")
    expect(apiErr.body).toEqual({
      error: { code: "billing_profile_incomplete", message: "incomplete" },
      missing_fields: [{ field: "legal_name", reason: "required" }],
    })
  })

  it("falls back to defaults for empty error responses", async () => {
    mockFetchResponse(500, null)
    let caught: unknown
    try {
      await apiClient<unknown>("/test")
    } catch (err) {
      caught = err
    }
    expect(caught).toBeInstanceOf(ApiError)
    const apiErr = caught as ApiError
    expect(apiErr.status).toBe(500)
    expect(apiErr.code).toBe("unknown_error")
    expect(apiErr.body).toBeNull()
  })

  it("keeps backwards compatibility with the legacy flat error shape", async () => {
    mockFetchResponse(404, { error: "not_found", message: "missing" })
    let caught: unknown
    try {
      await apiClient<unknown>("/test")
    } catch (err) {
      caught = err
    }
    const apiErr = caught as ApiError
    expect(apiErr.code).toBe("not_found")
    expect(apiErr.message).toBe("missing")
  })
})
