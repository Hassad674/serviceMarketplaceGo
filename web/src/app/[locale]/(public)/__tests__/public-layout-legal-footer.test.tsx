/**
 * PublicLayout — LegalFooter wiring regression.
 *
 * Asserts that the `/[locale]/(public)/layout.tsx` shell renders the
 * sitewide `<LegalFooter />` in BOTH the authenticated and
 * unauthenticated branches. This is the page-level guarantee that the
 * 7 legal routes (privacy, cookies, legal, cgu, cgv, sous-processeurs,
 * decisions-automatisees) reach every public surface — without this
 * wiring, the legal-footer component (covered separately in
 * `legal-footer.test.tsx`) would never render on the routes that need
 * it the most.
 *
 * Regression: when the auth branch was added to the layout, the
 * footer was at risk of being moved only into the unauth branch.
 * These tests pin both branches.
 */
import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import frMessages from "@/../messages/fr.json"

// next-intl Link shim — the LegalFooter renders <Link> nodes.
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

// PublicNavbar pulls in a stack of unrelated hooks; stub it so the
// layout test focuses on the footer wiring.
vi.mock("@/shared/components/layouts/public-navbar", () => ({
  PublicNavbar: () => <header data-testid="stub-public-navbar" />,
}))

// DashboardShell same story — stub it down to a passthrough so the
// authenticated branch can be rendered without the sidebar wiring.
vi.mock("@/shared/components/layouts/dashboard-shell", () => ({
  DashboardShell: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="stub-dashboard-shell">{children}</div>
  ),
}))

// useUser is the gate that picks the auth vs unauth branch — flip it
// per-test via this ref.
const userState: { current: { isLoading: boolean; data: unknown } } = {
  current: { isLoading: false, data: null },
}
vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => userState.current,
}))

import PublicLayout from "../layout"

function renderLayout() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={qc}>
      <NextIntlClientProvider locale="fr" messages={frMessages}>
        <PublicLayout>
          <main data-testid="page-content">page</main>
        </PublicLayout>
      </NextIntlClientProvider>
    </QueryClientProvider>,
  )
}

describe("PublicLayout — LegalFooter wiring", () => {
  it("renders LegalFooter with all 7 legal links when the visitor is unauthenticated", () => {
    userState.current = { isLoading: false, data: null }
    renderLayout()
    // <footer role="contentinfo"> is emitted by the LegalFooter component.
    const footer = screen.getByRole("contentinfo")
    expect(footer).toBeInTheDocument()
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
      const link = footer.querySelector(`a[href="${href}"]`)
      expect(link, `expected footer anchor with href=${href}`).not.toBeNull()
    }
  })

  it("renders LegalFooter with all 7 legal links when the visitor is authenticated", () => {
    userState.current = {
      isLoading: false,
      data: { id: "u-1", role: "provider" },
    }
    renderLayout()
    const footer = screen.getByRole("contentinfo")
    expect(footer).toBeInTheDocument()
    // DashboardShell stub branch was used — verify both elements live.
    expect(screen.getByTestId("stub-dashboard-shell")).toBeInTheDocument()
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
      const link = footer.querySelector(`a[href="${href}"]`)
      expect(link, `expected footer anchor with href=${href}`).not.toBeNull()
    }
  })

  it("renders the skeleton (no LegalFooter) while user state is loading", () => {
    userState.current = { isLoading: true, data: undefined }
    renderLayout()
    // During the loading branch, the layout returns its skeleton and
    // does NOT render the footer — the footer becomes visible only once
    // the auth branch is known. Asserting absence locks this behaviour.
    expect(screen.queryByRole("contentinfo")).toBeNull()
  })
})
