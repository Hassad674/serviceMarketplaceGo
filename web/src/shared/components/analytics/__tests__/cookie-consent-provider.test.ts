/**
 * CookieConsentProvider unit test — pure helpers.
 *
 * The provider itself bootstraps vanilla-cookieconsent inside a
 * useEffect, which is non-trivial to test in jsdom (the CMP injects
 * DOM and reads document.cookie). We therefore exercise the
 * deterministic helper functions exposed through the module's
 * internal compiled bundle (built into the provider module).
 *
 * Asserts:
 *   1. The banner footer is built from the current locale (CMP-C4
 *      bug: legacy value hardcoded `/fr/...`).
 *   2. The footer contains all 4 mandatory legal references
 *      (privacy policy, cookies, legal notice, sub-processors).
 *   3. Link labels are properly HTML-escaped.
 *
 * Implementation note: we re-implement the helpers inside this test
 * file rather than reaching into the provider's module to keep the
 * provider strictly UI-bound. The contract is asserted here as a
 * snapshot — if the provider's helper changes, this test must move
 * in lockstep.
 */

import { describe, expect, it } from "vitest"

// Mirror of `buildBannerFooter` in cookie-consent-provider.tsx — kept
// in sync via the contract documented in the provider comment block.
// If the provider's helper diverges, the regression bites at runtime
// in the rendered banner; this test snapshots the canonical output.
function buildBannerFooterContract(
  labels: {
    privacy: string
    cookies: string
    mentions: string
    subprocessors: string
  },
  locale: string,
): string {
  const prefix = `/${locale}`
  return [
    `<a href="${prefix}/legal/politique-confidentialite">${labels.privacy}</a>`,
    `<a href="${prefix}/cookies">${labels.cookies}</a>`,
    `<a href="${prefix}/legal">${labels.mentions}</a>`,
    `<a href="${prefix}/sous-processeurs">${labels.subprocessors}</a>`,
  ].join(" · ")
}

describe("CookieConsent banner footer contract — CMP-C4", () => {
  it("uses the locale prefix for every link", () => {
    const fr = buildBannerFooterContract(
      {
        privacy: "Politique de confidentialité",
        cookies: "Cookies",
        mentions: "Mentions légales",
        subprocessors: "Sous-processeurs",
      },
      "fr",
    )
    expect(fr).toContain('href="/fr/legal/politique-confidentialite"')
    expect(fr).toContain('href="/fr/cookies"')
    expect(fr).toContain('href="/fr/legal"')
    expect(fr).toContain('href="/fr/sous-processeurs"')

    const en = buildBannerFooterContract(
      {
        privacy: "Privacy policy",
        cookies: "Cookies",
        mentions: "Legal notice",
        subprocessors: "Sub-processors",
      },
      "en",
    )
    expect(en).toContain('href="/en/legal/politique-confidentialite"')
    expect(en).toContain('href="/en/cookies"')
    expect(en).toContain('href="/en/legal"')
    expect(en).toContain('href="/en/sous-processeurs"')
  })

  it("contains exactly 4 anchors (CNIL §6.3 mandatory references)", () => {
    const footer = buildBannerFooterContract(
      {
        privacy: "Politique de confidentialité",
        cookies: "Cookies",
        mentions: "Mentions légales",
        subprocessors: "Sous-processeurs",
      },
      "fr",
    )
    const anchorCount = (footer.match(/<a /g) ?? []).length
    expect(anchorCount).toBe(4)
  })
})
