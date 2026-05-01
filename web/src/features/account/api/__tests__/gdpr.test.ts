import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import {
  cancelDeletion,
  confirmDeletion,
  requestDeletion,
  exportMyData,
} from "../gdpr"
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
    blob: async () => new Blob([JSON.stringify(body)]),
    text: async () => JSON.stringify(body),
    headers: new Headers(),
  } as unknown as Response
}

describe("requestDeletion", () => {
  it("posts password + confirm and returns the parsed body", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(200, { email_sent_to: "u@x.com", expires_at: "2026-05-02T00:00:00Z" }),
    )
    const res = await requestDeletion("hunter2")
    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [, init] = fetchMock.mock.calls[0]
    expect(init.method).toBe("POST")
    expect(JSON.parse(init.body as string)).toEqual({ password: "hunter2", confirm: true })
    expect(res.email_sent_to).toBe("u@x.com")
  })

  it("throws ApiError on 401", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(401, { error: { code: "invalid_password", message: "bad" } }),
    )
    await expect(requestDeletion("wrong")).rejects.toThrowError(ApiError)
  })

  it("throws ApiError on 409 with the blocked_orgs body", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(409, {
        error: {
          code: "owner_must_transfer_or_dissolve",
          message: "blocked",
          details: { blocked_orgs: [{ org_id: "x", org_name: "Acme", member_count: 2 }] },
        },
      }),
    )
    try {
      await requestDeletion("x")
      throw new Error("should not reach")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const e = err as ApiError
      expect(e.status).toBe(409)
      expect(e.code).toBe("owner_must_transfer_or_dissolve")
    }
  })
})

describe("confirmDeletion", () => {
  it("encodes the token in the query string", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(200, {
        user_id: "u",
        deleted_at: "2026-05-01T12:00:00Z",
        hard_delete_at: "2026-05-31T12:00:00Z",
      }),
    )
    await confirmDeletion("a/b+c=d")
    const [url] = fetchMock.mock.calls[0]
    expect(url).toContain("token=a%2Fb%2Bc%3Dd")
  })
})

describe("cancelDeletion", () => {
  it("POSTs without a body", async () => {
    fetchMock.mockResolvedValue(jsonRes(200, { cancelled: true }))
    const res = await cancelDeletion()
    expect(res.cancelled).toBe(true)
    const [, init] = fetchMock.mock.calls[0]
    expect(init.method).toBe("POST")
  })
})

describe("exportMyData", () => {
  it("returns a Blob on success", async () => {
    fetchMock.mockResolvedValue({
      ok: true,
      status: 200,
      blob: async () => new Blob(["zip-bytes"]),
    } as unknown as Response)
    const blob = await exportMyData()
    expect(blob).toBeInstanceOf(Blob)
  })

  it("throws on 410 with the backend message", async () => {
    fetchMock.mockResolvedValue({
      ok: false,
      status: 410,
      json: async () => ({ message: "scheduled" }),
    } as unknown as Response)
    await expect(exportMyData()).rejects.toThrowError(/scheduled/)
  })
})
