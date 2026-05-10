/**
 * GoogleAnalyticsProvider — env + consent dual-gate.
 *
 * Asserts that:
 *   1. With no measurement ID, the provider renders nothing.
 *   2. With an ID but no consent, the provider renders nothing.
 *   3. With an ID AND consent="accepted", `<GoogleAnalytics gaId={...} />`
 *      is rendered with the right id.
 *   4. The provider re-renders when the consent flips via the
 *      `analytics:consent-changed` event.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { act, render } from "@testing-library/react"

const gaProbe = vi.hoisted(() => vi.fn((_id: string) => undefined))

vi.mock("@next/third-parties/google", () => ({
  GoogleAnalytics: (props: { gaId: string }) => {
    gaProbe(props.gaId)
    return <div data-testid="ga-mounted" data-id={props.gaId} />
  },
  sendGAEvent: vi.fn(),
}))

const ORIGINAL_ENV = { ...process.env }

beforeEach(() => {
  gaProbe.mockClear()
  window.localStorage.clear()
  process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID = "G-TEST123"
  vi.resetModules()
})

afterEach(() => {
  window.localStorage.clear()
  process.env = { ...ORIGINAL_ENV }
})

async function renderProvider() {
  const { GoogleAnalyticsProvider } = await import("../google-analytics-provider")
  return render(<GoogleAnalyticsProvider />)
}

describe("GoogleAnalyticsProvider", () => {
  it("renders nothing when NEXT_PUBLIC_GA_MEASUREMENT_ID is empty", async () => {
    delete process.env.NEXT_PUBLIC_GA_MEASUREMENT_ID
    window.localStorage.setItem("marketplace.analytics.consent", "accepted")
    const { container } = await renderProvider()
    expect(container.querySelector('[data-testid="ga-mounted"]')).toBeNull()
    expect(gaProbe).not.toHaveBeenCalled()
  })

  it("renders nothing when consent is not granted (null)", async () => {
    const { container } = await renderProvider()
    expect(container.querySelector('[data-testid="ga-mounted"]')).toBeNull()
  })

  it("renders nothing when consent is explicitly refused", async () => {
    window.localStorage.setItem("marketplace.analytics.consent", "refused")
    const { container } = await renderProvider()
    expect(container.querySelector('[data-testid="ga-mounted"]')).toBeNull()
  })

  it("mounts <GoogleAnalytics> with the env id when both gates pass", async () => {
    window.localStorage.setItem("marketplace.analytics.consent", "accepted")
    const { container } = await renderProvider()
    const node = container.querySelector('[data-testid="ga-mounted"]')
    expect(node).not.toBeNull()
    expect(node?.getAttribute("data-id")).toBe("G-TEST123")
    expect(gaProbe).toHaveBeenCalledWith("G-TEST123")
  })

  it("re-renders when consent flips via analytics:consent-changed event", async () => {
    const { container } = await renderProvider()
    expect(container.querySelector('[data-testid="ga-mounted"]')).toBeNull()

    // User clicks "Accept" — applyConsent persists + dispatches the
    // custom event. Simulate the same flow here.
    window.localStorage.setItem("marketplace.analytics.consent", "accepted")
    await act(async () => {
      window.dispatchEvent(new CustomEvent("analytics:consent-changed"))
    })
    expect(container.querySelector('[data-testid="ga-mounted"]')).not.toBeNull()
  })
})
