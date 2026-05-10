/**
 * CookieBanner — RGPD analytics consent banner.
 *
 * Covers:
 *   - Banner does not render when consent already exists.
 *   - "Refuser" persists the choice and calls posthog.opt_out_capturing.
 *   - "Accepter" persists the choice and calls posthog.opt_in_capturing.
 *   - Banner unmounts immediately after a click (no flash on next render).
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"

const sdkMock = vi.hoisted(() => ({
  init: vi.fn(),
  opt_in_capturing: vi.fn(),
  opt_out_capturing: vi.fn(),
  has_opted_out_capturing: vi.fn(() => false),
  capture: vi.fn(),
  identify: vi.fn(),
  group: vi.fn(),
  reset: vi.fn(),
  debug: vi.fn(),
}))
vi.mock("posthog-js", () => ({
  default: sdkMock,
  ...sdkMock,
}))

const messages = {
  analyticsConsent: {
    ariaLabel: "Préférences d'analyse",
    description: "On utilise un outil d'analyse pour améliorer le service. Tu peux refuser sans rien perdre.",
    accept: "Accepter",
    refuse: "Refuser",
  },
}

beforeEach(() => {
  Object.values(sdkMock).forEach((fn) => {
    if (typeof fn === "function" && "mockClear" in fn) fn.mockClear()
  })
  process.env.NEXT_PUBLIC_POSTHOG_KEY = "phc_test_key"
  window.localStorage.clear()
  vi.resetModules()
})

afterEach(() => {
  window.localStorage.clear()
})

async function renderBanner() {
  const { CookieBanner } = await import("../cookie-banner")
  return render(
    <NextIntlClientProvider locale="fr" messages={messages}>
      <CookieBanner />
    </NextIntlClientProvider>,
  )
}

describe("CookieBanner", () => {
  it("renders when no consent has been recorded", async () => {
    await renderBanner()
    expect(await screen.findByText(messages.analyticsConsent.description)).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Refuser" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Accepter" })).toBeInTheDocument()
  })

  it("does not render when consent is already recorded", async () => {
    window.localStorage.setItem("marketplace.analytics.consent", "accepted")
    await renderBanner()
    expect(screen.queryByText(messages.analyticsConsent.description)).not.toBeInTheDocument()
  })

  it("opts out and persists the choice when Refuser is clicked", async () => {
    const user = userEvent.setup()
    await renderBanner()
    await user.click(await screen.findByRole("button", { name: "Refuser" }))
    expect(sdkMock.opt_out_capturing).toHaveBeenCalledTimes(1)
    expect(window.localStorage.getItem("marketplace.analytics.consent")).toBe("refused")
    expect(screen.queryByText(messages.analyticsConsent.description)).not.toBeInTheDocument()
  })

  it("opts in and persists the choice when Accepter is clicked", async () => {
    const user = userEvent.setup()
    await renderBanner()
    await user.click(await screen.findByRole("button", { name: "Accepter" }))
    expect(sdkMock.opt_in_capturing).toHaveBeenCalledTimes(1)
    expect(window.localStorage.getItem("marketplace.analytics.consent")).toBe("accepted")
  })
})
