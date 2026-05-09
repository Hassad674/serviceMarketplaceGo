import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { NextIntlClientProvider } from "next-intl"
import frMessages from "@/../messages/fr.json"
import enMessages from "@/../messages/en.json"
import { LandingSearchBar } from "../landing-search-bar"

/**
 * landing-search-bar.test.tsx covers the only client island of the
 * landing page. The shape of these assertions is deliberate:
 *  - submit + role tab combinations must produce the right URL on the
 *    next-intl router (regression guard for the "no budget" rule)
 *  - popular suggestion chips push the same router with the chip label
 *  - keyboard Enter and click button take the same path
 *
 * The router is mocked at the @i18n/navigation module boundary so the
 * test is hermetic and never hits next/navigation internals.
 */

const mockPush = vi.fn()
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: mockPush, replace: vi.fn(), back: vi.fn() }),
}))

beforeEach(() => {
  mockPush.mockReset()
})

function renderWithIntl(locale: "fr" | "en" = "fr") {
  return render(
    <NextIntlClientProvider
      locale={locale}
      messages={locale === "fr" ? frMessages : enMessages}
    >
      <LandingSearchBar />
    </NextIntlClientProvider>,
  )
}

describe("LandingSearchBar", () => {
  it("submits to /freelancers with the typed query when the freelance tab is active", async () => {
    const user = userEvent.setup()
    renderWithIntl()
    const queryInput = screen.getByLabelText("Recherche")
    await user.type(queryInput, "developer Stripe")
    const submitButton = screen.getByRole("button", {
      name: /lancer la recherche/i,
    })
    await user.click(submitButton)
    expect(mockPush).toHaveBeenCalledTimes(1)
    expect(mockPush).toHaveBeenCalledWith(
      "/freelancers?q=developer+Stripe",
    )
  })

  it("redirects to /referrers when the apporteur tab is selected", async () => {
    const user = userEvent.setup()
    renderWithIntl()
    await user.click(screen.getByRole("tab", { name: /apporteur/i }))
    await user.type(screen.getByLabelText("Recherche"), "tech CTO")
    await user.click(
      screen.getByRole("button", { name: /lancer la recherche/i }),
    )
    expect(mockPush).toHaveBeenCalledWith("/referrers?q=tech+CTO")
  })

  it("redirects to /agencies when the agence tab is selected", async () => {
    const user = userEvent.setup()
    renderWithIntl()
    await user.click(screen.getByRole("tab", { name: /agence/i }))
    await user.type(screen.getByLabelText("Recherche"), "studio branding")
    await user.click(
      screen.getByRole("button", { name: /lancer la recherche/i }),
    )
    expect(mockPush).toHaveBeenCalledWith("/agencies?q=studio+branding")
  })

  it("appends the city when the location field is filled", async () => {
    const user = userEvent.setup()
    renderWithIntl()
    await user.type(screen.getByLabelText("Recherche"), "designer")
    await user.type(screen.getByLabelText("Lieu"), "Paris")
    await user.click(
      screen.getByRole("button", { name: /lancer la recherche/i }),
    )
    expect(mockPush).toHaveBeenCalledWith(
      "/freelancers?q=designer&city=Paris",
    )
  })

  it("submits without query string when both inputs are empty", async () => {
    const user = userEvent.setup()
    renderWithIntl()
    await user.click(
      screen.getByRole("button", { name: /lancer la recherche/i }),
    )
    expect(mockPush).toHaveBeenCalledWith("/freelancers")
  })

  it("submits via Enter keyboard inside the query input", async () => {
    const user = userEvent.setup()
    renderWithIntl()
    const input = screen.getByLabelText("Recherche")
    await user.type(input, "growth lead{Enter}")
    expect(mockPush).toHaveBeenCalledWith("/freelancers?q=growth+lead")
  })

  it("popular suggestion chip clicks navigate with the chip label as query", async () => {
    const user = userEvent.setup()
    renderWithIntl()
    const chip = screen.getByRole("button", { name: "Designer produit senior" })
    await user.click(chip)
    expect(mockPush).toHaveBeenCalledWith(
      "/freelancers?q=Designer+produit+senior",
    )
  })

  it("renders no budget input — regression guard", () => {
    renderWithIntl()
    // Look for any field labelled "Budget" or matching common budget
    // copy. The user explicitly asked for NO budget input on the
    // landing search bar — this assertion locks that.
    expect(screen.queryByLabelText(/budget/i)).toBeNull()
    expect(screen.queryByText(/tous budgets/i)).toBeNull()
    expect(screen.queryByPlaceholderText(/budget/i)).toBeNull()
  })

  it("trims whitespace before redirecting", async () => {
    const user = userEvent.setup()
    renderWithIntl()
    await user.type(screen.getByLabelText("Recherche"), "   designer   ")
    await user.click(
      screen.getByRole("button", { name: /lancer la recherche/i }),
    )
    expect(mockPush).toHaveBeenCalledWith("/freelancers?q=designer")
  })

  it("uses English placeholders when the locale is en", () => {
    renderWithIntl("en")
    expect(
      screen.getByPlaceholderText(/Product designer/i),
    ).toBeInTheDocument()
  })
})
