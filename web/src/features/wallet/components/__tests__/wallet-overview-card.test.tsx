import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { WalletOverviewCard } from "../wallet-overview-card"
import { ApiError } from "@/shared/lib/api-client"

// Mock next/link so it renders as a plain <a> in tests
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

function defaultProps(overrides: Partial<Parameters<typeof WalletOverviewCard>[0]> = {}) {
  return {
    totalEarned: 1_500_00,
    available: 500_00,
    stripeAccountId: "acct_123",
    payoutsEnabled: true,
    canWithdraw: true,
    payoutPending: false,
    payoutMessage: "",
    payoutError: null,
    onPayout: vi.fn(),
    ...overrides,
  }
}

describe("WalletOverviewCard", () => {
  it("renders the heading and the total earned in EUR", () => {
    render(<WalletOverviewCard {...defaultProps()} />)
    expect(screen.getByRole("heading", { name: /Portefeuille/i })).toBeInTheDocument()
    // 150 000 cents -> 1 500,00 €
    expect(screen.getByText(/1\s?500,00/)).toBeInTheDocument()
  })

  it("shows the 'compte prêt' line when payouts are enabled", () => {
    render(<WalletOverviewCard {...defaultProps({ payoutsEnabled: true })} />)
    expect(screen.getByText(/Compte Stripe prêt/i)).toBeInTheDocument()
  })

  it("shows the 'en cours de vérification' line when payouts are disabled but account exists", () => {
    render(
      <WalletOverviewCard
        {...defaultProps({ payoutsEnabled: false, stripeAccountId: "acct_x" })}
      />,
    )
    expect(screen.getByText(/Compte Stripe en cours de vérification/i)).toBeInTheDocument()
    expect(screen.getByText(/Finaliser/i)).toBeInTheDocument()
  })

  it("shows the 'non configuré' line when there is no account", () => {
    render(
      <WalletOverviewCard
        {...defaultProps({ stripeAccountId: "", payoutsEnabled: false })}
      />,
    )
    expect(screen.getByText(/Compte Stripe non configuré/i)).toBeInTheDocument()
    expect(screen.getByText(/Configurer$/i)).toBeInTheDocument()
  })

  it("renders the Retirer button enabled when canWithdraw and available > 0", () => {
    render(<WalletOverviewCard {...defaultProps()} />)
    const btn = screen.getByRole("button", { name: /Retirer/i })
    expect(btn).toBeEnabled()
  })

  it("disables the Retirer button when available is zero", () => {
    render(<WalletOverviewCard {...defaultProps({ available: 0 })} />)
    const btn = screen.getByRole("button", { name: /Retirer/i })
    expect(btn).toBeDisabled()
    expect(screen.getByText(/Aucun fonds disponible/i)).toBeInTheDocument()
  })

  it("disables the Retirer button when canWithdraw is false (member, not owner)", () => {
    render(<WalletOverviewCard {...defaultProps({ canWithdraw: false })} />)
    const btn = screen.getByRole("button", { name: /Retirer/i })
    expect(btn).toBeDisabled()
    expect(btn).toHaveAttribute(
      "title",
      "Seul le propriétaire peut demander un retrait",
    )
  })

  it("disables the Retirer button while pending and shows the loader", () => {
    const onPayout = vi.fn()
    render(
      <WalletOverviewCard
        {...defaultProps({ payoutPending: true, onPayout })}
      />,
    )
    const btn = screen.getByRole("button", { name: /Retirer/i })
    expect(btn).toBeDisabled()
  })

  it("calls onPayout when the Retirer button is clicked", () => {
    const onPayout = vi.fn()
    render(<WalletOverviewCard {...defaultProps({ onPayout })} />)
    fireEvent.click(screen.getByRole("button", { name: /Retirer/i }))
    expect(onPayout).toHaveBeenCalledOnce()
  })

  it("renders the success payoutMessage when provided", () => {
    render(
      <WalletOverviewCard
        {...defaultProps({ payoutMessage: "Virement initié" })}
      />,
    )
    expect(screen.getByText(/Virement initié/)).toBeInTheDocument()
  })

  it("renders the generic error banner when payoutError is a non-ApiError", () => {
    render(
      <WalletOverviewCard
        {...defaultProps({ payoutError: new Error("boom") })}
      />,
    )
    expect(screen.getByText(/Erreur lors du retrait/i)).toBeInTheDocument()
  })

  it("renders the targeted 'configurer' copy when error is stripe_account_missing ApiError", () => {
    const err = new ApiError(
      403,
      "stripe_account_missing",
      "missing",
      undefined,
    )
    render(<WalletOverviewCard {...defaultProps({ payoutError: err })} />)
    expect(
      screen.getByText(/configurer vos informations de paiement/i),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("link", { name: /Configurer maintenant/i }),
    ).toBeInTheDocument()
  })

  it("renders the quick-link row to settings", () => {
    render(<WalletOverviewCard {...defaultProps()} />)
    expect(
      screen.getByRole("link", { name: /Modifier mes infos de facturation/i }),
    ).toHaveAttribute(
      "href",
      "/settings/billing-profile?return_to=/wallet",
    )
    expect(
      screen.getByRole("link", { name: /Mes infos de paiement Stripe/i }),
    ).toHaveAttribute("href", "/payment-info")
  })
})
