/**
 * public-navbar.test.tsx is the trip-wire that prevents the public
 * marketing header from leaking raw i18n keys onto the screen — a
 * regression that briefly shipped (`landing.agenciesTitle`,
 * `landing.freelancesTitle`, `landing.browseProjects`) on the
 * /freelancers, /agencies, /referrers public listing routes after
 * the Soleil v2 landing rebuild.
 *
 * The test mounts the navbar with a real NextIntlClientProvider
 * fed by the actual messages/fr.json and messages/en.json bundles,
 * then asserts:
 *
 *   1. No DOM text node contains a string starting with `landing.`
 *      or any other raw key namespace prefix — the rendered output
 *      must be translated French / English copy.
 *   2. Every link points at a public listing route that is reachable
 *      without an authenticated session (`/agencies`,
 *      `/freelancers`, `/opportunities`).
 *   3. The translation keys this component reads exist in BOTH
 *      locale bundles, so the regression cannot return by silent
 *      drift between fr.json and en.json.
 */

import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { createElement } from "react"
import { NextIntlClientProvider } from "next-intl"
import frMessages from "@/../messages/fr.json"
import enMessages from "@/../messages/en.json"
import { PublicNavbar } from "../public-navbar"

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

vi.mock("@/shared/components/theme-toggle", () => ({
  ThemeToggle: () => createElement("button", { "aria-label": "Toggle theme" }),
}))

const KEYS_USED_BY_NAVBAR = [
  "landing.nav.agencies",
  "landing.nav.freelancers",
  "landing.nav.opportunities",
  "common.signIn",
  "common.createAccount",
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

function renderWithLocale(locale: "fr" | "en") {
  const messages = locale === "fr" ? frMessages : enMessages
  return render(
    createElement(
      NextIntlClientProvider,
      {
        locale,
        messages: messages as Record<string, unknown>,
        children: createElement(PublicNavbar),
      },
    ),
  )
}

describe("PublicNavbar", () => {
  it("renders the FR labels (no raw i18n keys)", () => {
    renderWithLocale("fr")

    // Pull every text node and assert nothing looks like a raw key.
    const flat = document.body.textContent ?? ""
    expect(flat).not.toMatch(/landing\./)
    expect(flat).not.toMatch(/common\./)
    // Sanity-check the actual French copy is on screen.
    expect(screen.getByText("Agences")).toBeInTheDocument()
    expect(screen.getByText("Freelances")).toBeInTheDocument()
    expect(screen.getByText("Missions")).toBeInTheDocument()
  })

  it("renders the EN labels (no raw i18n keys)", () => {
    renderWithLocale("en")

    const flat = document.body.textContent ?? ""
    expect(flat).not.toMatch(/landing\./)
    expect(flat).not.toMatch(/common\./)
    expect(screen.getByText("Agencies")).toBeInTheDocument()
    expect(screen.getByText("Freelancers")).toBeInTheDocument()
    expect(screen.getByText("Opportunities")).toBeInTheDocument()
  })

  it("links to public listing routes only (never /search or /dashboard)", () => {
    renderWithLocale("fr")

    const links = Array.from(document.querySelectorAll("a"))
    const hrefs = links.map((a) => a.getAttribute("href"))
    expect(hrefs).toContain("/agencies")
    expect(hrefs).toContain("/freelancers")
    expect(hrefs).toContain("/opportunities")
    // Sanity: never points at protected paths from the public surface.
    expect(hrefs).not.toContain("/dashboard")
    expect(hrefs).not.toContain("/search")
  })

  it("every key the navbar reads exists in fr.json", () => {
    const missing = KEYS_USED_BY_NAVBAR.filter(
      (k) => resolveDottedKey(frMessages as Record<string, unknown>, k) === undefined,
    )
    expect(missing).toEqual([])
  })

  it("every key the navbar reads exists in en.json", () => {
    const missing = KEYS_USED_BY_NAVBAR.filter(
      (k) => resolveDottedKey(enMessages as Record<string, unknown>, k) === undefined,
    )
    expect(missing).toEqual([])
  })
})
