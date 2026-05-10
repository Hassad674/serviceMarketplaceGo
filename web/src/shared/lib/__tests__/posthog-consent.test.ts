import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

const sdkMock = vi.hoisted(() => ({
  init: vi.fn(),
  opt_in_capturing: vi.fn(),
  opt_out_capturing: vi.fn(),
  has_opted_out_capturing: vi.fn(() => false),
  capture: vi.fn(),
  identify: vi.fn(),
  group: vi.fn(),
  reset: vi.fn(),
}))

vi.mock("posthog-js", () => ({
  default: sdkMock,
  ...sdkMock,
}))

beforeEach(() => {
  Object.values(sdkMock).forEach((fn) => {
    if (typeof fn === "function" && "mockClear" in fn) fn.mockClear()
  })
  process.env.NEXT_PUBLIC_POSTHOG_KEY = "phc_test_key"
  vi.resetModules()
  if (typeof window !== "undefined") {
    window.localStorage.clear()
  }
})

afterEach(() => {
  if (typeof window !== "undefined") {
    window.localStorage.clear()
  }
})

describe("readConsent", () => {
  it("returns null when nothing has been persisted", async () => {
    const mod = await import("../posthog-consent")
    expect(mod.readConsent()).toBeNull()
  })

  it("returns 'accepted' or 'refused' when set", async () => {
    window.localStorage.setItem("marketplace.analytics.consent", "accepted")
    const mod = await import("../posthog-consent")
    expect(mod.readConsent()).toBe("accepted")
  })

  it("ignores garbage in storage", async () => {
    window.localStorage.setItem("marketplace.analytics.consent", "??")
    const mod = await import("../posthog-consent")
    expect(mod.readConsent()).toBeNull()
  })
})

describe("applyConsent", () => {
  it("opts the SDK in and persists 'accepted'", async () => {
    const mod = await import("../posthog-consent")
    mod.applyConsent("accepted")
    expect(sdkMock.opt_in_capturing).toHaveBeenCalledTimes(1)
    expect(sdkMock.opt_out_capturing).not.toHaveBeenCalled()
    expect(window.localStorage.getItem("marketplace.analytics.consent")).toBe("accepted")
  })

  it("opts the SDK out and persists 'refused'", async () => {
    const mod = await import("../posthog-consent")
    mod.applyConsent("refused")
    expect(sdkMock.opt_out_capturing).toHaveBeenCalledTimes(1)
    expect(sdkMock.opt_in_capturing).not.toHaveBeenCalled()
    expect(window.localStorage.getItem("marketplace.analytics.consent")).toBe("refused")
  })

  it("forces SDK init before toggling so the flag has somewhere to land", async () => {
    const mod = await import("../posthog-consent")
    mod.applyConsent("accepted")
    expect(sdkMock.init).toHaveBeenCalled()
  })
})

describe("clearConsent", () => {
  it("removes the stored choice", async () => {
    window.localStorage.setItem("marketplace.analytics.consent", "refused")
    const mod = await import("../posthog-consent")
    mod.clearConsent()
    expect(window.localStorage.getItem("marketplace.analytics.consent")).toBeNull()
  })
})
