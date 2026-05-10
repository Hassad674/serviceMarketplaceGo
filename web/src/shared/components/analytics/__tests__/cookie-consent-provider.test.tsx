/**
 * CookieConsentProvider — vanilla-cookieconsent CMP integration.
 *
 * Covers:
 *   - mounts vanilla-cookieconsent.run with an opt-in CMP config and
 *     all NON-necessary categories starting OFF.
 *   - the `analytics` category is opt-in (not readOnly).
 *   - the `necessary` category is opt-in by default AND readOnly.
 *   - `onChange` syncs the analytics opt-in flag into the legacy
 *     localStorage key (so PostHog + GA4 providers light up).
 *   - the component returns null (no DOM of its own — the CMP injects
 *     its own modal into document.body).
 */
import { describe, expect, it, vi, beforeEach, afterEach } from "vitest"
import { render } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"

// Cast `vi.fn()` to a permissive shape so `.mock.calls[0][0]` reads as
// the CMP config object — the `vanilla-cookieconsent` types declare
// `run(cfg: CookieConsentConfig): Promise<void>`, but jest/vitest's
// hoisted mock infers `unknown[]` and we need `any` here just for the
// fixture access. No runtime behavior is altered.
type CmpRunArg = {
  mode: string
  autoShow: boolean
  categories: Record<string, { enabled: boolean; readOnly?: boolean }>
  language: { default: string; translations: Record<string, unknown> }
  onChange: (p: {
    cookie: { categories: string[] }
    changedCategories: string[]
    changedServices: Record<string, string[]>
  }) => void
  onFirstConsent: (p: { cookie: { categories: string[] } }) => void
}
const cookieConsentMock = vi.hoisted(() => ({
  run: vi.fn<(cfg: unknown) => Promise<void>>(() => Promise.resolve()),
  show: vi.fn(),
  hide: vi.fn(),
  showPreferences: vi.fn(),
  hidePreferences: vi.fn(),
  acceptedCategory: vi.fn(),
  acceptedService: vi.fn(),
  getCookie: vi.fn(),
  getConfig: vi.fn(),
  getUserPreferences: vi.fn(),
  setLanguage: vi.fn(),
  reset: vi.fn(),
}))
vi.mock("vanilla-cookieconsent", () => cookieConsentMock)

function lastRunConfig(): CmpRunArg {
  const calls = cookieConsentMock.run.mock.calls
  if (calls.length === 0) throw new Error("run() was not called")
  return calls[calls.length - 1][0] as CmpRunArg
}

const posthogSdkMock = vi.hoisted(() => ({
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
  default: posthogSdkMock,
  ...posthogSdkMock,
}))

// Block style imports — vitest doesn't handle CSS files in jsdom.
vi.mock("vanilla-cookieconsent/dist/cookieconsent.css", () => ({}))
vi.mock("@/styles/cookie-consent.css", () => ({}))

const messages = {
  cookieConsent: {
    banner: {
      title: "Tu gardes la main",
      description: "On utilise...",
      acceptAll: "Tout accepter",
      refuseAll: "Tout refuser",
      preferences: "Personnaliser",
      footer: "<a>Cookies</a>",
    },
    preferences: {
      title: "Préférences",
      save: "Enregistrer",
      close: "Fermer",
      intro: { title: "À quoi ça sert", description: "..." },
      necessary: { title: "Nécessaires", description: "..." },
      analytics: { title: "Mesure", description: "..." },
    },
  },
}

beforeEach(() => {
  cookieConsentMock.run.mockClear()
  Object.values(posthogSdkMock).forEach((fn) => {
    if (typeof fn === "function" && "mockClear" in fn) fn.mockClear()
  })
  process.env.NEXT_PUBLIC_POSTHOG_KEY = "phc_test_key"
  window.localStorage.clear()
  vi.resetModules()
})

afterEach(() => {
  window.localStorage.clear()
})

async function mountProvider(locale: "fr" | "en" = "fr") {
  const { CookieConsentProvider } = await import("../cookie-consent-provider")
  return render(
    <NextIntlClientProvider locale={locale} messages={messages}>
      <CookieConsentProvider />
    </NextIntlClientProvider>,
  )
}

describe("CookieConsentProvider", () => {
  it("calls vanilla-cookieconsent.run with an opt-in CMP config", async () => {
    await mountProvider()
    expect(cookieConsentMock.run).toHaveBeenCalledTimes(1)
    const cfg = lastRunConfig()
    expect(cfg.mode).toBe("opt-in")
    expect(cfg.autoShow).toBe(true)
  })

  it("declares the necessary category as readOnly + enabled", async () => {
    await mountProvider()
    const cfg = lastRunConfig()
    expect(cfg.categories.necessary).toEqual(
      expect.objectContaining({ enabled: true, readOnly: true }),
    )
  })

  it("declares the analytics category as opt-in (disabled by default, not readOnly)", async () => {
    await mountProvider()
    const cfg = lastRunConfig()
    expect(cfg.categories.analytics.enabled).toBe(false)
    expect(cfg.categories.analytics.readOnly).toBeFalsy()
  })

  it("does not register a 'functional' category (none in scope today)", async () => {
    await mountProvider()
    const cfg = lastRunConfig()
    expect(cfg.categories.functional).toBeUndefined()
  })

  it("renders no DOM of its own", async () => {
    const { container } = await mountProvider()
    // The CMP injects itself into document.body (mocked here, so nothing
    // renders). The provider must contribute zero markup of its own.
    expect(container.firstChild).toBeNull()
  })

  it("uses FR locale when next-intl is FR", async () => {
    await mountProvider("fr")
    const cfg = lastRunConfig()
    expect(cfg.language.default).toBe("fr")
    expect(cfg.language.translations.fr).toBeDefined()
    expect(cfg.language.translations.en).toBeDefined()
  })

  it("uses EN locale when next-intl is EN", async () => {
    await mountProvider("en")
    const cfg = lastRunConfig()
    expect(cfg.language.default).toBe("en")
  })

  it("opts PostHog in when onChange reports analytics consent", async () => {
    await mountProvider()
    const cfg = lastRunConfig()
    cfg.onChange({
      cookie: { categories: ["necessary", "analytics"] },
      changedCategories: ["analytics"],
      changedServices: {},
    })
    expect(posthogSdkMock.opt_in_capturing).toHaveBeenCalledTimes(1)
    expect(window.localStorage.getItem("marketplace.analytics.consent")).toBe("accepted")
  })

  it("opts PostHog out when onChange reports no analytics consent", async () => {
    await mountProvider()
    const cfg = lastRunConfig()
    cfg.onChange({
      cookie: { categories: ["necessary"] },
      changedCategories: ["analytics"],
      changedServices: {},
    })
    expect(posthogSdkMock.opt_out_capturing).toHaveBeenCalledTimes(1)
    expect(window.localStorage.getItem("marketplace.analytics.consent")).toBe("refused")
  })

  it("opts PostHog in on the very first consent", async () => {
    await mountProvider()
    const cfg = lastRunConfig()
    cfg.onFirstConsent({ cookie: { categories: ["necessary", "analytics"] } })
    expect(posthogSdkMock.opt_in_capturing).toHaveBeenCalledTimes(1)
  })
})
