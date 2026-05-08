import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { listSecurityActivity } from "../security-api"
import { ApiError } from "@/shared/lib/api-client"

const fetchMock = vi.fn()

beforeEach(() => {
  fetchMock.mockReset()
  globalThis.fetch = fetchMock as unknown as typeof fetch
})

afterEach(() => {
  vi.restoreAllMocks()
})

function jsonRes(status: number, body: unknown): Response {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: async () => body,
    text: async () => JSON.stringify(body),
    headers: new Headers(),
  } as unknown as Response
}

describe("listSecurityActivity", () => {
  it("calls the activity endpoint without query params when none are passed", async () => {
    fetchMock.mockResolvedValue(jsonRes(200, { data: [], next_cursor: "" }))
    const res = await listSecurityActivity()
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url] = fetchMock.mock.calls[0]
    expect(url).toContain("/api/v1/me/security/activity")
    expect(url).not.toContain("?")
    expect(res.data).toEqual([])
  })

  it("forwards cursor and limit as query parameters", async () => {
    fetchMock.mockResolvedValue(jsonRes(200, { data: [], next_cursor: "" }))
    await listSecurityActivity({ cursor: "abc", limit: 5 })
    const [url] = fetchMock.mock.calls[0]
    expect(url).toContain("cursor=abc")
    expect(url).toContain("limit=5")
  })

  it("returns the parsed event list", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(200, {
        data: [
          {
            id: "evt-1",
            action: "auth.login_success",
            ip_address: "203.0.113.4",
            user_agent_summary: "Ordinateur (Chrome 120)",
            access_kind: "desktop",
            created_at: "2026-05-08T12:00:00Z",
          },
        ],
        next_cursor: "next-page",
      }),
    )
    const res = await listSecurityActivity({ limit: 20 })
    expect(res.data).toHaveLength(1)
    expect(res.data[0].action).toBe("auth.login_success")
    expect(res.data[0].access_kind).toBe("desktop")
    expect(res.next_cursor).toBe("next-page")
  })

  it("throws ApiError on 401", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(401, {
        error: { code: "unauthorized", message: "no session" },
      }),
    )
    await expect(listSecurityActivity()).rejects.toBeInstanceOf(ApiError)
  })

  it("throws ApiError on 500", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(500, {
        error: { code: "security_activity_error", message: "boom" },
      }),
    )
    try {
      await listSecurityActivity()
      throw new Error("should not reach")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const e = err as ApiError
      expect(e.status).toBe(500)
      expect(e.code).toBe("security_activity_error")
    }
  })
})
