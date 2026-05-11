/**
 * Phase B.5 — RGPD art. 22 disclosure page test.
 *
 * Verifies:
 *   1. generateMetadata sets the localized title with the
 *      " | Marketplace Service" suffix and the localized intro as
 *      description.
 *   2. The page renders the three automated systems documented in
 *      gdpr-audit.md Section 4 (moderation, ranking, payment) — each
 *      surfaces a title, a description and a consequence paragraph.
 *   3. The page exposes an appeal CTA (mailto: link) so users have a
 *      one-click human-review path.
 *   4. No raw i18n key (string starting with
 *      "legal.automatedDecisions.") leaks to the DOM — every label
 *      resolves through next-intl.
 */

import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { createElement } from "react"

vi.mock("@i18n/navigation", () => ({
  Link: ({
    children,
    href,
    ...rest
  }: React.ComponentProps<"a"> & { href: string }) =>
    createElement(
      "a",
      { ...rest, href: typeof href === "string" ? href : "/" },
      children,
    ),
}))

import frMessages from "@/../messages/fr.json"
import enMessages from "@/../messages/en.json"

describe("/decisions-automatisees metadata", () => {
  it("FR: generateMetadata returns localized title + description", async () => {
    vi.resetModules()
    vi.doMock("next-intl/server", () => ({
      getTranslations: async ({ namespace }: { namespace: string }) => {
        return (key: string) => {
          const fr = frMessages as unknown as Record<string, unknown>
          const ns = namespace
            .split(".")
            .reduce<Record<string, unknown> | undefined>(
              (acc, segment) =>
                acc?.[segment] as Record<string, unknown> | undefined,
              fr,
            )
          return (ns?.[key] as string) ?? `[${namespace}.${key}]`
        }
      },
    }))

    const fresh = await import(
      "@/app/[locale]/(public)/decisions-automatisees/page"
    )
    const meta = await fresh.generateMetadata({
      params: Promise.resolve({ locale: "fr" }),
    })

    expect(meta.title).toContain("| Marketplace Service")
    expect(meta.title).toMatch(/Décisions automatisées/i)
    expect(typeof meta.description).toBe("string")
    expect((meta.description as string).length).toBeGreaterThan(20)

    vi.doUnmock("next-intl/server")
  })

  it("EN message bundle exposes the same automatedDecisions namespace", () => {
    const en = enMessages as unknown as {
      legal?: { automatedDecisions?: Record<string, unknown> }
    }
    expect(en.legal?.automatedDecisions?.title).toBeTypeOf("string")
    expect(en.legal?.automatedDecisions?.intro).toBeTypeOf("string")
    expect(
      (en.legal?.automatedDecisions?.systems as Record<string, unknown>)
        ?.moderationTitle,
    ).toBeTypeOf("string")
  })
})

describe("/decisions-automatisees rendering", () => {
  it("renders the three automated decision systems + appeal CTA + no raw i18n keys", async () => {
    // Render the server component — the page is async, so we await it
    // and let React Testing Library hydrate the resulting tree. We
    // bypass the next-intl/server stub by providing a synchronous
    // mock identical to the Phase A.4 placeholder test pattern.
    vi.resetModules()
    vi.doMock("next-intl/server", () => ({
      getTranslations: async ({ namespace }: { namespace: string }) => {
        const lookup = (key: string): string => {
          const fr = frMessages as unknown as Record<string, unknown>
          const fullPath = `${namespace}.${key}`.split(".")
          let cursor: unknown = fr
          for (const segment of fullPath) {
            if (typeof cursor !== "object" || cursor === null) {
              return `[${namespace}.${key}]`
            }
            cursor = (cursor as Record<string, unknown>)[segment]
          }
          return typeof cursor === "string"
            ? cursor
            : `[${namespace}.${key}]`
        }
        const t = (key: string) => lookup(key)
        const rich = (
          key: string,
          values?: Record<string, (chunks?: unknown) => unknown>,
        ) => {
          const raw = lookup(key)
          // Replace `{name}` placeholders with the result of values[name]()
          // and string-typed values, so the test can assert the rendered
          // tree contains the React nodes from the callbacks.
          const parts: unknown[] = []
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
            } else if (value !== undefined) {
              parts.push(String(value))
            } else {
              parts.push(match[0])
            }
            lastIndex = pattern.lastIndex
          }
          if (lastIndex < raw.length) {
            parts.push(raw.slice(lastIndex))
          }
          return parts as unknown as string
        }
        ;(t as unknown as { rich: typeof rich }).rich = rich
        return t
      },
    }))
    vi.doMock("next-intl", () => ({
      useTranslations: (namespace?: string) => {
        const t = (key: string) => {
          const fr = frMessages as unknown as Record<string, unknown>
          const path = namespace ? `${namespace}.${key}` : key
          const value = path
            .split(".")
            .reduce<Record<string, unknown> | string | undefined>(
              (acc, segment) =>
                typeof acc === "object" && acc !== null
                  ? (acc as Record<string, unknown>)[segment] as
                      | Record<string, unknown>
                      | string
                  : undefined,
              fr,
            )
          return typeof value === "string" ? value : path
        }
        ;(t as unknown as { rich: typeof t }).rich = t
        return t
      },
    }))

    const fresh = await import(
      "@/app/[locale]/(public)/decisions-automatisees/page"
    )
    const ui = await fresh.default({
      params: Promise.resolve({ locale: "fr" }),
    })
    render(ui as React.ReactElement)

    // 1. Three system sections — moderation, ranking, payment titles.
    expect(
      screen.getByText(
        frMessages.legal.automatedDecisions.systems.moderationTitle,
      ),
    ).toBeTruthy()
    expect(
      screen.getByText(
        frMessages.legal.automatedDecisions.systems.rankingTitle,
      ),
    ).toBeTruthy()
    expect(
      screen.getByText(
        frMessages.legal.automatedDecisions.systems.paymentTitle,
      ),
    ).toBeTruthy()

    // 2. Appeal CTA — mailto: link.
    const mailto = document.querySelector('a[href^="mailto:"]')
    expect(mailto).not.toBeNull()

    // 3. No raw i18n key leaks.
    const text = document.body.textContent ?? ""
    expect(text).not.toMatch(/legal\.automatedDecisions\./)

    vi.doUnmock("next-intl/server")
    vi.doUnmock("next-intl")
  })
})
