/**
 * Phase A.4 → legal-max-blindage — placeholder legal route metadata
 * tests.
 *
 * The remaining placeholder pages (/cookies, /legal, /cgu, /cgv,
 * /sous-processeurs) all expose generateMetadata that:
 *   1. interpolates a localized title with " | Marketplace Service" suffix
 *   2. sets robots noindex/nofollow (internal placeholder shells —
 *      indexable canonical surfaces live under /legal/*)
 *   3. surfaces the localized intro string as the description
 *
 * The legacy short /privacy page was removed in the legal-max-blindage
 * round (CNIL requires a single privacy policy — /legal/politique-confidentialite
 * is now the only canonical version).
 *
 * Mocks next-intl/server, next-intl, and the @i18n/navigation Link so
 * the page modules can be imported in a node environment without a
 * Next.js runtime.
 */

import { describe, it, expect, vi } from "vitest"

vi.mock("next-intl/server", () => ({
  getTranslations: async ({ namespace }: { namespace: string }) => {
    return (key: string) => `[${namespace}.${key}]`
  },
}))

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

const redirectMock = vi.fn()
vi.mock("@i18n/navigation", () => ({
  Link: ({ children }: { children: React.ReactNode }) => children,
  redirect: (...args: unknown[]) => redirectMock(...args),
}))

import * as Cookies from "@/app/[locale]/(public)/cookies/page"
import * as LegalMentions from "@/app/[locale]/(public)/legal/page"
import * as Cgu from "@/app/[locale]/(public)/cgu/page"
import * as Cgv from "@/app/[locale]/(public)/cgv/page"
import * as Sub from "@/app/[locale]/(public)/sous-processeurs/page"

// The /legal index moved to the legal.docs namespace in D4: it now
// serves as the sommaire of the 6 D4 documents while still hosting the
// mentions légales block at the top. Title + description come from
// `legal.docs.indexTitle` / `legal.docs.indexIntro`.
//
// Stripe + DSA art. 14 require the /legal index to be indexable (the
// "Conditions générales claires" obligation). It now intentionally
// omits the `robots` metadata so the default policy (indexable) applies.
const CASES = [
  { mod: Cookies, namespace: "legal.cookies", label: "cookies", indexable: false },
  {
    mod: LegalMentions,
    namespace: "legal",
    titleKey: "docs.indexTitle",
    introKey: "docs.indexIntro",
    label: "legal",
    indexable: true,
  },
  // /cgu and /cgv are no longer placeholder shells: they now
  // permanently redirect to the canonical full documents under
  // /legal/cgu and /legal/cgv (locale-aware). They are asserted
  // separately below — they intentionally export no generateMetadata.
  // /sous-processeurs MUST be crawlable — RGPD art. 28 transparency
  // + DSA art. 14 require visitors and auditors to access the
  // sub-processors list without authentication.
  { mod: Sub, namespace: "legal.subprocessors", label: "sous-processeurs", indexable: true },
] as const

describe("legal placeholder pages metadata", () => {
  for (const c of CASES) {
    const titleKey = "titleKey" in c ? c.titleKey : "title"
    const introKey = "introKey" in c ? c.introKey : "intro"
    it(`${c.label}: generateMetadata interpolates a localized title and description`, async () => {
      const generate = (c.mod as { generateMetadata?: unknown })
        .generateMetadata
      expect(typeof generate).toBe("function")

      const meta = await (generate as (args: {
        params: Promise<{ locale: string }>
      }) => Promise<Record<string, unknown>>)({
        params: Promise.resolve({ locale: "fr" }),
      })

      expect(meta.title).toBe(`[${c.namespace}.${titleKey}] | Marketplace Service`)
      expect(meta.description).toBe(`[${c.namespace}.${introKey}]`)
      if (c.indexable) {
        // Stripe + DSA art. 14: the legal index MUST be crawlable.
        expect(meta.robots).toBeUndefined()
      } else {
        expect(meta.robots).toEqual({ index: false, follow: false })
      }
    })
  }
})

// /cgu and /cgv used to be empty placeholder shells that only rendered
// a "this page will be completed soon" banner. They now permanently
// redirect to the canonical, legally-reviewed full documents so there
// is a single source of truth and zero content-less duplicate URL.
describe("/cgu and /cgv redirect to the canonical full documents", () => {
  const REDIRECTS = [
    { mod: Cgu, label: "cgu", target: "/legal/cgu" },
    { mod: Cgv, label: "cgv", target: "/legal/cgv" },
  ] as const

  for (const r of REDIRECTS) {
    it(`${r.label}: redirects to ${r.target} (no metadata, no placeholder shell)`, async () => {
      redirectMock.mockClear()
      // These pages intentionally export no generateMetadata.
      expect(
        (r.mod as { generateMetadata?: unknown }).generateMetadata,
      ).toBeUndefined()

      const Page = (r.mod as { default: unknown }).default as (args: {
        params: Promise<{ locale: string }>
      }) => Promise<unknown>
      await Page({ params: Promise.resolve({ locale: "fr" }) })

      expect(redirectMock).toHaveBeenCalledWith({
        href: r.target,
        locale: "fr",
      })
    })
  }
})
