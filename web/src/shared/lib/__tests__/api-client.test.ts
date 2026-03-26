import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { apiClient, ApiError } from "../api-client"

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch)
})

afterEach(() => {
  mockFetch.mockReset()
  vi.unstubAllGlobals()
})

describe("apiClient", () => {
  it("makes fetch with credentials: include", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ id: "1" }),
    })

    await apiClient("/api/v1/users/me")

    expect(mockFetch).toHaveBeenCalledOnce()
    const [url, options] = mockFetch.mock.calls[0]
    expect(url).toContain("/api/v1/users/me")
    expect(options.credentials).toBe("include")
  })

  it("sends Content-Type application/json header by default", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    })

    await apiClient("/api/v1/test")

    const [, options] = mockFetch.mock.calls[0]
    expect(options.headers["Content-Type"]).toBe("application/json")
  })

  it("uses GET method by default", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    })

    await apiClient("/api/v1/test")

    const [, options] = mockFetch.mock.calls[0]
    expect(options.method).toBe("GET")
  })

  it("sends JSON body for POST requests", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 201,
      json: () => Promise.resolve({ id: "new" }),
    })

    await apiClient("/api/v1/items", {
      method: "POST",
      body: { name: "test" },
    })

    const [, options] = mockFetch.mock.calls[0]
    expect(options.method).toBe("POST")
    expect(options.body).toBe(JSON.stringify({ name: "test" }))
  })

  it("does not include body key when no body provided", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    })

    await apiClient("/api/v1/test")

    const [, options] = mockFetch.mock.calls[0]
    expect(options).not.toHaveProperty("body")
  })

  it("merges custom headers with defaults", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({}),
    })

    await apiClient("/api/v1/test", {
      headers: { "X-Custom": "value" },
    })

    const [, options] = mockFetch.mock.calls[0]
    expect(options.headers["Content-Type"]).toBe("application/json")
    expect(options.headers["X-Custom"]).toBe("value")
  })

  it("throws ApiError on non-ok response", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 422,
      json: () =>
        Promise.resolve({
          error: "validation_error",
          message: "Email is required",
        }),
    })

    await expect(apiClient("/api/v1/test")).rejects.toThrow(ApiError)
  })

  it("ApiError has correct status, code, and message", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 403,
      json: () =>
        Promise.resolve({
          error: "forbidden",
          message: "Access denied",
        }),
    })

    try {
      await apiClient("/api/v1/test")
      expect.fail("Should have thrown")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const apiErr = err as ApiError
      expect(apiErr.status).toBe(403)
      expect(apiErr.code).toBe("forbidden")
      expect(apiErr.message).toBe("Access denied")
      expect(apiErr.name).toBe("ApiError")
    }
  })

  it("provides fallback when error response is not JSON", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: () => Promise.reject(new Error("not json")),
    })

    try {
      await apiClient("/api/v1/test")
      expect.fail("Should have thrown")
    } catch (err) {
      const apiErr = err as ApiError
      expect(apiErr.status).toBe(500)
      expect(apiErr.code).toBe("unknown_error")
      expect(apiErr.message).toBe("An error occurred")
    }
  })

  it("returns undefined on 204 No Content", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 204,
    })

    const result = await apiClient("/api/v1/test")

    expect(result).toBeUndefined()
  })

  it("parses JSON response on success", async () => {
    const payload = { id: "abc", name: "Test" }
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve(payload),
    })

    const result = await apiClient<{ id: string; name: string }>("/api/v1/test")

    expect(result).toEqual(payload)
  })
})

describe("ApiError", () => {
  it("extends Error", () => {
    const err = new ApiError(404, "not_found", "Resource not found")
    expect(err).toBeInstanceOf(Error)
  })

  it("has correct name property", () => {
    const err = new ApiError(400, "bad_request", "Bad request")
    expect(err.name).toBe("ApiError")
  })

  it("has correct status, code, and message", () => {
    const err = new ApiError(409, "conflict", "Already exists")
    expect(err.status).toBe(409)
    expect(err.code).toBe("conflict")
    expect(err.message).toBe("Already exists")
  })
})
