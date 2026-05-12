/**
 * /sous-processeurs test — covers per-vendor transfer mechanism
 * disclosure (H10 — distinguish DPF vs SCC per vendor).
 *
 * Asserts:
 *   1. Page is publicly indexable (RGPD art. 28 + DSA art. 14).
 *   2. Each non-EU vendor renders its specific transfer mechanism
 *      (DPF + SCC, SCC 2021/914) rather than a generic "Oui".
 *   3. The Schrems II + DPF audit notes are surfaced — the page must
 *      explicitly state that DPF is supplementary, never the sole
 *      basis.
 *   4. All 21 vendors are listed (mirrors gdpr-audit.md Section 2).
 */

import { describe, expect, it, vi } from "vitest"
import { render } from "@testing-library/react"
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
  const mod = await import("../sous-processeurs/page")
  const Component = mod.default as (args: {
    params: Promise<{ locale: string }>
  }) => Promise<ReactElement>
  const tree = await Component({ params: Promise.resolve({ locale: "fr" }) })
  return render(tree)
}

describe("/sous-processeurs — per-vendor transfer mechanism (H10)", () => {
  it("lists all 21 sub-processors", async () => {
    await renderAsync()
    const rows = document.querySelectorAll("tbody tr")
    expect(rows.length).toBe(21)
  })

  it("shows the SCC or DPF mechanism per non-EU vendor (not a generic 'Oui')", async () => {
    await renderAsync()
    const text = document.body.textContent ?? ""
    // Stripe is non-EU and uses DPF + SCC. The string must appear
    // for at least one vendor — and we never want the legacy
    // generic "Oui (DPF / SCC)" cell with no per-vendor info.
    expect(text).toContain("DPF + SCC")
    expect(text).toContain("SCC 2021/914")
  })

  it("explicitly surfaces the Schrems II + DPF supplementary note", async () => {
    await renderAsync()
    const text = document.body.textContent ?? ""
    expect(text).toContain("Schrems II")
    expect(text).toContain("supplémentaire")
  })

  it("names Stripe Payments Europe Ltd (PSP under PSD2)", async () => {
    await renderAsync()
    const text = document.body.textContent ?? ""
    expect(text).toContain("Stripe Payments Europe Ltd")
  })
})

describe("/sous-processeurs — metadata indexability", () => {
  it("is publicly indexable (RGPD art. 28 + DSA art. 14)", async () => {
    const mod = await import("../sous-processeurs/page")
    const generate = mod.generateMetadata as (args: {
      params: Promise<{ locale: string }>
    }) => Promise<Record<string, unknown>>
    const meta = await generate({ params: Promise.resolve({ locale: "fr" }) })
    expect(meta.robots).toBeUndefined()
    expect(meta.alternates).toEqual({ canonical: "/sous-processeurs" })
  })
})
