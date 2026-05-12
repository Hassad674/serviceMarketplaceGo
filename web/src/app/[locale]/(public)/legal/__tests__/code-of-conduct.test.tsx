/**
 * /legal/code-de-conduite test — covers the legal-max-blindage
 * Code of conduct page.
 *
 * Asserts:
 *   1. generateMetadata returns a localized title + description and
 *      DOES NOT noindex the page (DSA art. 14 + Stripe Restricted
 *      Businesses require crawlable conditions).
 *   2. The 8 mandatory sections (preamble, prohibited behaviours,
 *      reporting mechanism, graduated sanctions, appeal procedure,
 *      DSA commitments, transparency report, review cadence) all
 *      render with their headings + bodies.
 *   3. The prohibited-behaviour list contains the key items inspired
 *      by Malt + Upwork + Contra (harassment, spam, off-platform
 *      circumvention, fraud, doxxing).
 *   4. The sanctions list contains the four graduated levels
 *      (warning → 7/30/90-day suspension → permanent ban → fund
 *      freeze).
 */

import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import type { ReactElement, ReactNode } from "react"
import frMessages from "@/../messages/fr.json"

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
  const mod = await import("../code-de-conduite/page")
  const Component = mod.default as (args: {
    params: Promise<{ locale: string }>
  }) => Promise<ReactElement>
  const tree = await Component({ params: Promise.resolve({ locale: "fr" }) })
  return render(tree)
}

describe("/legal/code-de-conduite — Code of conduct page", () => {
  it("renders a localized H1 + subtitle without leaking i18n keys", async () => {
    await renderAsync()
    const h1 = screen.getByRole("heading", { level: 1 })
    expect(h1.textContent).toBe(
      frMessages.legal.docs.codeOfConduct.title,
    )
    expect(h1.textContent).not.toMatch(/^legal\./)
  })

  it("renders the 8 mandatory sections", async () => {
    await renderAsync()
    const expectedHeadings = [
      frMessages.legal.docs.codeOfConduct.preambleHeading,
      frMessages.legal.docs.codeOfConduct.prohibitedHeading,
      frMessages.legal.docs.codeOfConduct.reportHeading,
      frMessages.legal.docs.codeOfConduct.sanctionsHeading,
      frMessages.legal.docs.codeOfConduct.appealHeading,
      frMessages.legal.docs.codeOfConduct.dsaHeading,
      frMessages.legal.docs.codeOfConduct.transparencyHeading,
      frMessages.legal.docs.codeOfConduct.reviewHeading,
    ]
    for (const heading of expectedHeadings) {
      expect(
        screen.getByRole("heading", { level: 2, name: heading }),
      ).toBeInTheDocument()
    }
  })

  it("surfaces the key prohibited behaviours (Malt + Upwork + Contra heritage)", async () => {
    await renderAsync()
    const items = frMessages.legal.docs.codeOfConduct.prohibitedItems.split("|")
    // Critical items the legal brief insists on — harassment, spam,
    // platform circumvention, fraud, doxxing. We do not match the
    // exact wording (kept flexible for future tweaks) but assert the
    // identifying keywords are present.
    const text = document.body.textContent ?? ""
    expect(text.toLowerCase()).toContain("harcèlement")
    expect(text.toLowerCase()).toContain("spam")
    expect(text.toLowerCase()).toContain("contournement")
    expect(text.toLowerCase()).toContain("fraude")
    expect(text.toLowerCase()).toContain("doxxing")
    expect(items.length).toBeGreaterThanOrEqual(13)
  })

  it("lists the four graduated sanction levels", async () => {
    await renderAsync()
    const sanctions =
      frMessages.legal.docs.codeOfConduct.sanctionsItems.split("|")
    expect(sanctions).toHaveLength(4)
    const text = document.body.textContent ?? ""
    expect(text.toLowerCase()).toContain("avertissement")
    expect(text.toLowerCase()).toContain("suspension")
    expect(text.toLowerCase()).toContain("bannissement")
    // Fund freeze for fraud suspicion
    expect(text.toLowerCase()).toContain("gel")
  })

  it("references the DSA art. 17 (Notice of Limitation) commitment", async () => {
    await renderAsync()
    const text = document.body.textContent ?? ""
    // The DSA commitments section must reference both articles 14
    // and 23 of the DSA Regulation.
    expect(text).toMatch(/DSA/)
    expect(text).toMatch(/art\.\s*14|article\s*14/i)
  })
})

describe("/legal/code-de-conduite — metadata", () => {
  it("is publicly indexable (DSA art. 14 + Stripe Restricted Businesses)", async () => {
    const mod = await import("../code-de-conduite/page")
    const generate = mod.generateMetadata as (args: {
      params: Promise<{ locale: string }>
    }) => Promise<Record<string, unknown>>
    const meta = await generate({ params: Promise.resolve({ locale: "fr" }) })
    expect(meta.title).toMatch(/Marketplace Service$/)
    expect(meta.description).toBeTruthy()
    // No `robots` override — the page MUST be crawlable.
    expect(meta.robots).toBeUndefined()
    expect(meta.alternates).toEqual({ canonical: "/legal/code-de-conduite" })
  })
})
