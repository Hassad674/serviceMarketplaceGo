// Tests for PaymentCheckoutShell — the minimal chrome rendered on the
// /projects/pay client checkout. The shell deliberately strips the
// dashboard chrome (sidebar + top header) so the page reads as a
// focused single-task flow.
//
// Regression target: the original page inherited (app)/layout.tsx's
// DashboardShell, which presented a sidebar AND a top header on top
// of the page's own editorial banner — the "deux navbar" bug.

import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"

// next-intl mock returns the key so we can assert on the i18n contract
// rather than the rendered French copy.
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => `i18n:${key}`,
}))

// `@i18n/navigation` is the localised Link wrapper. Stub to a plain
// anchor so the test doesn't need the next-intl provider stack.
vi.mock("@i18n/navigation", () => ({
  Link: ({
    href,
    children,
    className,
  }: {
    href: string
    children: React.ReactNode
    className?: string
  }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
}))

// lucide icons render as inert spans — keeps the DOM uncluttered.
vi.mock("lucide-react", () => ({
  ArrowLeft: () => <span data-testid="icon-arrow-left" />,
}))

async function renderShell(children: React.ReactNode) {
  const { PaymentCheckoutShell } = await import("../payment-checkout-shell")
  return render(<PaymentCheckoutShell>{children}</PaymentCheckoutShell>)
}

describe("PaymentCheckoutShell", () => {
  it("renders the supplied children inside the main element", async () => {
    await renderShell(<p data-testid="content">payment body</p>)
    const main = screen.getByRole("main")
    expect(main).toBeInTheDocument()
    expect(main).toContainElement(screen.getByTestId("content"))
  })

  it("renders the brand wordmark linking back to /", async () => {
    await renderShell(<p>x</p>)
    const brand = screen.getByText("i18n:proposalFlow_pay_shellBrand")
    expect(brand).toBeInTheDocument()
    expect(brand.closest("a")?.getAttribute("href")).toBe("/")
  })

  it("renders the back-to-dashboard link with the canonical href", async () => {
    await renderShell(<p>x</p>)
    const backLink = screen.getByText(
      "i18n:proposalFlow_pay_shellBackLink",
    )
    expect(backLink).toBeInTheDocument()
    expect(backLink.closest("a")?.getAttribute("href")).toBe("/dashboard")
  })

  it("does NOT render any dashboard chrome (no nav, no aside, no DashboardShell marker)", async () => {
    // The whole point of the shell is to escape the dashboard chrome.
    // The dashboard shell exposes a sidebar (Sidebar component) and a
    // top Header. Neither must surface here.
    const { container } = await renderShell(<p>x</p>)
    // No <nav>, no <aside> — these would be the sidebar's structural
    // hooks if it had leaked into the shell.
    expect(container.querySelector("nav")).toBeNull()
    expect(container.querySelector("aside")).toBeNull()
    // The dashboard sidebar uses data-testid="sidebar" via the real
    // component; ensure no node carries that hint.
    expect(container.querySelector('[data-testid="sidebar"]')).toBeNull()
  })

  it("uses exactly one header and one main landmark (a11y: single landmark each)", async () => {
    const { container } = await renderShell(<p>x</p>)
    expect(container.querySelectorAll("header").length).toBe(1)
    expect(container.querySelectorAll("main").length).toBe(1)
  })

  it("uses Soleil v2 semantic tokens (no hardcoded hex colors)", async () => {
    // Hardcoded hex (#xxxxxx) in className would bypass the Tailwind
    // theme — a regression on the design-system invariant.
    const { container } = await renderShell(<p>x</p>)
    const html = container.innerHTML
    expect(html).not.toMatch(/#[0-9a-fA-F]{3,8}\b/)
  })
})
