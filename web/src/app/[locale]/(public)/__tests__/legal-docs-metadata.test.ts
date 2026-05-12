/**
 * D4 (GDPR Phase C) + legal-max-blindage — metadata regression for the
 * /legal/* doc pages.
 *
 * Each page exposes generateMetadata that:
 *   1. Pulls title + subtitle from the corresponding legal.docs.<key>
 *      namespace.
 *   2. Suffixes the title with " | Marketplace Service".
 *   3. Sets the appropriate `robots` policy:
 *      - Internal / draft documents (Art. 30 register, AIPD, DPA
 *        template) are kept noindex.
 *      - User-facing legal documents (CGU, CGV, Privacy policy, Code
 *        of conduct) MUST be indexable per Stripe Restricted
 *        Businesses + DSA art. 14 + RGPD art. 12 transparency.
 *
 * The pages are imported lazily via dynamic import inside each test to
 * avoid hoisting issues with the next-intl/server mock.
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

vi.mock("@i18n/navigation", () => ({
  Link: ({ children }: { children: React.ReactNode }) => children,
}))

const CASES: ReadonlyArray<{
  path: string
  namespace: string
  indexable: boolean
}> = [
  {
    path: "@/app/[locale]/(public)/legal/registre/page",
    namespace: "legal.docs.registre",
    indexable: false,
  },
  {
    path: "@/app/[locale]/(public)/legal/aipd/page",
    namespace: "legal.docs.aipd",
    indexable: false,
  },
  {
    path: "@/app/[locale]/(public)/legal/dpa-template/page",
    namespace: "legal.docs.dpaTemplate",
    indexable: false,
  },
  // The privacy policy is the canonical single source (after the
  // May 2026 /privacy merge) — MUST be crawlable (RGPD art. 12).
  {
    path: "@/app/[locale]/(public)/legal/politique-confidentialite/page",
    namespace: "legal.docs.politiqueConfidentialite",
    indexable: true,
  },
  // CGU + CGV + Code of conduct — Stripe Restricted Businesses +
  // DSA art. 14 require user-facing contractual documents to be
  // publicly indexable.
  {
    path: "@/app/[locale]/(public)/legal/cgu/page",
    namespace: "legal.docs.cgu",
    indexable: true,
  },
  {
    path: "@/app/[locale]/(public)/legal/cgv/page",
    namespace: "legal.docs.cgv",
    indexable: true,
  },
  {
    path: "@/app/[locale]/(public)/legal/code-de-conduite/page",
    namespace: "legal.docs.codeOfConduct",
    indexable: true,
  },
]

describe("/legal/* doc pages metadata (D4 + legal-max-blindage)", () => {
  for (const { path, namespace, indexable } of CASES) {
    it(`${namespace}: generateMetadata interpolates a localized title and description, indexable=${indexable}`, async () => {
      const mod = await import(path)
      const generate = (mod as { generateMetadata?: unknown })
        .generateMetadata
      expect(typeof generate).toBe("function")
      const meta = await (
        generate as (args: {
          params: Promise<{ locale: string }>
        }) => Promise<Record<string, unknown>>
      )({ params: Promise.resolve({ locale: "fr" }) })

      expect(meta.title).toBe(`[${namespace}.title] | Marketplace Service`)
      expect(meta.description).toBe(`[${namespace}.subtitle]`)
      if (indexable) {
        expect(meta.robots).toBeUndefined()
      } else {
        expect(meta.robots).toEqual({ index: false, follow: false })
      }
    })
  }
})
