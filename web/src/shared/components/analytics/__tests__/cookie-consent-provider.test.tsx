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
      footer:
        "<privacy>Politique de confidentialité</privacy> · <cookies>Cookies</cookies> · <notices>Mentions légales</notices> · <subprocessors>Sous-processeurs</subprocessors>",
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

  // Regression: the cookieConsent.banner.footer message used to embed
  // raw `<a href="...">` markup, which next-intl 4.x rejects with
  // `INVALID_MESSAGE: INVALID_TAG` (the ICU parser does not allow tag
  // attributes). It now uses ICU rich-text tags inflated by `t.markup()`
  // so the i18n value is parser-safe AND the resulting HTML carries
  // locale-aware hrefs.
  describe("banner footer", () => {
    function footerOnLocale(locale: "fr" | "en"): string {
      const cfg = lastRunConfig()
      // both translations are built with the same buildFooterMarkup helper
      // and use the active `locale` for href resolution.
      const trans = cfg.language.translations[locale] as {
        consentModal: { footer: string }
      }
      return trans.consentModal.footer
    }

    it("renders four legal anchors (privacy + cookies + notices + subprocessors)", async () => {
      await mountProvider("fr")
      const html = footerOnLocale("fr")
      expect(html).toMatch(/Politique de confidentialité/)
      expect(html).toMatch(/Cookies/)
      expect(html).toMatch(/Mentions légales/)
      expect(html).toMatch(/Sous-processeurs/)
      // 4 anchors total
      const anchorMatches = html.match(/<a /g)
      expect(anchorMatches).toHaveLength(4)
    })

    it("emits FR-prefixed hrefs on the FR locale (as-needed + non-default)", async () => {
      await mountProvider("fr")
      const html = footerOnLocale("fr")
      expect(html).toContain(
        'href="/fr/legal/politique-confidentialite"',
      )
      expect(html).toContain('href="/fr/cookies"')
      expect(html).toContain('href="/fr/legal"')
      expect(html).toContain('href="/fr/sous-processeurs"')
    })

    it("emits bare EN-named hrefs on the EN (default) locale", async () => {
      await mountProvider("en")
      const html = footerOnLocale("en")
      // EN is defaultLocale + as-needed → no /en prefix.
      // Legal segments use their EN-named slugs.
      expect(html).toContain('href="/legal/privacy"')
      expect(html).toContain('href="/cookies"')
      expect(html).toContain('href="/legal"')
      expect(html).toContain('href="/subprocessors"')
    })

    it("never embeds raw <a href> attribute syntax in the i18n string itself", () => {
      // Belt-and-braces against a regression where someone re-introduces
      // the legacy footer pattern. The fixture string must use ICU rich
      // tags, never raw attribute-bearing anchors.
      const raw = messages.cookieConsent.banner.footer
      expect(raw).not.toMatch(/<a\s+href=/)
      expect(raw).toMatch(/<privacy>/)
      expect(raw).toMatch(/<cookies>/)
      expect(raw).toMatch(/<notices>/)
      expect(raw).toMatch(/<subprocessors>/)
    })
  })
})
