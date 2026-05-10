import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { NextIntlClientProvider } from "next-intl"
import messages from "@/../messages/fr.json"
import { ProviderDashboard } from "../components/provider-dashboard"
import { EnterpriseDashboard } from "../components/enterprise-dashboard"
import { ReferrerDashboard } from "../components/referrer-dashboard"

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}))

function wrap(children: React.ReactNode) {
  return (
    <NextIntlClientProvider locale="fr" messages={messages}>
      {children}
    </NextIntlClientProvider>
  )
}

describe("ProviderDashboard", () => {
  it("renders the four Provider/Agency stat tiles", () => {
    render(
      wrap(
        <ProviderDashboard
          visibilityStats={{
            organization_id: "x",
            period_days: 7,
            total_views: 42,
            unique_viewers: 30,
            search_appearances: 18,
            avg_search_position: 3.5,
            series: [
              { date: "2026-05-01T00:00:00Z", count: 1 },
              { date: "2026-05-02T00:00:00Z", count: 2 },
            ],
          }}
          isVisibilityLoading={false}
          monthlyRevenueLabel="—"
          pipelineCount={0}
          pipelineCtaHref="/missions"
          actions={[]}
          actionsLoading={false}
        />,
      ),
    )
    expect(screen.getByText(/Vues du profil/i)).toBeInTheDocument()
    expect(screen.getByText(/Apparitions recherche/i)).toBeInTheDocument()
    expect(screen.getByText(/Position moyenne/i)).toBeInTheDocument()
    expect(screen.getByText("42")).toBeInTheDocument()
    expect(screen.getByText("3.5")).toBeInTheDocument()
  })

  it("falls back to em dash when avg_search_position is null", () => {
    render(
      wrap(
        <ProviderDashboard
          visibilityStats={{
            organization_id: "x",
            period_days: 7,
            total_views: 0,
            unique_viewers: 0,
            search_appearances: 0,
            avg_search_position: null,
            series: [],
          }}
          isVisibilityLoading={false}
          monthlyRevenueLabel="—"
          pipelineCount={0}
          pipelineCtaHref="/missions"
          actions={[]}
          actionsLoading={false}
        />,
      ),
    )
    expect(screen.getAllByText("—").length).toBeGreaterThan(0)
  })

  it("renders the link to /stats", () => {
    render(
      wrap(
        <ProviderDashboard
          visibilityStats={undefined}
          isVisibilityLoading={false}
          monthlyRevenueLabel="—"
          pipelineCount={0}
          pipelineCtaHref="/missions"
          actions={[]}
          actionsLoading={false}
        />,
      ),
    )
    const links = screen.getAllByRole("link") as HTMLAnchorElement[]
    expect(links.some((a) => a.getAttribute("href") === "/stats")).toBe(true)
  })

  it("does not render the visibility card data when loading", () => {
    render(
      wrap(
        <ProviderDashboard
          visibilityStats={undefined}
          isVisibilityLoading={true}
          monthlyRevenueLabel="—"
          pipelineCount={0}
          pipelineCtaHref="/missions"
          actions={[]}
          actionsLoading={false}
        />,
      ),
    )
    // Loading state shows the em-dash placeholder (StatCard.isLoading)
    expect(screen.getAllByText("—").length).toBeGreaterThan(0)
  })
})

describe("EnterpriseDashboard", () => {
  it("renders the Enterprise stat strip and never the visibility card", () => {
    render(
      wrap(
        <EnterpriseDashboard
          applicationsStats={{
            organization_id: "x",
            period_days: 7,
            total_count: 12,
            series: [
              { date: "2026-05-01T00:00:00Z", count: 4 },
              { date: "2026-05-02T00:00:00Z", count: 8 },
            ],
          }}
          isApplicationsLoading={false}
          activeRecruitments={3}
          pendingProposals={5}
          spendingLabel="2 400 €"
          actions={[]}
          actionsLoading={false}
        />,
      ),
    )
    // "Recrutements actifs" appears as both stat tile label AND section
    // header — one occurrence each is the expected layout.
    expect(screen.getAllByText(/Recrutements actifs/i).length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText(/Candidatures reçues/i)).toBeInTheDocument()
    expect(screen.getByText("12")).toBeInTheDocument()
    expect(screen.getAllByText("3").length).toBeGreaterThan(0)
    expect(screen.getByText("5")).toBeInTheDocument()
    expect(screen.queryByText(/Vues du profil/i)).not.toBeInTheDocument()
  })

  it("uses the empty state when there are no recruitments", () => {
    render(
      wrap(
        <EnterpriseDashboard
          applicationsStats={undefined}
          isApplicationsLoading={false}
          activeRecruitments={0}
          pendingProposals={0}
          spendingLabel="—"
          actions={[]}
          actionsLoading={false}
        />,
      ),
    )
    expect(screen.getByText(/Publie une mission/i)).toBeInTheDocument()
  })
})

describe("ReferrerDashboard", () => {
  it("renders the four referrer KPIs", () => {
    render(
      wrap(
        <ReferrerDashboard
          activeReferrals={2}
          pendingCommissionsLabel="450 €"
          paid30dLabel="1 200 €"
          lifetimeTotalLabel="8 600 €"
          actions={[]}
          actionsLoading={false}
        />,
      ),
    )
    // "Mises en relation actives" duplicates between the stat tile and
    // the section header — both are expected.
    expect(screen.getAllByText(/Mises en relation actives/i).length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText(/Commissions en attente/i)).toBeInTheDocument()
    expect(screen.getByText(/Total cumul/i)).toBeInTheDocument()
    expect(screen.getByText("450 €")).toBeInTheDocument()
    expect(screen.getByText("1 200 €")).toBeInTheDocument()
    expect(screen.getByText("8 600 €")).toBeInTheDocument()
  })
})
