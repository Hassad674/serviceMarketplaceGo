import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { login, register, forgotPassword, resetPassword, AuthApiError } from "../auth-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...a: unknown[]) => mockApiClient(...a),
  API_BASE_URL: "",
}))

const mockUser = {
  id: "u-1",
  email: "x@x.com",
  first_name: "Joe",
  last_name: "Doe",
  display_name: "Joe",
  role: "agency",
  referrer_enabled: false,
  email_verified: true,
  created_at: "2026-04-30T00:00:00Z",
}

beforeEach(() => {
  vi.clearAllMocks()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe("auth-api / login", () => {
  it("POSTs credentials and returns the user on success", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: true,
      json: async () => mockUser,
    }))
    vi.stubGlobal("fetch", fetchMock)

    const user = await login("x@x.com", "pwd")
    expect(user).toEqual(mockUser)
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/auth/login",
      expect.objectContaining({
        method: "POST",
        credentials: "include",
        body: JSON.stringify({ email: "x@x.com", password: "pwd" }),
      }),
    )
  })

  it("throws AuthApiError with backend code on 4xx", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      json: async () => ({ error: "invalid_credentials", message: "no", reason: "bad" }),
    }))
    vi.stubGlobal("fetch", fetchMock)
    try {
      await login("x@x.com", "wrong")
      throw new Error("should have thrown")
    } catch (err) {
      expect(err).toBeInstanceOf(AuthApiError)
      expect((err as AuthApiError).code).toBe("invalid_credentials")
      expect((err as AuthApiError).reason).toBe("bad")
    }
  })

  it("falls back to generic error if backend body is invalid", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      json: async () => {
        throw new Error("not json")
      },
    }))
    vi.stubGlobal("fetch", fetchMock)
    try {
      await login("x@x.com", "wrong")
      throw new Error("should have thrown")
    } catch (err) {
      expect(err).toBeInstanceOf(AuthApiError)
      expect((err as AuthApiError).code).toBe("unknown")
    }
  })
})

describe("auth-api / register", () => {
  it("POSTs the data and returns the user on success", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: true,
      json: async () => mockUser,
    }))
    vi.stubGlobal("fetch", fetchMock)
    const user = await register({
      email: "x@x.com",
      password: "pwd",
      role: "agency",
    })
    expect(user).toEqual(mockUser)
  })

  it("throws Error with backend message on 4xx", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      json: async () => ({ message: "Email already in use" }),
    }))
    vi.stubGlobal("fetch", fetchMock)
    await expect(
      register({ email: "x@x.com", password: "pwd", role: "agency" }),
    ).rejects.toThrow("Email already in use")
  })

  it("falls back to 'An error occurred' if body unparseable (default fallback)", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      json: async () => {
        throw new Error("not json")
      },
    }))
    vi.stubGlobal("fetch", fetchMock)
    await expect(
      register({ email: "x@x.com", password: "pwd", role: "agency" }),
    ).rejects.toThrow("An error occurred")
  })

  it("falls back to 'Registration failed' when message is empty string", async () => {
    const fetchMock = vi.fn(async () => ({
      ok: false,
      json: async () => ({ message: "" }),
    }))
    vi.stubGlobal("fetch", fetchMock)
    await expect(
      register({ email: "x@x.com", password: "pwd", role: "agency" }),
    ).rejects.toThrow("Registration failed")
  })
})

describe("auth-api / forgotPassword", () => {
  it("POSTs the email", async () => {
    mockApiClient.mockResolvedValue({ message: "sent" })
    await forgotPassword("x@x.com")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/auth/forgot-password",
      { method: "POST", body: { email: "x@x.com" } },
    )
  })
})

describe("auth-api / resetPassword", () => {
  it("POSTs token + new_password", async () => {
    mockApiClient.mockResolvedValue({ message: "ok" })
    await resetPassword("tok", "newpwd")
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/auth/reset-password",
      { method: "POST", body: { token: "tok", new_password: "newpwd" } },
    )
  })

  it("propagates errors", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("expired"))
    await expect(resetPassword("t", "p")).rejects.toThrow("expired")
  })
})

describe("AuthApiError", () => {
  it("preserves the code and message", () => {
    const err = new AuthApiError("invalid", "Login failed")
    expect(err.code).toBe("invalid")
    expect(err.message).toBe("Login failed")
  })

  it("preserves an optional reason", () => {
    const err = new AuthApiError("locked", "Locked", "rate_limit")
    expect(err.reason).toBe("rate_limit")
  })

  it("is an instance of Error", () => {
    const err = new AuthApiError("x", "y")
    expect(err).toBeInstanceOf(Error)
  })
})
