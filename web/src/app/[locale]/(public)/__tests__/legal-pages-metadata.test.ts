/**
 * Phase A.4 — placeholder legal route metadata tests.
 *
 * The 6 placeholder pages (/privacy, /cookies, /legal, /cgu, /cgv,
 * /sous-processeurs) all expose generateMetadata that:
 *   1. interpolates a localized title with " | Marketplace Service" suffix
 *   2. sets robots noindex/nofollow (placeholders are not indexable
 *      until Phase C content lands)
 *   3. surfaces the localized intro string as the description
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

vi.mock("@i18n/navigation", () => ({
  Link: ({ children }: { children: React.ReactNode }) => children,
}))

import * as Privacy from "@/app/[locale]/(public)/privacy/page"
import * as Cookies from "@/app/[locale]/(public)/cookies/page"
import * as LegalMentions from "@/app/[locale]/(public)/legal/page"
import * as Cgu from "@/app/[locale]/(public)/cgu/page"
import * as Cgv from "@/app/[locale]/(public)/cgv/page"
import * as Sub from "@/app/[locale]/(public)/sous-processeurs/page"

const CASES = [
  { mod: Privacy, namespace: "legal.privacy", label: "privacy" },
  { mod: Cookies, namespace: "legal.cookies", label: "cookies" },
  { mod: LegalMentions, namespace: "legal.mentions", label: "legal" },
  { mod: Cgu, namespace: "legal.cgu", label: "cgu" },
  { mod: Cgv, namespace: "legal.cgv", label: "cgv" },
  { mod: Sub, namespace: "legal.subprocessors", label: "sous-processeurs" },
] as const

describe("legal placeholder pages metadata", () => {
  for (const { mod, namespace, label } of CASES) {
    it(`${label}: generateMetadata sets noindex + localized title and description`, async () => {
      const generate = (mod as { generateMetadata?: unknown }).generateMetadata
      expect(typeof generate).toBe("function")

      const meta = await (generate as (args: {
        params: Promise<{ locale: string }>
      }) => Promise<Record<string, unknown>>)({
        params: Promise.resolve({ locale: "fr" }),
      })

      expect(meta.title).toBe(`[${namespace}.title] | Marketplace Service`)
      expect(meta.description).toBe(`[${namespace}.intro]`)
      expect(meta.robots).toEqual({ index: false, follow: false })
    })
  }
})
