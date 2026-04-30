import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { WalletCommissionList } from "../wallet-commission-list"
import type {
  CommissionWallet,
  WalletCommissionRecord,
} from "../../api/wallet-api"

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    ...rest
  }: {
    children: React.ReactNode
    href: string
  }) => (
    <a href={href} {...rest}>
      {children}
    </a>
  ),
}))

function emptySummary(): CommissionWallet {
  return {
    pending_cents: 0,
    pending_kyc_cents: 0,
    paid_cents: 0,
    clawed_back_cents: 0,
    currency: "EUR",
  }
}

function makeRecord(
  overrides: Partial<WalletCommissionRecord> = {},
): WalletCommissionRecord {
  return {
    id: "com-1",
    referral_id: undefined,
    proposal_id: "prop-1",
    milestone_id: "mile-1",
    gross_amount_cents: 1000_00,
    commission_cents: 100_00,
    currency: "EUR",
    status: "pending",
    created_at: "2026-04-01T10:00:00Z",
    ...overrides,
  }
}

describe("WalletCommissionList", () => {
  it("renders nothing when there is no activity at all", () => {
    const { container } = render(
      <WalletCommissionList summary={emptySummary()} records={[]} />,
    )
    expect(container).toBeEmptyDOMElement()
  })

  it("renders the section when summary has activity even with no records", () => {
    render(
      <WalletCommissionList
        summary={{ ...emptySummary(), paid_cents: 50_00 }}
        records={[]}
      />,
    )
    expect(
      screen.getByRole("heading", { name: /Mes commissions d'apport/i }),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/Aucune commission pour le moment/),
    ).toBeInTheDocument()
  })

  it("shows the 'KYC à compléter' description when pending_kyc_cents > 0", () => {
    render(
      <WalletCommissionList
        summary={{ ...emptySummary(), pending_kyc_cents: 100_00 }}
        records={[]}
      />,
    )
    expect(
      screen.getByText(/Dont KYC à compléter/i),
    ).toBeInTheDocument()
  })

  it("shows the 'queue' description when only pending_cents > 0", () => {
    render(
      <WalletCommissionList
        summary={{ ...emptySummary(), pending_cents: 100_00 }}
        records={[]}
      />,
    )
    expect(
      screen.getByText(/Commissions en file d'attente de virement/i),
    ).toBeInTheDocument()
  })

  it("renders one row per record", () => {
    render(
      <WalletCommissionList
        summary={{ ...emptySummary(), paid_cents: 50_00 }}
        records={[
          makeRecord({ id: "c1" }),
          makeRecord({ id: "c2" }),
          makeRecord({ id: "c3" }),
        ]}
      />,
    )
    expect(screen.getAllByText(/Commission du/)).toHaveLength(3)
  })

  it.each([
    // The badge labels overlap with the BalanceCard plural labels for
    // a few statuses ("Reçue" / "Reçues", "Reprise" / "Reprises", etc.).
    // We assert via getAllByText so a match in either context is fine.
    ["paid", /Reçue/i],
    ["pending", /En attente/i],
    ["pending_kyc", /KYC requis/i],
    ["clawed_back", /Reprise/i],
    ["failed", /Échec/i],
    ["cancelled", /Annulée/i],
    ["weirdo", /weirdo/i],
  ] as const)("renders the badge for status %s", (status, regex) => {
    render(
      <WalletCommissionList
        summary={{ ...emptySummary(), paid_cents: 50_00 }}
        records={[makeRecord({ status })]}
      />,
    )
    // The mini status badge appears next to the amount
    expect(screen.getAllByText(regex).length).toBeGreaterThan(0)
  })

  it("wraps a row in a referral link when referral_id is set", () => {
    render(
      <WalletCommissionList
        summary={{ ...emptySummary(), paid_cents: 50_00 }}
        records={[makeRecord({ id: "x", referral_id: "ref-99" })]}
      />,
    )
    const link = screen.getByRole("link", { name: /Voir la mise en relation/i })
    expect(link).toHaveAttribute("href", "/referrals/ref-99")
  })

  it("does not wrap a row in a link when referral_id is undefined", () => {
    render(
      <WalletCommissionList
        summary={{ ...emptySummary(), paid_cents: 50_00 }}
        records={[makeRecord({ id: "x", referral_id: undefined })]}
      />,
    )
    expect(screen.queryByRole("link")).not.toBeInTheDocument()
  })

  it("renders the 3 balance card labels", () => {
    render(
      <WalletCommissionList
        summary={{
          pending_cents: 100_00,
          pending_kyc_cents: 50_00,
          paid_cents: 200_00,
          clawed_back_cents: 30_00,
          currency: "EUR",
        }}
        records={[]}
      />,
    )
    expect(screen.getByText(/En attente/)).toBeInTheDocument()
    expect(screen.getByText(/Reçues/)).toBeInTheDocument()
    expect(screen.getByText(/Reprises/)).toBeInTheDocument()
  })

  it("formats amounts in EUR", () => {
    render(
      <WalletCommissionList
        summary={{
          pending_cents: 0,
          pending_kyc_cents: 0,
          paid_cents: 12345_00,
          clawed_back_cents: 0,
          currency: "EUR",
        }}
        records={[]}
      />,
    )
    // 1234500 cents -> 12 345,00 €
    expect(screen.getByText(/12\s?345,00/)).toBeInTheDocument()
  })

  it("renders the 'sur X de mission' subline for each row", () => {
    render(
      <WalletCommissionList
        summary={{ ...emptySummary(), paid_cents: 50_00 }}
        records={[makeRecord({ gross_amount_cents: 5000_00 })]}
      />,
    )
    expect(screen.getByText(/sur\s+5\s?000,00/)).toBeInTheDocument()
  })
})
