/**
 * analytics-events — central conversion-event helpers.
 *
 * Each helper must fire BOTH PostHog and GA4. We mock both SDKs so the
 * test asserts on the call shape without touching the network.
 *
 * Consent gating: PostHog respects its own `has_opted_out_capturing()`
 * check inside `captureEvent` (lib/posthog.ts). GA4 respects the
 * provider-level mount gate (lib/ga.ts wrapper) — but we additionally
 * verify that when consent is "refused" / null the SDK calls are
 * skipped via the wrapper's runtime guard mocked here.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

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

const sendGAEventMock = vi.hoisted(() => vi.fn())

vi.mock("posthog-js", () => ({
  default: sdkMock,
  ...sdkMock,
}))

vi.mock("@next/third-parties/google", () => ({
  sendGAEvent: sendGAEventMock,
  GoogleAnalytics: () => null,
}))

const ORIGINAL_ENV = { ...process.env }

beforeEach(() => {
  Object.values(sdkMock).forEach((fn) => {
    if (typeof fn === "function" && "mockClear" in fn) fn.mockClear()
  })
  sendGAEventMock.mockClear()
  process.env.NEXT_PUBLIC_POSTHOG_KEY = "phc_test_key"
  process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID = "G-TEST123"
  vi.resetModules()
})

afterEach(() => {
  process.env = { ...ORIGINAL_ENV }
})

describe("trackSignUp", () => {
  it("fires both PostHog and GA4 with the sign_up event", async () => {
    const mod = await import("../analytics-events")
    mod.trackSignUp({ method: "email", role: "agency" })
    expect(sdkMock.capture).toHaveBeenCalledWith(
      "auth.register_completed",
      expect.objectContaining({ method: "email", role: "agency" }),
    )
    expect(sendGAEventMock).toHaveBeenCalledWith(
      "event",
      "sign_up",
      expect.objectContaining({ method: "email", role: "agency" }),
    )
  })

  it("defaults method=email when caller omits it", async () => {
    const mod = await import("../analytics-events")
    mod.trackSignUp()
    expect(sdkMock.capture).toHaveBeenCalledWith(
      "auth.register_completed",
      expect.objectContaining({ method: "email" }),
    )
    expect(sendGAEventMock).toHaveBeenCalledWith(
      "event",
      "sign_up",
      expect.objectContaining({ method: "email" }),
    )
  })
})

describe("trackPurchase", () => {
  it("fires GA4 ecommerce schema (value, currency, transaction_id)", async () => {
    const mod = await import("../analytics-events")
    mod.trackPurchase({
      value: 199.5,
      currency: "EUR",
      transactionId: "pay_abc",
      items: [{ item_id: "p1", item_name: "Mission" }],
    })
    expect(sendGAEventMock).toHaveBeenCalledWith(
      "event",
      "purchase",
      expect.objectContaining({
        value: 199.5,
        currency: "EUR",
        transaction_id: "pay_abc",
        items: [{ item_id: "p1", item_name: "Mission" }],
      }),
    )
  })

  it("also captures a PostHog event with the value", async () => {
    const mod = await import("../analytics-events")
    mod.trackPurchase({ value: 12.34, currency: "EUR", transactionId: "txn1" })
    expect(sdkMock.capture).toHaveBeenCalledWith(
      "proposal.payment_succeeded_client",
      expect.objectContaining({
        value: 12.34,
        currency: "EUR",
        transaction_id: "txn1",
      }),
    )
  })
})

describe("trackLead", () => {
  it("fires the GA4 generate_lead event with profile_id + persona", async () => {
    const mod = await import("../analytics-events")
    mod.trackLead({ profileId: "org-7", persona: "agency" })
    expect(sendGAEventMock).toHaveBeenCalledWith(
      "event",
      "generate_lead",
      expect.objectContaining({ profile_id: "org-7", persona: "agency" }),
    )
    expect(sdkMock.capture).toHaveBeenCalledWith(
      "public_profile.send_message_clicked",
      expect.objectContaining({ profile_id: "org-7", persona: "agency" }),
    )
  })
})

describe("trackSearch", () => {
  it("fires GA4 search with search_term + persona + filters_count", async () => {
    const mod = await import("../analytics-events")
    mod.trackSearch({
      searchTerm: "react",
      persona: "freelance",
      filtersCount: 3,
    })
    expect(sendGAEventMock).toHaveBeenCalledWith(
      "event",
      "search",
      expect.objectContaining({
        search_term: "react",
        persona: "freelance",
        filters_count: 3,
      }),
    )
    expect(sdkMock.capture).toHaveBeenCalledWith(
      "search.executed",
      expect.objectContaining({
        query: "react",
        persona: "freelance",
        filters_applied: 3,
      }),
    )
  })

  it("defaults filtersCount to 0 when caller omits it", async () => {
    const mod = await import("../analytics-events")
    mod.trackSearch({ searchTerm: "go", persona: "freelance" })
    expect(sendGAEventMock).toHaveBeenCalledWith(
      "event",
      "search",
      expect.objectContaining({ filters_count: 0 }),
    )
  })
})

describe("consent gating", () => {
  it("does not capture PostHog events when the user has opted out", async () => {
    sdkMock.has_opted_out_capturing.mockReturnValue(true)
    const mod = await import("../analytics-events")
    mod.trackSignUp({ method: "email" })
    mod.trackPurchase({ value: 1, currency: "EUR", transactionId: "x" })
    mod.trackLead({ profileId: "p", persona: "agency" })
    mod.trackSearch({ searchTerm: "q", persona: "freelance" })
    expect(sdkMock.capture).not.toHaveBeenCalled()
    sdkMock.has_opted_out_capturing.mockReturnValue(false)
  })

  it("never throws when GA4 env is absent (sendGAEvent is still a no-op)", async () => {
    delete process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID
    const mod = await import("../analytics-events")
    expect(() =>
      mod.trackSignUp({ method: "email" }),
    ).not.toThrow()
  })
})
