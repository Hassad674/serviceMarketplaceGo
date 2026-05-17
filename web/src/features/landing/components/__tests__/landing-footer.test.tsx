import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import frMessages from "@/../messages/fr.json"
import { LandingFooter } from "../landing-footer"

// Regression coverage for UI-POLISH items 3 & 4:
//   * Item 3 — no dead links: every footer anchor points at a real
//     in-page section (#how-it-works / #pricing / #referrers) or a
//     real route. The "Company" column (About/Contact → "/") and the
//     "Credit system" link (→ "#") are gone entirely.
//   * Item 4 — the old "© Atelier · Made in Paris" line and the X /
//     LinkedIn / Instagram social trio are replaced by a single
//     author signature + one LinkedIn icon to the maintainer's
//     profile.

vi.mock("@i18n/navigation", () => ({
  Link: ({
    href,
    children,
    ...rest
  }: {
    href: string
    children: React.ReactNode
    className?: string
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}))

vi.mock("@i18n/routing", () => ({
  legalPathnames: {
    "/legal/cgu": { fr: "/legal/cgu", en: "/legal/terms" },
    "/legal/politique-confidentialite": {
      fr: "/legal/politique-confidentialite",
      en: "/legal/privacy",
    },
  },
  legalHref: (canonical: string) => canonical,
}))

function renderFooter() {
  return render(
    <NextIntlClientProvider locale="fr" messages={frMessages}>
      <LandingFooter />
    </NextIntlClientProvider>,
  )
}

const AUTHOR_LINKEDIN = "https://www.linkedin.com/in/hassad-s-24714929b/"

describe("LandingFooter — no dead links (item 3)", () => {
  it("renders zero links pointing at a bare '#' or the dead '/' home stub", () => {
    const { container } = renderFooter()
    const anchors = Array.from(container.querySelectorAll("a"))
    const hrefs = anchors
      .map((a) => a.getAttribute("href"))
      .filter((h): h is string => h !== null)

    // No bare-hash anchors (the old understandCredits "#").
    expect(hrefs.filter((h) => h === "#")).toEqual([])
    // The brand mark legitimately points home; no OTHER link may use
    // "/" as a dead destination (old companyAbout / companyContact).
    const homeLinks = anchors.filter((a) => a.getAttribute("href") === "/")
    expect(homeLinks.length).toBeLessThanOrEqual(2) // desktop + mobile brand
    // The brand mark is now an inline SVG logo carrying its accessible
    // name via aria-label (no text node), so assert on that instead.
    homeLinks.forEach((a) =>
      expect(a.getAttribute("aria-label") ?? "").toMatch(
        /DesignedTrust Services/,
      ),
    )
  })

  it("does not render the removed Company column or its links", () => {
    renderFooter()
    expect(screen.queryByText("Entreprise")).toBeNull()
    expect(screen.queryByText("À propos")).toBeNull()
    expect(screen.queryByText("Contact")).toBeNull()
    expect(screen.queryByText("Système de crédits")).toBeNull()
  })

  it("keeps the valid in-page anchors and legal routes", () => {
    const { container } = renderFooter()
    const hrefs = Array.from(container.querySelectorAll("a")).map((a) =>
      a.getAttribute("href"),
    )
    expect(hrefs).toContain("#how-it-works")
    expect(hrefs).toContain("#pricing")
    expect(hrefs).toContain("#referrers")
    expect(hrefs).toContain("/legal/cgu")
    expect(hrefs).toContain("/legal/politique-confidentialite")
  })
})

describe("LandingFooter — author signature + LinkedIn (item 4)", () => {
  it("removes the old Atelier copyright and the X/Instagram socials", () => {
    renderFooter()
    expect(screen.queryByText(/Made in Paris/i)).toBeNull()
    expect(screen.queryByText(/Atelier ·/)).toBeNull()
    expect(screen.queryByText(/^X$/)).toBeNull()
    expect(screen.queryByText(/Instagram/i)).toBeNull()
  })

  it("renders the DesignedTrust author signature with the current year", () => {
    renderFooter()
    const year = String(new Date().getFullYear())
    const matches = screen.getAllByText(
      (_, node) =>
        node?.textContent ===
        `DesignedTrust Services ${year} — made with  by Hassad Smara`,
    )
    expect(matches.length).toBeGreaterThan(0)
  })

  it("links a single LinkedIn icon to the maintainer profile (new tab, safe rel)", () => {
    renderFooter()
    const linkedinLinks = screen
      .getAllByRole("link", { name: /LinkedIn — Hassad Smara/i })
      .filter((a) => a.getAttribute("href") === AUTHOR_LINKEDIN)
    expect(linkedinLinks.length).toBeGreaterThan(0)
    linkedinLinks.forEach((a) => {
      expect(a).toHaveAttribute("target", "_blank")
      expect(a.getAttribute("rel")).toContain("noopener")
      expect(a.getAttribute("rel")).toContain("noreferrer")
    })
  })
})
