import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { changeEmail, changePassword } from "../account-api"
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

describe("changeEmail", () => {
  it("posts current_password + new_email and returns the parsed body", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(200, {
        data: { email: "new@example.com" },
        meta: { request_id: "abc-123" },
      }),
    )

    const res = await changeEmail({
      current_password: "hunter2!ABC",
      new_email: "new@example.com",
    })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toContain("/api/v1/auth/change-email")
    expect(init.method).toBe("POST")
    expect(JSON.parse(init.body as string)).toEqual({
      current_password: "hunter2!ABC",
      new_email: "new@example.com",
    })
    expect(res.data.email).toBe("new@example.com")
    expect(res.meta.request_id).toBe("abc-123")
  })

  it("throws ApiError on 401 invalid_credentials", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(401, {
        error: { code: "invalid_credentials", message: "wrong password" },
      }),
    )

    try {
      await changeEmail({
        current_password: "wrong",
        new_email: "new@example.com",
      })
      throw new Error("should not reach")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const e = err as ApiError
      expect(e.status).toBe(401)
      expect(e.code).toBe("invalid_credentials")
    }
  })

  it("throws ApiError on 409 email_already_exists", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(409, {
        error: { code: "email_already_exists", message: "taken" },
      }),
    )

    try {
      await changeEmail({
        current_password: "ok",
        new_email: "taken@example.com",
      })
      throw new Error("should not reach")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const e = err as ApiError
      expect(e.code).toBe("email_already_exists")
    }
  })

  it("throws ApiError on 400 same_email", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(400, {
        error: { code: "same_email", message: "same" },
      }),
    )

    try {
      await changeEmail({
        current_password: "ok",
        new_email: "same@example.com",
      })
      throw new Error("should not reach")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      const e = err as ApiError
      expect(e.code).toBe("same_email")
    }
  })
})

describe("changePassword", () => {
  it("posts current_password + new_password and returns the parsed body", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(200, {
        data: { ok: true },
        meta: { request_id: "xyz-789" },
      }),
    )

    const res = await changePassword({
      current_password: "OldPass1!aaa",
      new_password: "NewPass1!aaa",
    })

    expect(fetchMock).toHaveBeenCalledTimes(1)
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toContain("/api/v1/auth/change-password")
    expect(init.method).toBe("POST")
    expect(JSON.parse(init.body as string)).toEqual({
      current_password: "OldPass1!aaa",
      new_password: "NewPass1!aaa",
    })
    expect(res.data.ok).toBe(true)
    expect(res.meta.request_id).toBe("xyz-789")
  })

  it("throws ApiError on 401 invalid_credentials", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(401, {
        error: { code: "invalid_credentials", message: "wrong" },
      }),
    )

    try {
      await changePassword({
        current_password: "wrong",
        new_password: "NewPass1!aaa",
      })
      throw new Error("should not reach")
    } catch (err) {
      expect(err).toBeInstanceOf(ApiError)
      expect((err as ApiError).code).toBe("invalid_credentials")
    }
  })

  it("throws ApiError on 400 weak_password", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(400, { error: { code: "weak_password", message: "weak" } }),
    )

    try {
      await changePassword({
        current_password: "old",
        new_password: "weak",
      })
      throw new Error("should not reach")
    } catch (err) {
      expect((err as ApiError).code).toBe("weak_password")
    }
  })

  it("throws ApiError on 400 same_password", async () => {
    fetchMock.mockResolvedValue(
      jsonRes(400, { error: { code: "same_password", message: "same" } }),
    )

    try {
      await changePassword({
        current_password: "OldPass1!aaa",
        new_password: "OldPass1!aaa",
      })
      throw new Error("should not reach")
    } catch (err) {
      expect((err as ApiError).code).toBe("same_password")
    }
  })
})
