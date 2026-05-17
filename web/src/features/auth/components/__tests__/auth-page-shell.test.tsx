import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/fr.json"
import { AuthPageShell } from "../auth-page-shell"

// AuthPageShell tests — verify the split layout shared by login,
// forgot-password and reset-password renders the correct slots and
// the editorial hero copy stays identical across pages.

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
  useRouter: () => ({
    push: vi.fn(),
    replace: vi.fn(),
    back: vi.fn(),
    prefetch: vi.fn(),
  }),
}))

vi.mock("@/shared/hooks/use-theme", () => ({
  useTheme: () => ({ theme: "light", toggle: vi.fn() }),
}))

function renderShell(props: React.ComponentProps<typeof AuthPageShell>) {
  return render(
    <NextIntlClientProvider locale="fr" messages={messages}>
      <AuthPageShell {...props} />
    </NextIntlClientProvider>,
  )
}

describe("AuthPageShell", () => {
  const baseProps = {
    eyebrow: "Bon retour",
    titlePrefix: "Reprends là",
    titleAccent: "où tu en étais.",
    subtitle: "Connecte-toi pour suivre tes missions.",
    children: <div data-testid="form-slot">FORM</div>,
  }

  it("renders eyebrow above the H1", () => {
    renderShell(baseProps)
    expect(screen.getByText("Bon retour")).toBeInTheDocument()
  })

  it("renders titlePrefix and titleAccent inside a single H1", () => {
    renderShell(baseProps)
    const heading = screen.getByRole("heading", { level: 1 })
    expect(heading).toHaveTextContent("Reprends là où tu en étais.")
  })

  it("renders the subtitle paragraph", () => {
    renderShell(baseProps)
    expect(
      screen.getByText("Connecte-toi pour suivre tes missions."),
    ).toBeInTheDocument()
  })

  it("renders the children slot", () => {
    renderShell(baseProps)
    expect(screen.getByTestId("form-slot")).toBeInTheDocument()
  })

  it("renders the editorial hero with the marketplace value prop", () => {
    renderShell(baseProps)
    // Right column hero — H2 carries the heroTitle (split across two
    // italic accents) — assert the literal join is present.
    expect(
      screen.getByRole("heading", { level: 2 }),
    ).toHaveTextContent(messages.auth.heroTitlePart1)
  })

  it("renders the three trust pillars from the auth.pillar.* namespace", () => {
    renderShell(baseProps)
    expect(
      screen.getByText(messages.auth.pillar.secure.label),
    ).toBeInTheDocument()
    expect(
      screen.getByText(messages.auth.pillar.verified.label),
    ).toBeInTheDocument()
    expect(
      screen.getByText(messages.auth.pillar.noFee.label),
    ).toBeInTheDocument()
  })

  it("states the 0% commission positioning in the hero and the trust pillar", () => {
    renderShell(baseProps)
    // Hero H2 — punchier value prop: keep 100% of your rate, no cut.
    const hero = screen.getByRole("heading", { level: 2 })
    expect(hero).toHaveTextContent(/100\s*% de ton tarif/i)
    expect(hero).toHaveTextContent(/aucune part/i)
    // Third pillar — was the vague "Sans commission cachée", now an
    // explicit "0 % de commission" promise (keep 100% of what you bill).
    expect(
      screen.getByText(/^0\s*% de commission$/i),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/tu gardes 100\s*% de ce que tu factures/i),
    ).toBeInTheDocument()
    // No leftover "commission cachée" / "frais cachés" wording.
    expect(screen.queryByText(/commission cachée/i)).toBeNull()
    expect(screen.queryByText(/frais cachés/i)).toBeNull()
  })

  it("links the brand mark to the home page", () => {
    renderShell(baseProps)
    const brandLink = screen.getByRole("link", {
      name: /DesignedTrust Services/,
    })
    expect(brandLink).toHaveAttribute("href", "/")
  })

  it("links the footer terms to /legal/cgu and privacy to /legal/politique-confidentialite", () => {
    renderShell(baseProps)
    const termsLink = screen.getByRole("link", {
      name: messages.auth.terms,
    })
    const privacyLink = screen.getByRole("link", {
      name: messages.auth.privacy,
    })
    // Phase legal-max-blindage — the auth shell now points at the
    // canonical legal routes (CGU lives at /legal/cgu, single privacy
    // policy at /legal/politique-confidentialite). The stub /terms
    // and /privacy URLs would 404 in production.
    expect(termsLink).toHaveAttribute("href", "/legal/cgu")
    expect(privacyLink).toHaveAttribute("href", "/legal/politique-confidentialite")
  })

  it("uses the same hero copy regardless of page-specific eyebrow / title", () => {
    const { rerender } = renderShell(baseProps)
    // Hero H2 is rendered as part of the shell — it stays identical
    // when the page swaps its own eyebrow / title / subtitle.
    const heroBefore = screen.getByRole("heading", { level: 2 }).textContent
    expect(heroBefore).toContain(messages.auth.heroTitlePart1)

    rerender(
      <NextIntlClientProvider locale="fr" messages={messages}>
        <AuthPageShell
          {...baseProps}
          eyebrow="Mot de passe oublié"
          titlePrefix="Mot de passe"
          titleAccent="oublié ?"
          subtitle="Donne ton email."
        />
      </NextIntlClientProvider>,
    )

    const heroAfter = screen.getByRole("heading", { level: 2 }).textContent
    expect(heroAfter).toBe(heroBefore)
    // And the page-specific H1 reflects the new props.
    expect(
      screen.getByRole("heading", { level: 1 }),
    ).toHaveTextContent("Mot de passe oublié ?")
  })
})
