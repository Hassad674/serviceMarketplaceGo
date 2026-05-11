/**
 * /cookies page — table rendering pinned against the
 * COOKIE_INVENTORY config.
 *
 * Covers:
 *   1. The page renders one row per entry in `COOKIE_INVENTORY`.
 *   2. Each row uses the entry's stable `key` (asserted via the i18n
 *      path `legal.cookies.rows.<key>.*`) so the table cannot silently
 *      drop a cookie when a category is updated.
 *   3. The LegalShell wrapper is in place (page is gated by the legal
 *      placeholder shell, not raw markup).
 *
 * The page is a Server Component that resolves an async `params`
 * promise. We import it dynamically after stubbing next-intl/server so
 * the test runtime stays node-friendly.
 */
import { describe, expect, it, vi } from "vitest"
import { render } from "@testing-library/react"
import { createElement } from "react"
import frMessages from "@/../messages/fr.json"
import { COOKIE_INVENTORY } from "@/shared/lib/cookie-consent-config"

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

function resolveFromFr(namespace: string, key: string): string {
  // Walk `legal.cookies.rows.<entry>.<field>` (or whatever the
  // namespace points at) inside the FR bundle. Falls back to a
  // sentinel string so missing keys surface visibly in the test
  // failure output.
  const path = namespace.split(".").concat(key.split("."))
  let cursor: unknown = frMessages
  for (const segment of path) {
    if (typeof cursor !== "object" || cursor === null) {
      return `MISSING:${namespace}.${key}`
    }
    cursor = (cursor as Record<string, unknown>)[segment]
  }
  return typeof cursor === "string" ? cursor : `MISSING:${namespace}.${key}`
}

describe("/cookies page table", () => {
  it("renders one row per COOKIE_INVENTORY entry with the stable key", async () => {
    vi.doMock("next-intl/server", () => ({
      getTranslations: async ({ namespace }: { namespace: string }) => {
        return (key: string) => resolveFromFr(namespace, key)
      },
    }))
    vi.doMock("next-intl", () => ({
      useTranslations: (namespace?: string) => {
        const t = (key: string) =>
          namespace
            ? resolveFromFr(namespace, key)
            : resolveFromFr("legal", key)
        ;(t as unknown as { rich: typeof t }).rich = t
        return t
      },
    }))

    const fresh = await import("@/app/[locale]/(public)/cookies/page")
    const ui = await fresh.default({
      params: Promise.resolve({ locale: "fr" }),
    })
    const { container } = render(ui as React.ReactElement)

    // One tbody row per inventory entry.
    const rows = container.querySelectorAll("tbody tr")
    expect(rows.length).toBe(COOKIE_INVENTORY.length)

    // Each entry's "name" translation must appear in the table — this
    // is the assertion that the page iterates over COOKIE_INVENTORY
    // (and not a hardcoded ad-hoc list).
    const text = container.textContent ?? ""
    for (const entry of COOKIE_INVENTORY) {
      const expectedName = resolveFromFr(
        "legal.cookies.rows",
        `${entry.key}.name`,
      )
      expect(
        text,
        `expected cookies page to render row '${entry.key}' (resolved name: ${expectedName})`,
      ).toContain(expectedName)
    }

    vi.doUnmock("next-intl/server")
    vi.doUnmock("next-intl")
  })

  it("generateMetadata emits noindex + localized title for /cookies", async () => {
    vi.doMock("next-intl/server", () => ({
      getTranslations: async ({ namespace }: { namespace: string }) => {
        return (key: string) => resolveFromFr(namespace, key)
      },
    }))

    const fresh = await import("@/app/[locale]/(public)/cookies/page")
    const meta = await fresh.generateMetadata({
      params: Promise.resolve({ locale: "fr" }),
    })

    expect(meta.title).toContain("| Marketplace Service")
    expect(typeof meta.description).toBe("string")
    expect(meta.robots).toEqual({ index: false, follow: false })

    vi.doUnmock("next-intl/server")
  })
})
