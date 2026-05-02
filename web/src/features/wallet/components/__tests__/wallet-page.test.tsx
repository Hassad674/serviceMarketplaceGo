import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen } from "@testing-library/react"
import { WalletPage } from "../wallet-page"
import type { WalletOverview } from "../../api/wallet-api"

// Stub the heavy sub-components — this test focuses on the orchestrator
// concerns (loading / error / data flow). Each child has its own dedicated
// test file with full coverage.
vi.mock("../wallet-payout-section", () => ({
  WalletPayoutSection: (props: {
    totalEarned: number
    available: number
    stripeAccountId: string
    payoutsEnabled: boolean
  }) => (
    <div
      data-testid="payout-section"
      data-total={props.totalEarned}
      data-available={props.available}
      data-stripe={props.stripeAccountId}
      data-enabled={String(props.payoutsEnabled)}
    />
  ),
}))

vi.mock("../wallet-transactions-list", () => ({
  WalletTransactionsList: (props: {
    escrow: number
    available: number
    transferred: number
    records: unknown[]
  }) => (
    <div
      data-testid="transactions-list"
      data-escrow={props.escrow}
      data-available={props.available}
      data-transferred={props.transferred}
      data-records={props.records.length}
    />
  ),
  // re-export passthroughs we don't need
  BalanceCard: () => null,
  SectionHeader: () => null,
}))

vi.mock("../wallet-commission-list", () => ({
  WalletCommissionList: (props: {
    summary: { pending_cents: number }
    records: unknown[]
  }) => (
    <div
      data-testid="commission-list"
      data-pending={props.summary.pending_cents}
      data-records={props.records.length}
    />
  ),
}))

// `CurrentMonthAggregate` moved to shared (P9). wallet-page now
// imports the shared path.
vi.mock("@/shared/components/billing-profile/current-month-aggregate", () => ({
  CurrentMonthAggregate: () => <div data-testid="current-month" />,
}))

const mockUseWallet = vi.fn()
vi.mock("../../hooks/use-wallet", () => ({
  useWallet: () => mockUseWallet(),
}))

beforeEach(() => {
  vi.clearAllMocks()
})

function fullWallet(overrides: Partial<WalletOverview> = {}): WalletOverview {
  return {
    stripe_account_id: "acct_x",
    charges_enabled: true,
    payouts_enabled: true,
    escrow_amount: 100_00,
    available_amount: 200_00,
    transferred_amount: 300_00,
    records: [
      {
        id: "r1",
        proposal_id: "p1",
        proposal_amount: 0,
        platform_fee: 0,
        provider_payout: 0,
        payment_status: "succeeded",
        transfer_status: "completed",
        mission_status: "paid",
        created_at: "2026-01-01T00:00:00Z",
      },
    ],
    commissions: {
      pending_cents: 50_00,
      pending_kyc_cents: 0,
      paid_cents: 75_00,
      clawed_back_cents: 0,
      currency: "EUR",
    },
    commission_records: [],
    ...overrides,
  }
}

describe("WalletPage", () => {
  it("renders the skeleton while loading", () => {
    mockUseWallet.mockReturnValue({
      data: undefined,
      isLoading: true,
      isError: false,
    })
    const { container } = render(<WalletPage />)
    // Skeleton uses animate-shimmer
    expect(container.innerHTML).toContain("animate-shimmer")
  })

  it("renders the error state when isError", () => {
    mockUseWallet.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
    })
    render(<WalletPage />)
    expect(
      screen.getByText(/Impossible de charger le wallet/i),
    ).toBeInTheDocument()
  })

  it("renders the error state when data is undefined", () => {
    mockUseWallet.mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: false,
    })
    render(<WalletPage />)
    expect(
      screen.getByText(/Impossible de charger le wallet/i),
    ).toBeInTheDocument()
  })

  it("composes all 4 sections when data is loaded", () => {
    mockUseWallet.mockReturnValue({
      data: fullWallet(),
      isLoading: false,
      isError: false,
    })
    render(<WalletPage />)
    expect(screen.getByTestId("payout-section")).toBeInTheDocument()
    expect(screen.getByTestId("current-month")).toBeInTheDocument()
    expect(screen.getByTestId("transactions-list")).toBeInTheDocument()
    expect(screen.getByTestId("commission-list")).toBeInTheDocument()
  })

  it("computes totalEarned = transferred + commissions.paid", () => {
    mockUseWallet.mockReturnValue({
      data: fullWallet({
        transferred_amount: 1000_00,
        commissions: {
          pending_cents: 0,
          pending_kyc_cents: 0,
          paid_cents: 250_00,
          clawed_back_cents: 0,
          currency: "EUR",
        },
      }),
      isLoading: false,
      isError: false,
    })
    render(<WalletPage />)
    expect(screen.getByTestId("payout-section")).toHaveAttribute(
      "data-total",
      String(1000_00 + 250_00),
    )
  })

  it("forwards the wallet's stripe_account_id and payouts_enabled to the payout section", () => {
    mockUseWallet.mockReturnValue({
      data: fullWallet({
        stripe_account_id: "acct_xyz",
        payouts_enabled: false,
      }),
      isLoading: false,
      isError: false,
    })
    render(<WalletPage />)
    const section = screen.getByTestId("payout-section")
    expect(section).toHaveAttribute("data-stripe", "acct_xyz")
    expect(section).toHaveAttribute("data-enabled", "false")
  })

  it("forwards the records list to the transactions list", () => {
    mockUseWallet.mockReturnValue({
      data: fullWallet(),
      isLoading: false,
      isError: false,
    })
    render(<WalletPage />)
    expect(screen.getByTestId("transactions-list")).toHaveAttribute(
      "data-records",
      "1",
    )
  })

  it("treats null records as an empty list", () => {
    mockUseWallet.mockReturnValue({
      data: fullWallet({ records: null, commission_records: null }),
      isLoading: false,
      isError: false,
    })
    render(<WalletPage />)
    expect(screen.getByTestId("transactions-list")).toHaveAttribute(
      "data-records",
      "0",
    )
    expect(screen.getByTestId("commission-list")).toHaveAttribute(
      "data-records",
      "0",
    )
  })

  it("forwards summary + records to the commission list", () => {
    mockUseWallet.mockReturnValue({
      data: fullWallet({
        commissions: {
          pending_cents: 99_00,
          pending_kyc_cents: 0,
          paid_cents: 0,
          clawed_back_cents: 0,
          currency: "EUR",
        },
      }),
      isLoading: false,
      isError: false,
    })
    render(<WalletPage />)
    expect(screen.getByTestId("commission-list")).toHaveAttribute(
      "data-pending",
      "9900",
    )
  })
})
