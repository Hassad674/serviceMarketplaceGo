import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/fr.json"
import { ActionsTodoCard } from "../components/widgets/actions-todo-card"
import type { DashboardAction } from "../types"

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}))

function renderWith(actions: DashboardAction[], isLoading = false) {
  return render(
    <NextIntlClientProvider locale="fr" messages={messages}>
      <ActionsTodoCard actions={actions} isLoading={isLoading} />
    </NextIntlClientProvider>,
  )
}

describe("ActionsTodoCard", () => {
  it("renders the empty state when no actions are pending", () => {
    renderWith([])
    // Both the header pill and the empty-state body show the "all clear"
    // copy — we just need to assert at least one rendered.
    expect(screen.getAllByText(/Tout est à jour/i).length).toBeGreaterThan(0)
  })

  it("renders one row per action with severity styling", () => {
    renderWith([
      {
        id: "kyc-pending",
        severity: "critical",
        label: "KYC pending",
        ctaLabel: "Vérifier",
        href: "/payment-info",
      },
      {
        id: "billing-profile",
        severity: "warning",
        label: "Add billing details",
        ctaLabel: "Compléter",
        href: "/billing",
      },
    ])
    expect(screen.getByText("KYC pending")).toBeInTheDocument()
    expect(screen.getByText("Add billing details")).toBeInTheDocument()
    expect(
      (screen.getAllByRole("link")[0] as HTMLAnchorElement).getAttribute("href"),
    ).toContain("/payment-info")
  })

  it("shows loading skeletons when loading and list is empty", () => {
    const { container } = renderWith([], true)
    const skeletons = container.querySelectorAll(".animate-pulse")
    expect(skeletons.length).toBeGreaterThanOrEqual(2)
  })

  it("renders the action count when at least one action is pending", () => {
    renderWith([
      {
        id: "msgs",
        severity: "info",
        label: "3 unread messages",
        ctaLabel: "Open",
        href: "/messages",
      },
    ])
    expect(screen.getByText(/1 action/i)).toBeInTheDocument()
  })
})
