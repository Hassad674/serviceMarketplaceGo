import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { FeePreview } from "../fee-preview"

const mockUseFeePreview = vi.fn()

vi.mock("../../hooks/use-fee-preview", () => ({
  useFeePreview: (...args: unknown[]) => mockUseFeePreview(...args),
}))

const TIERS = [
  { label: "0 - 200", max_cents: 20000, fee_cents: 200 },
  { label: "200 - 1000", max_cents: 100000, fee_cents: 500 },
  { label: "Plus de 1 000 €", max_cents: null, fee_cents: 1500 },
]

function activeData(overrides: Record<string, unknown> = {}) {
  return {
    isLoading: false,
    isError: false,
    data: {
      amount_cents: 30000,
      fee_cents: 500,
      net_cents: 29500,
      role: "freelance",
      active_tier_index: 1,
      tiers: TIERS,
      viewer_is_provider: true,
      viewer_is_subscribed: false,
      ...overrides,
    },
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("FeePreview — gating", () => {
  it("renders nothing when viewer is NOT provider (server fail-closed)", () => {
    mockUseFeePreview.mockReturnValue(
      activeData({ viewer_is_provider: false }),
    )
    const { container } = render(
      <FeePreview
        milestones={[{ key: "1", label: "M1", amountCents: 10000 }]}
        mode="one_time"
      />,
    )
    expect(container.firstChild).toBeNull()
  })

  it("renders the section when viewer IS provider", () => {
    mockUseFeePreview.mockReturnValue(activeData())
    render(
      <FeePreview
        milestones={[{ key: "1", label: "M1", amountCents: 10000 }]}
        mode="one_time"
      />,
    )
    expect(screen.getByRole("heading")).toBeInTheDocument()
  })
})

describe("FeePreview — header", () => {
  it("uses default heading when none supplied", () => {
    mockUseFeePreview.mockReturnValue(activeData())
    render(
      <FeePreview milestones={[]} mode="one_time" />,
    )
    expect(screen.getByText("Frais plateforme")).toBeInTheDocument()
  })

  it("respects custom heading prop", () => {
    mockUseFeePreview.mockReturnValue(activeData())
    render(
      <FeePreview milestones={[]} mode="one_time" heading="Custom title" />,
    )
    expect(screen.getByText("Custom title")).toBeInTheDocument()
  })

  it("shows 'Premium actif' subtitle when subscribed", () => {
    mockUseFeePreview.mockReturnValue(
      activeData({ viewer_is_subscribed: true }),
    )
    render(<FeePreview milestones={[]} mode="one_time" />)
    expect(screen.getByText("Premium actif")).toBeInTheDocument()
  })
})

describe("FeePreview — body states", () => {
  it("shows skeleton on loading", () => {
    mockUseFeePreview.mockReturnValue({ isLoading: true, isError: false })
    const { container } = render(
      <FeePreview milestones={[]} mode="one_time" />,
    )
    const skeletons = container.querySelectorAll(".animate-shimmer")
    expect(skeletons.length).toBeGreaterThan(0)
  })

  it("shows error message on error", () => {
    mockUseFeePreview.mockReturnValue({ isLoading: false, isError: true })
    render(<FeePreview milestones={[]} mode="one_time" />)
    expect(
      screen.getByRole("alert"),
    ).toHaveTextContent(/Impossible/)
  })

  it("shows empty hint when no data and no amount", () => {
    mockUseFeePreview.mockReturnValue(
      activeData({ amount_cents: 0 }),
    )
    render(
      <FeePreview milestones={[]} mode="one_time" />,
    )
    expect(
      screen.getByText(/Renseignez un montant/),
    ).toBeInTheDocument()
  })

  it("shows premium notice when subscribed instead of tariff grid", () => {
    mockUseFeePreview.mockReturnValue(
      activeData({ viewer_is_subscribed: true }),
    )
    render(
      <FeePreview
        milestones={[{ key: "1", label: "M1", amountCents: 10000 }]}
        mode="one_time"
      />,
    )
    expect(
      screen.getByText(/Grâce à votre abonnement/),
    ).toBeInTheDocument()
  })
})

describe("FeePreview — one-time mode", () => {
  it("renders the OneTimeSummary line with net + fee amounts", () => {
    mockUseFeePreview.mockReturnValue(activeData())
    render(
      <FeePreview
        milestones={[{ key: "1", label: "M1", amountCents: 30000 }]}
        mode="one_time"
      />,
    )
    expect(screen.getByText(/Tu encaisses/)).toBeInTheDocument()
  })

  it("highlights the active tier", () => {
    mockUseFeePreview.mockReturnValue(activeData({ active_tier_index: 1 }))
    render(
      <FeePreview
        milestones={[{ key: "1", label: "M1", amountCents: 30000 }]}
        mode="one_time"
      />,
    )
    const items = screen.getAllByRole("listitem")
    expect(items[1].getAttribute("aria-current")).toBe("true")
    expect(items[0].hasAttribute("aria-current")).toBe(false)
  })
})

describe("FeePreview — milestone mode", () => {
  it("renders one row per milestone with its computed fee", () => {
    mockUseFeePreview.mockReturnValue(activeData())
    render(
      <FeePreview
        milestones={[
          { key: "1", label: "Phase A", amountCents: 10000 },
          { key: "2", label: "Phase B", amountCents: 50000 },
        ]}
        mode="milestone"
      />,
    )
    expect(screen.getByText("Phase A")).toBeInTheDocument()
    expect(screen.getByText("Phase B")).toBeInTheDocument()
  })

  it("sums fees correctly across milestones in different tiers", () => {
    mockUseFeePreview.mockReturnValue(activeData())
    render(
      <FeePreview
        milestones={[
          { key: "1", label: "Phase A", amountCents: 10000 }, // tier 0 → 200
          { key: "2", label: "Phase B", amountCents: 50000 }, // tier 1 → 500
          { key: "3", label: "Phase C", amountCents: 200000 }, // tier 2 → 1500
        ]}
        mode="milestone"
      />,
    )
    expect(screen.getByText(/Total frais plateforme/)).toBeInTheDocument()
  })

  it("renders dash for milestones with zero amount", () => {
    mockUseFeePreview.mockReturnValue(activeData({ amount_cents: 0 }))
    const { container } = render(
      <FeePreview
        milestones={[{ key: "1", label: "Empty", amountCents: 0 }]}
        mode="milestone"
      />,
    )
    expect(container.textContent).toContain("Renseignez")
  })

  it("shows hint when milestone list is empty", () => {
    mockUseFeePreview.mockReturnValue(activeData({ amount_cents: 0 }))
    render(<FeePreview milestones={[]} mode="milestone" />)
    expect(
      screen.getByText(/Renseignez un montant/),
    ).toBeInTheDocument()
  })
})

describe("FeePreview — Premium CTA", () => {
  it("renders the renderPremiumCta slot when not subscribed", () => {
    mockUseFeePreview.mockReturnValue(activeData())
    render(
      <FeePreview
        milestones={[{ key: "1", label: "M1", amountCents: 30000 }]}
        mode="one_time"
        renderPremiumCta={<button data-testid="upgrade-btn">Upgrade</button>}
      />,
    )
    expect(screen.getByTestId("upgrade-btn")).toBeInTheDocument()
  })

  it("does NOT render the CTA when subscribed", () => {
    mockUseFeePreview.mockReturnValue(
      activeData({ viewer_is_subscribed: true }),
    )
    render(
      <FeePreview
        milestones={[{ key: "1", label: "M1", amountCents: 30000 }]}
        mode="one_time"
        renderPremiumCta={<button data-testid="upgrade-btn">Upgrade</button>}
      />,
    )
    expect(screen.queryByTestId("upgrade-btn")).not.toBeInTheDocument()
  })
})
