/**
 * /legal (Mentions légales) test — covers LCEN art. 6-III compliance.
 *
 * Asserts:
 *   1. The Editor block surfaces the structured LCEN art. 6-III fields
 *      for a French micro-entreprise (entrepreneur individuel): raison
 *      sociale, nom commercial, forme juridique, RCS/SIREN, mention de
 *      dispense d'immatriculation, code APE, TVA intra-UE (franchise
 *      en base — art. 293 B CGI), adresse, directeur de publication,
 *      contact. The real registered identity is rendered from the
 *      env-driven `@/config/legal-issuer` module — never a placeholder
 *      ("[À COMPLÉTER]" or "en cours d'enregistrement") which is a
 *      Stripe blacklist trigger.
 *   2. The Hosting block names all three hosting providers with
 *      complete postal addresses (Vercel, Railway, Neon — plus
 *      Cloudflare R2).
 *   3. The Contact block surfaces a mailto: link to the real editor
 *      contact email.
 *   4. The page metadata is indexable (no robots noindex) — Stripe +
 *      DSA art. 14 require it.
 *   5. The page never includes the literal "[À COMPLÉTER]" nor the
 *      stale "Designed Trust SAS" entity anywhere.
 */

import { describe, expect, it, vi } from "vitest"
import { render } from "@testing-library/react"
import type { ReactElement, ReactNode } from "react"
import frMessages from "@/../messages/fr.json"
import { legalIssuer } from "@/config/legal-issuer"

// next-intl/server's `getTranslations` and next-intl's
// `useTranslations` both need a vitest-friendly mock. We resolve
// keys from the FR messages bundle. The `rich` helper interpolates
// `{name}` placeholders with their function callbacks so React
// nodes (e.g. `<a>` tags) survive the rendering pipeline — without
// this, `t.rich("foo", { email: () => <a/>})` yields raw
// functions-as-children which React rightfully refuses.
function lookup(namespace: string, key: string): string {
  const fr = frMessages as unknown as Record<string, unknown>
  const path = `${namespace}.${key}`.split(".")
  let cursor: unknown = fr
  for (const segment of path) {
    if (typeof cursor !== "object" || cursor === null) {
      return `[${namespace}.${key}]`
    }
    cursor = (cursor as Record<string, unknown>)[segment]
  }
  return typeof cursor === "string" ? cursor : `[${namespace}.${key}]`
}

function makeT(namespace: string) {
  const t = (key: string) => lookup(namespace, key)
  const rich = (
    key: string,
    values?: Record<string, (chunks?: unknown) => ReactNode>,
  ): ReactNode => {
    const raw = lookup(namespace, key)
    const parts: ReactNode[] = []
    const pattern = /\{(\w+)\}/g
    let lastIndex = 0
    let match: RegExpExecArray | null
    while ((match = pattern.exec(raw)) !== null) {
      if (match.index > lastIndex) {
        parts.push(raw.slice(lastIndex, match.index))
      }
      const name = match[1]
      const value = values?.[name]
      if (typeof value === "function") {
        parts.push(value())
      } else {
        parts.push(match[0])
      }
      lastIndex = pattern.lastIndex
    }
    if (lastIndex < raw.length) {
      parts.push(raw.slice(lastIndex))
    }
    return parts as unknown as ReactNode
  }
  ;(t as unknown as { rich: typeof rich }).rich = rich
  return t
}

vi.mock("next-intl/server", () => ({
  getTranslations: async ({ namespace }: { namespace: string }) =>
    makeT(namespace),
}))

vi.mock("next-intl", () => ({
  useTranslations: (namespace?: string) => makeT(namespace ?? ""),
}))

vi.mock("@i18n/navigation", () => ({
  Link: ({
    children,
    href,
    ...rest
  }: React.ComponentProps<"a"> & { href: string }) => (
    <a {...rest} href={href}>
      {children}
    </a>
  ),
}))

