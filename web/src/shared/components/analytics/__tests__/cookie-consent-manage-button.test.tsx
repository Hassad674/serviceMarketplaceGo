/**
 * CookieConsentManageButton test — covers the CNIL Recommendation
 * 2020 §6.3 "withdrawal as easy as consent" affordance.
 *
 * Asserts:
 *   1. The button renders a localised label + ARIA description.
 *   2. The underlying anchor points to `/cookies` so the click works
 *      even without JS (the CMP `showPreferences()` is the JS
 *      enhancement).
 *   3. Clicking the button invokes `CookieConsent.showPreferences()`
 *      and prevents the default anchor navigation when the CMP is
 *      loaded — otherwise it falls back to the natural `/cookies`
 *      navigation (preserving the no-JS path).
 *   4. The `floating` variant adds the fixed-position classes; the
 *      `inline` variant stays in the document flow.
 */

import { describe, expect, it, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import frMessages from "@/../messages/fr.json"

// Capture all interactions with CookieConsent.showPreferences so we
// can assert the click handler wires it correctly.
const showPreferencesSpy = vi.fn()
vi.mock("vanilla-cookieconsent", () => ({
  showPreferences: () => showPreferencesSpy(),
}))

vi.mock("@i18n/navigation", () => ({
  Link: ({
    children,
    href,
    onClick,
    ...rest
  }: React.ComponentProps<"a"> & { href: string }) => (
    <a
      {...rest}
      href={href}
      onClick={onClick as React.MouseEventHandler<HTMLAnchorElement>}
    >
      {children}
    </a>
  ),
}))

import { CookieConsentManageButton } from "../cookie-consent-manage-button"

function renderButton(variant?: "inline" | "floating") {
  return render(
    <NextIntlClientProvider locale="fr" messages={frMessages}>
      <CookieConsentManageButton variant={variant} />
    </NextIntlClientProvider>,
  )
}

beforeEach(() => {
  showPreferencesSpy.mockClear()
})

describe("CookieConsentManageButton — CNIL §6.3", () => {
  it("renders the i18n label", () => {
    renderButton()
    expect(
      screen.getByText(frMessages.cookieConsent.banner.manageLabel),
    ).toBeInTheDocument()
  })

  it("exposes a localised aria-label", () => {
    renderButton()
    const link = screen.getByRole("link", {
      name: frMessages.cookieConsent.banner.manageAria,
    })
    expect(link).toBeInTheDocument()
    expect(link).toHaveAttribute("href", "/cookies")
  })

  it("invokes CookieConsent.showPreferences() on click", () => {
    renderButton()
    const link = screen.getByRole("link", {
      name: frMessages.cookieConsent.banner.manageAria,
    })
    fireEvent.click(link)
    expect(showPreferencesSpy).toHaveBeenCalledTimes(1)
  })

  it("applies fixed-position classes for the floating variant", () => {
    renderButton("floating")
    const link = screen.getByRole("link", {
      name: frMessages.cookieConsent.banner.manageAria,
    })
    expect(link.className).toMatch(/fixed/)
    expect(link.className).toMatch(/bottom-4/)
    expect(link.className).toMatch(/left-4/)
  })

  it("stays in document flow for the inline variant", () => {
    renderButton("inline")
    const link = screen.getByRole("link", {
      name: frMessages.cookieConsent.banner.manageAria,
    })
    expect(link.className).not.toMatch(/fixed/)
    expect(link.className).toMatch(/inline-flex/)
  })
})
