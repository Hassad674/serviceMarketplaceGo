import { describe, expect, it, vi } from "vitest"
import { render, screen } from "@testing-library/react"

import { ReferralMissionsSection } from "../referral-missions-section"

// useReferralAttributions + useReferralCommissions are the two data
// hooks this component now depends on (Run C added the per-milestone
// projection). We mock both so the component renders against the
// controlled fixtures below — no MSW, no QueryClientProvider needed.
//
// next-intl is mocked separately so the projection list (which uses
// useTranslations) renders without needing the IntlProvider wiring.
vi.mock("next-intl", () => ({
  useTranslations:
    (namespace?: string) =>
    (key: string, params?: Record<string, string | number>) => {
      const full = namespace ? `${namespace}.${key}` : key
      return params ? `${full}(${JSON.stringify(params)})` : full
    },
}))

vi.mock("../../hooks/use-referrals", () => ({
  useReferralAttributions: vi.fn(),
  useReferralCommissions: vi.fn(),
}))

import {
  useReferralAttributions,
  useReferralCommissions,
} from "../../hooks/use-referrals"

function mockAttributions(rows: unknown[]) {
  ;(useReferralAttributions as ReturnType<typeof vi.fn>).mockReturnValue({
    data: rows,
    isLoading: false,
    isError: false,
  })
  // Default: no commission rows, so the projection list falls into
  // the empty-state branch (silent — invisible to existing tests).
  ;(useReferralCommissions as ReturnType<typeof vi.fn>).mockReturnValue({
    data: [],
    isLoading: false,
    isError: false,
  })
}

// makeAttribution returns the minimum shape ReferralMissionsSection
// needs. Fields the component does not read are filled with safe
// defaults so the test stays focused on the milestone counter.
function makeAttribution(over: Partial<Record<string, unknown>> = {}) {
  return {
    id: "att-1",
    proposal_id: "00000000-0000-0000-0000-000000000001",
    proposal_title: "Mission alpha",
    proposal_status: "completed",
    attributed_at: "2026-05-01T10:00:00Z",
    rate_pct_snapshot: 5,
    total_amount_cents: 0,
    total_commission_cents: 0,
    escrow_commission_cents: 0,
    clawed_back_commission_cents: 0,
    milestones_paid: 0,
    milestones_pending: 0,
    milestones_total: 0,
    ...over,
  }
}

describe("ReferralMissionsSection — milestone counter", () => {
  it("renders 2/2 jalons when both milestones are completed (post-backfill regression)", () => {
    mockAttributions([
      makeAttribution({
        proposal_status: "completed",
        milestones_paid: 2,
        milestones_pending: 0,
        milestones_total: 2,
      }),
    ])

    render(
      <ReferralMissionsSection referralId="ref-1" viewerIsClient={false} />,
    )

    expect(screen.getByText("2/2 jalons")).toBeInTheDocument()
  })

  it("renders 1/3 jalons when one of three milestones is completed", () => {
    mockAttributions([
      makeAttribution({
        proposal_status: "active",
        milestones_paid: 1,
        milestones_pending: 2,
        milestones_total: 3,
      }),
    ])

    render(
      <ReferralMissionsSection referralId="ref-1" viewerIsClient={false} />,
    )

    expect(screen.getByText("1/3 jalons")).toBeInTheDocument()
  })

  it("falls back to paid+pending math when milestones_total is missing (legacy API shim)", () => {
    mockAttributions([
      makeAttribution({
        proposal_status: "active",
        milestones_paid: 1,
        milestones_pending: 1,
        // milestones_total intentionally 0 — old API
        milestones_total: 0,
      }),
    ])

    render(
      <ReferralMissionsSection referralId="ref-1" viewerIsClient={false} />,
    )

    expect(screen.getByText("1/2 jalons")).toBeInTheDocument()
  })

  it("renders 0/2 jalons when nothing has been approved yet (in-flight, pre-approval)", () => {
    mockAttributions([
      makeAttribution({
        proposal_status: "active",
        milestones_paid: 0,
        milestones_pending: 2,
        milestones_total: 2,
      }),
    ])

    render(
      <ReferralMissionsSection referralId="ref-1" viewerIsClient={false} />,
    )

    expect(screen.getByText("0/2 jalons")).toBeInTheDocument()
  })

  it("exposes the progressbar with the correct aria value", () => {
    mockAttributions([
      makeAttribution({
        proposal_status: "active",
        milestones_paid: 1,
        milestones_pending: 1,
        milestones_total: 2,
      }),
    ])

    render(
      <ReferralMissionsSection referralId="ref-1" viewerIsClient={false} />,
    )

    const bar = screen.getByRole("progressbar")
    expect(bar.getAttribute("aria-valuenow")).toBe("50")
    expect(bar.getAttribute("aria-valuemin")).toBe("0")
    expect(bar.getAttribute("aria-valuemax")).toBe("100")
  })

  it("hides the section entirely when there are no attributions", () => {
    mockAttributions([])

    const { container } = render(
      <ReferralMissionsSection referralId="ref-1" viewerIsClient={false} />,
    )

    expect(container.firstChild).toBeNull()
  })
})

describe("ReferralMissionsSection — mission total amount", () => {
  it("renders the total amount pill alongside the title when > 0", () => {
    mockAttributions([
      makeAttribution({
        proposal_title: "Refonte LP",
        total_amount_cents: 123_000,
      }),
    ])
    render(
      <ReferralMissionsSection referralId="ref-1" viewerIsClient={false} />,
    )
    const pill = screen.getByTestId("attribution-total-amount")
    expect(pill).toBeInTheDocument()
    // Intl French formatting: NBSP + €. Assert digits + currency are present.
    expect(pill.textContent ?? "").toContain("1")
    expect(pill.textContent ?? "").toContain("€")
  })

  it("omits the total amount pill when the value is 0 (proposal lookup failed)", () => {
    mockAttributions([
      makeAttribution({ proposal_title: "Mission inconnue", total_amount_cents: 0 }),
    ])
    render(
      <ReferralMissionsSection referralId="ref-1" viewerIsClient={false} />,
    )
    expect(
      screen.queryByTestId("attribution-total-amount"),
    ).not.toBeInTheDocument()
  })

  it("renders the total amount pill ALSO for the client viewer (public mission price, not commission)", () => {
    mockAttributions([
      makeAttribution({ total_amount_cents: 50_000 }),
    ])
    render(
      <ReferralMissionsSection referralId="ref-1" viewerIsClient />,
    )
    expect(
      screen.getByTestId("attribution-total-amount"),
    ).toBeInTheDocument()
  })
})