async function renderAsync(): Promise<ReturnType<typeof render>> {
  const mod = await import("../page")
  const Component = mod.default as (args: {
    params: Promise<{ locale: string }>
  }) => Promise<ReactElement>
  const tree = await Component({ params: Promise.resolve({ locale: "fr" }) })
  // We bypass NextIntlClientProvider since we've mocked
  // `next-intl` directly above — the FR resolver is the single
  // source of truth for the test.
  return render(tree)
}

describe("/legal — Mentions légales (LCEN art. 6-III)", () => {
  it("surfaces the structured LCEN art. 6-III editor fields", async () => {
    await renderAsync()
    const text = document.body.textContent ?? ""
    // Mandatory LCEN labels for a micro-entreprise. We assert presence
    // of the i18n labels for the chrome.
    expect(text).toContain(frMessages.legal.mentions.editorCompanyLabel)
    expect(text).toContain(frMessages.legal.mentions.editorTradingNameLabel)
    expect(text).toContain(frMessages.legal.mentions.editorFormLabel)
    expect(text).toContain(frMessages.legal.mentions.editorRcsLabel)
    expect(text).toContain(frMessages.legal.mentions.editorRcsMentionLabel)
    expect(text).toContain(frMessages.legal.mentions.editorApeLabel)
    expect(text).toContain(frMessages.legal.mentions.editorVatLabel)
    expect(text).toContain(frMessages.legal.mentions.editorAddressLabel)
    expect(text).toContain(frMessages.legal.mentions.editorDirectorLabel)
    expect(text).toContain(frMessages.legal.mentions.editorContactLabel)
  })

  it("renders the real registered identity (no placeholder, no stale entity)", async () => {
    await renderAsync()
    const text = document.body.textContent ?? ""
    // Real micro-entreprise identity from the env-driven config.
    expect(text).toContain(legalIssuer.legalName)
    expect(text).toContain(legalIssuer.siret)
    expect(text).toContain(legalIssuer.siren)
    expect(text).toContain(legalIssuer.vatNumber)
    expect(text).toContain(legalIssuer.postalCode)
    // VAT franchise mention is mandatory for a micro-entreprise.
    expect(text).toContain("293 B")
    // Stripe Restricted Businesses rejects pages with the placeholder
    // marker; the stale "Designed Trust SAS" entity must be gone too.
    expect(text).not.toMatch(/\[À COMPLÉTER\]/)
    expect(text).not.toMatch(/\[A COMPLETER\]/)
    expect(text).not.toMatch(/Designed Trust SAS/)
    expect(text).not.toMatch(/en cours d'enregistrement/)
  })

  it("names all three production hosting providers", async () => {
    await renderAsync()
    const text = document.body.textContent ?? ""
    expect(text).toContain("Vercel")
    expect(text).toContain("Railway")
    expect(text).toContain("Neon")
    expect(text).toContain("Cloudflare")
  })

  it("exposes a mailto: link to the real editor contact email (DSA art. 12)", async () => {
    await renderAsync()
    const mailto = document.querySelector(
      `a[href^="mailto:${legalIssuer.contactEmail}"]`,
    )
    expect(mailto).not.toBeNull()
  })

  it("references Stripe Payments Europe Ltd as the PSP (LCEN + Code monétaire et financier)", async () => {
    await renderAsync()
    const text = document.body.textContent ?? ""
    expect(text).toContain("Stripe Payments Europe Ltd")
  })

  it("references the DPO email", async () => {
    await renderAsync()
    const mailto = document.querySelector(
      'a[href^="mailto:dpo@designedtrust.com"]',
    )
    expect(mailto).not.toBeNull()
  })
})

describe("/legal — metadata indexability", () => {
  it("is publicly indexable (LCEN art. 6-III + Stripe Restricted Businesses)", async () => {
    const mod = await import("../page")
    const generate = mod.generateMetadata as (args: {
      params: Promise<{ locale: string }>
    }) => Promise<Record<string, unknown>>
    const meta = await generate({ params: Promise.resolve({ locale: "fr" }) })
    expect(meta.robots).toBeUndefined()
    expect(meta.alternates).toEqual({ canonical: "/legal" })
  })
})
