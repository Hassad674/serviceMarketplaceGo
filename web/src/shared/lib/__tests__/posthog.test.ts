import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

// We mock posthog-js so we never touch a real network in tests but
// can assert on the SDK calls our wrapper made. Keeping the mock
// flexible (function fields per call) lets each test fence its own
// state without the previous one leaking through.
const sdkMock = vi.hoisted(() => ({
  init: vi.fn(),
  capture: vi.fn(),
  identify: vi.fn(),
  group: vi.fn(),
  reset: vi.fn(),
  has_opted_out_capturing: vi.fn(() => false),
  opt_in_capturing: vi.fn(),
  opt_out_capturing: vi.fn(),
  debug: vi.fn(),
}))

vi.mock("posthog-js", () => ({
  default: sdkMock,
  ...sdkMock,
}))

const ORIGINAL_ENV = { ...process.env }

beforeEach(() => {
  Object.values(sdkMock).forEach((fn) => {
    if (typeof fn === "function" && "mockClear" in fn) fn.mockClear()
  })
  process.env.NEXT_PUBLIC_POSTHOG_KEY = "phc_test_key"
  process.env.NEXT_PUBLIC_POSTHOG_HOST = "https://eu.posthog.com"
  vi.resetModules()
})

afterEach(() => {
  process.env = { ...ORIGINAL_ENV }
})

describe("getPostHogConfig", () => {
  it("returns enabled=true when key is set in browser", async () => {
    const mod = await import("../posthog")
    const cfg = mod.getPostHogConfig()
    expect(cfg.apiKey).toBe("phc_test_key")
    expect(cfg.apiHost).toBe("https://eu.posthog.com")
    expect(cfg.isEnabled).toBe(true)
  })

  it("returns enabled=false when the key is missing", async () => {
    delete process.env.NEXT_PUBLIC_POSTHOG_KEY
    const mod = await import("../posthog")
    expect(mod.getPostHogConfig().isEnabled).toBe(false)
  })

  it("defaults to the EU host when NEXT_PUBLIC_POSTHOG_HOST is unset", async () => {
    delete process.env.NEXT_PUBLIC_POSTHOG_HOST
    const mod = await import("../posthog")
    expect(mod.getPostHogConfig().apiHost).toBe("https://eu.posthog.com")
  })
})

describe("initPostHog", () => {
  it("calls posthog.init with opt-out-by-default and pageview capture", async () => {
    const mod = await import("../posthog")
    mod.initPostHog()
    expect(sdkMock.init).toHaveBeenCalledTimes(1)
    const [key, options] = sdkMock.init.mock.calls[0]
    expect(key).toBe("phc_test_key")
    expect(options).toMatchObject({
      api_host: "https://eu.posthog.com",
      opt_out_capturing_by_default: true,
      capture_pageview: true,
      autocapture: false,
    })
  })

  it("is idempotent — second call is a no-op", async () => {
    const mod = await import("../posthog")
    mod.initPostHog()
    mod.initPostHog()
    expect(sdkMock.init).toHaveBeenCalledTimes(1)
  })

  it("returns null when SDK is disabled", async () => {
    delete process.env.NEXT_PUBLIC_POSTHOG_KEY
    const mod = await import("../posthog")
    expect(mod.initPostHog()).toBeNull()
    expect(sdkMock.init).not.toHaveBeenCalled()
  })
})

describe("captureEvent", () => {
  it("captures with the right name and props", async () => {
    const mod = await import("../posthog")
    mod.captureEvent("landing.search_submitted", { query: "react" })
    expect(sdkMock.capture).toHaveBeenCalledWith("landing.search_submitted", { query: "react" })
  })

  it("respects opt-out — does not capture when user refused", async () => {
    sdkMock.has_opted_out_capturing.mockReturnValueOnce(true)
    const mod = await import("../posthog")
    mod.captureEvent("smoke.event", {})
    expect(sdkMock.capture).not.toHaveBeenCalled()
  })

  it("never throws when the SDK is disabled", async () => {
    delete process.env.NEXT_PUBLIC_POSTHOG_KEY
    const mod = await import("../posthog")
    expect(() => mod.captureEvent("evt", {})).not.toThrow()
    expect(sdkMock.capture).not.toHaveBeenCalled()
  })
})

describe("identifyUser + setOrganizationGroup", () => {
  it("calls posthog.identify with the distinct id", async () => {
    const mod = await import("../posthog")
    mod.identifyUser("user-123", { role: "agency" })
    expect(sdkMock.identify).toHaveBeenCalledWith("user-123", { role: "agency" })
  })

  it("attaches the organization group", async () => {
    const mod = await import("../posthog")
    mod.setOrganizationGroup("org-42", { type: "agency" })
    expect(sdkMock.group).toHaveBeenCalledWith("organization", "org-42", { type: "agency" })
  })
})

describe("resetPostHog", () => {
  it("calls posthog.reset only after init", async () => {
    const mod = await import("../posthog")
    mod.resetPostHog()
    // Not initialized yet.
    expect(sdkMock.reset).not.toHaveBeenCalled()
    mod.initPostHog()
    mod.resetPostHog()
    expect(sdkMock.reset).toHaveBeenCalledTimes(1)
  })
})

describe("isOnPublicAuthPath", () => {
  it("matches /login + /register + reset paths", async () => {
    const mod = await import("../posthog")
    expect(mod.isOnPublicAuthPath("/login")).toBe(true)
    expect(mod.isOnPublicAuthPath("/register")).toBe(true)
    expect(mod.isOnPublicAuthPath("/forgot-password")).toBe(true)
    expect(mod.isOnPublicAuthPath("/reset-password/abc")).toBe(true)
    expect(mod.isOnPublicAuthPath("/dashboard")).toBe(false)
  })
})
