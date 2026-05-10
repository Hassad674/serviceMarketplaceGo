/**
 * Phase A.4 + A.5 — LegalFooter test.
 *
 * Asserts:
 *   1. All 6 legal placeholder routes are linked (privacy, cookies,
 *      legal, cgu, cgv, sous-processeurs).
 *   2. The DPO email (NEXT_PUBLIC_DPO_EMAIL or fallback) is rendered
 *      as a mailto: link so visitors have a one-click RGPD contact.
 *   3. No raw i18n key (string starting with "legal.footer.") leaks to
 *      the DOM — every label resolves through next-intl.
 */

import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { createElement } from "react"
import { NextIntlClientProvider } from "next-intl"
import frMessages from "@/../messages/fr.json"
import { LegalFooter } from "../legal-footer"

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

function renderFooter() {
  return render(
    <NextIntlClientProvider locale="fr" messages={frMessages}>
      <LegalFooter />
    </NextIntlClientProvider>,
  )
}

describe("LegalFooter", () => {
  it("links to all legal routes including /decisions-automatisees (RGPD art. 22)", () => {
    renderFooter()
    const expected = [
      "/privacy",
      "/cookies",
      "/legal",
      "/cgu",
      "/cgv",
      "/sous-processeurs",
      "/decisions-automatisees",
    ]
    for (const href of expected) {
      const link = document.querySelector(`a[href="${href}"]`)
      expect(link, `expected anchor with href=${href}`).not.toBeNull()
    }
  })

  it("exposes the DPO contact via a mailto: link", () => {
    renderFooter()
    const mailto = document.querySelector('a[href^="mailto:"]')
    expect(mailto).not.toBeNull()
    expect((mailto as HTMLAnchorElement).href.startsWith("mailto:")).toBe(true)
  })

  it("renders no raw legal.footer.* i18n keys", () => {
    renderFooter()
    const text = screen.getByRole("contentinfo").textContent ?? ""
    expect(text).not.toMatch(/legal\.footer\./)
  })
})
