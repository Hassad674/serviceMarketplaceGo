import { describe, it, expect } from "vitest"
import frMessages from "@/../messages/fr.json"
import enMessages from "@/../messages/en.json"

/**
 * landing-i18n-keys.test.ts is the trip-wire that guarantees every
 * key consumed by the landing components lives in BOTH `fr.json`
 * and `en.json`. If a future refactor renames a key in one locale
 * but forgets the other, the runtime would silently fall back to
 * the key name on screen — this test fails fast instead.
 *
 * The expected list is hand-maintained — it intentionally mirrors
 * the keys the landing components consume. Adding a new section
 * means adding its keys here too. That friction is desirable: it's
 * how we keep the i18n contract honest.
 */

const REQUIRED_KEYS = [
  "metaTitle",
  "metaDescription",
  "metaOgAlt",
  "skipToContent",
  "nav.howItWorks",
  "nav.providers",
  "nav.agencies",
  "nav.pricing",
  "hero.eyebrowDesktop",
  "hero.eyebrowMobile",
  "hero.titleLead",
  "hero.titleAccent",
  "hero.subtitleDesktop",
  "hero.subtitleMobile",
  "hero.ctaPrimary",
  "hero.ctaSecondary",
  "search.label",
  "search.tabFreelance",
  "search.tabFreelanceDescription",
  "search.tabReferrer",
  "search.tabReferrerDescription",
  "search.tabAgency",
  "search.tabAgencyDescription",
  "search.queryLabel",
  "search.queryPlaceholderFreelance",
  "search.queryPlaceholderReferrer",
  "search.queryPlaceholderAgency",
  "search.locationLabel",
  "search.locationPlaceholder",
  "search.submit",
  "search.submitAria",
  "search.popularLabel",
  "search.suggestion1",
  "search.suggestion5",
  "features.eyebrow",
  "features.titleLead",
  "features.titleAccent",
  "features.titleTrail",
  "features.card1Tag",
  "features.card1Title",
  "features.card1Body",
  "features.card2Tag",
  "features.card3Body",
  "pricing.eyebrow",
  "pricing.titleLead",
  "pricing.titleAccent",
  "pricing.intro",
  "pricing.step1Title",
  "pricing.step1Body",
  "pricing.step3Body",
  "pricing.tableEyebrow",
  "pricing.tableTitleLead",
  "pricing.tableTitleAccent",
  "pricing.tableTitleTrail",
  "pricing.rowClient",
  "pricing.rowReferrer",
  "pricing.rowFreelance",
  "pricing.rowAgency",
  "pricing.footnote",
  "credits.eyebrow",
  "credits.titleLead",
  "credits.titleAccent",
  "credits.p1",
  "credits.cardTitle",
  "credits.cardCount",
  "credits.cardFootnote",
  "referrers.eyebrow",
  "referrers.titleLead",
  "referrers.titleAccent",
  "referrers.titleTrail",
  "referrers.cardName",
  "referrers.cardItem4Score",
  "agencies.eyebrow",
  "agencies.titleLead",
  "agencies.titleAccent",
  "agencies.cta",
  "agencies.compareBeforeLabel",
  "agencies.compareAfterLabel",
  "ctaSection.titleLead",
  "ctaSection.titleAccent",
  "ctaSection.titleTrail",
  "ctaSection.subtitle",
  "ctaSection.ctaEnterprise",
  "ctaSection.ctaProvider",
  "ctaSection.ctaReferrer",
  "ctaSection.mobileTitleLead",
  "ctaSection.mobileTitleAccent",
  "ctaSection.mobileTitleTrail",
  "ctaSection.mobileSubtitle",
  "ctaSection.mobileCtaPrimary",
  "ctaSection.mobileCtaSecondary",
  "footer.tagline",
  "footer.mobileTagline",
  "footer.productLabel",
  "footer.productEnterprises",
  "footer.productPricing",
  "footer.understandLabel",
  "footer.legalLabel",
  "footer.madeBy",
  "footer.authorLinkedInAria",
] as const

function resolveDottedKey(
  obj: Record<string, unknown>,
  key: string,
): string | undefined {
  const segments = key.split(".")
  let current: unknown = obj
  for (const segment of segments) {
    if (current === null || typeof current !== "object") return undefined
    current = (current as Record<string, unknown>)[segment]
  }
  return typeof current === "string" ? current : undefined
}

describe("landing i18n keys", () => {
  const frLanding = (frMessages as Record<string, unknown>).landing as Record<
    string,
    unknown
  >
  const enLanding = (enMessages as Record<string, unknown>).landing as Record<
    string,
    unknown
  >

  it("every required key exists in fr.json under landing.*", () => {
    const missing = REQUIRED_KEYS.filter(
      (k) => resolveDottedKey(frLanding, k) === undefined,
    )
    expect(missing).toEqual([])
  })

  it("every required key exists in en.json under landing.*", () => {
    const missing = REQUIRED_KEYS.filter(
      (k) => resolveDottedKey(enLanding, k) === undefined,
    )
    expect(missing).toEqual([])
  })

  it("French copy uses tutoiement (regression guard for vouvoiement leaks)", () => {
    const credits = (
      (frLanding as Record<string, Record<string, string>>).credits ?? {}
    )
    // The credits section explicitly uses "Tu peux", "tu en reçois",
    // "Tes crédits". Lock the leading "Tu peux" — vouvoiement would
    // produce "Vous pouvez".
    expect(credits.p1).toMatch(/^Tu peux/)
    expect(credits.cardTitle).toMatch(/^Tes /)
  })
})
