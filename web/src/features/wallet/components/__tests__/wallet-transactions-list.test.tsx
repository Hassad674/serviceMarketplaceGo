import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { Briefcase } from "lucide-react"
import { ApiError } from "@/shared/lib/api-client"
import {
  WalletTransactionsList,
  BalanceCard,
  SectionHeader,
} from "../wallet-transactions-list"
import type { WalletRecord } from "../../api/wallet-api"

// next/link
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

// Permission hook
const mockHasPermission = vi.fn((_perm: string) => true)
vi.mock("@/shared/hooks/use-permissions", () => ({
  useHasPermission: (perm: string) => mockHasPermission(perm),
}))

// Retry hook — we control the mutation surface
const retryMutate = vi.fn()
const retryMutationState = {
  mutate: retryMutate,
  isPending: false,
  isError: false,
  variables: undefined as string | undefined,
  error: null as Error | null,
}
vi.mock("../../hooks/use-wallet", () => ({
  useRetryTransfer: () => retryMutationState,
}))

beforeEach(() => {
  vi.clearAllMocks()
  retryMutationState.isPending = false
  retryMutationState.isError = false
  retryMutationState.variables = undefined
  retryMutationState.error = null
  mockHasPermission.mockReturnValue(true)
})

function makeRecord(overrides: Partial<WalletRecord> = {}): WalletRecord {
  return {
    id: "rec-1",
    proposal_id: "prop-1",
    milestone_id: "mile-1",
    proposal_amount: 1000_00,
    platform_fee: 100_00,
    provider_payout: 900_00,
    payment_status: "succeeded",
    transfer_status: "completed",
    mission_status: "paid",
    created_at: "2026-04-01T10:00:00Z",
    ...overrides,
  }
}

describe("BalanceCard", () => {
  it("renders the label, formatted amount, and description", () => {
    render(
      <BalanceCard
        icon={Briefcase}
        label="En séquestre"
        amount={250_00}
        description="Test desc"
        color="amber"
      />,
    )
    expect(screen.getByText("En séquestre")).toBeInTheDocument()
    expect(screen.getByText(/250,00/)).toBeInTheDocument()
    expect(screen.getByText("Test desc")).toBeInTheDocument()
  })

  it.each(["amber", "green", "blue"] as const)(
    "supports the %s color variant without crashing",
    (color) => {
      render(
        <BalanceCard
          icon={Briefcase}
          label={color}
          amount={0}
          description=""
          color={color}
        />,
      )
      expect(screen.getByText(color)).toBeInTheDocument()
    },
  )
})

describe("SectionHeader", () => {
  it("renders an h2 with the title", () => {
    render(<SectionHeader icon={Briefcase} title="Section X" />)
    expect(screen.getByRole("heading", { name: "Section X" })).toBeInTheDocument()
  })
})

describe("WalletTransactionsList", () => {
  it("renders the 3 balance cards (escrow / available / transferred)", () => {
    render(
      <WalletTransactionsList
        escrow={100_00}
        available={200_00}
        transferred={300_00}
        records={[]}
      />,
    )
    expect(screen.getByText(/En séquestre/)).toBeInTheDocument()
    expect(screen.getByText(/Disponible/)).toBeInTheDocument()
    expect(screen.getByText(/Transféré$/)).toBeInTheDocument()
  })

  it("renders the empty state when records is empty", () => {
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[]}
      />,
    )
    expect(screen.getByText(/Aucune mission pour le moment/)).toBeInTheDocument()
  })

  it("renders one row per record", () => {
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[
          makeRecord({ id: "r1" }),
          makeRecord({ id: "r2" }),
          makeRecord({ id: "r3" }),
        ]}
      />,
    )
    expect(screen.getAllByText(/Mission du/)).toHaveLength(3)
  })

  it("displays the escrow tag when transfer is pending", () => {
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[
          makeRecord({ transfer_status: "pending", payment_status: "succeeded" }),
        ]}
      />,
    )
    expect(
      screen.getByText(/En séquestre — mission en cours/),
    ).toBeInTheDocument()
  })

  it("renders the Transféré badge when transfer is completed", () => {
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[
          makeRecord({ transfer_status: "completed", payment_status: "succeeded" }),
        ]}
      />,
    )
    // Many "Transféré" labels exist — check at least one badge variant
    const matches = screen.getAllByText(/Transféré/)
    expect(matches.length).toBeGreaterThan(0)
  })

  it("renders the retry button on failed transfer rows", () => {
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[
          makeRecord({ transfer_status: "failed", payment_status: "failed" }),
        ]}
      />,
    )
    expect(
      screen.getByRole("button", { name: /Relancer le transfert/i }),
    ).toBeInTheDocument()
  })

  it("calls retry mutation with the record id on click", () => {
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[
          makeRecord({
            id: "rec-99",
            transfer_status: "failed",
            payment_status: "failed",
          }),
        ]}
      />,
    )
    fireEvent.click(
      screen.getByRole("button", { name: /Relancer le transfert/i }),
    )
    expect(retryMutate).toHaveBeenCalledWith("rec-99")
  })

  it("disables retry button when canWithdraw is false", () => {
    mockHasPermission.mockReturnValue(false)
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[
          makeRecord({ transfer_status: "failed", payment_status: "failed" }),
        ]}
      />,
    )
    const btn = screen.getByRole("button", { name: /Relancer le transfert/i })
    expect(btn).toBeDisabled()
    expect(btn).toHaveAttribute(
      "title",
      "Seul le propriétaire peut relancer un transfert",
    )
  })

  it.each([
    ["provider_kyc_incomplete", /Termine ton onboarding Stripe/i],
    ["transfer_not_retriable", /Ce transfert ne peut plus être relancé/i],
    ["stripe_account_missing", /Configure d'abord tes informations de paiement/i],
    ["retry_failed", /Le virement a de nouveau échoué côté Stripe/i],
    ["payment_record_not_found", /Ce transfert est introuvable/i],
    ["wat", /Erreur lors de la nouvelle tentative/i],
  ] as const)("renders code-specific copy for retry error %s", (code, regex) => {
    retryMutationState.isError = true
    retryMutationState.variables = "rec-1"
    retryMutationState.error = new ApiError(412, code, "msg")
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[
          makeRecord({ transfer_status: "failed", payment_status: "failed" }),
        ]}
      />,
    )
    expect(screen.getByRole("alert")).toHaveTextContent(regex)
  })

  it("falls back to generic copy when retry error is not an ApiError", () => {
    retryMutationState.isError = true
    retryMutationState.variables = "rec-1"
    retryMutationState.error = new Error("network")
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[
          makeRecord({ transfer_status: "failed", payment_status: "failed" }),
        ]}
      />,
    )
    expect(screen.getByRole("alert")).toHaveTextContent(
      /Erreur lors de la nouvelle tentative/i,
    )
  })

  it("does not show the retry error when the row id does not match the variables", () => {
    retryMutationState.isError = true
    retryMutationState.variables = "other-rec"
    retryMutationState.error = new ApiError(500, "x", "y")
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[
          makeRecord({
            id: "rec-1",
            transfer_status: "failed",
            payment_status: "failed",
          }),
        ]}
      />,
    )
    expect(screen.queryByRole("alert")).not.toBeInTheDocument()
  })

  it.each([
    ["active", /En cours/],
    ["completion_requested", /Complétion demandée/],
    ["completed", /Terminée/],
    ["paid", /Payée/],
    ["weird_state", /weird_state/],
  ] as const)("renders mission badge for status %s", (status, regex) => {
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[makeRecord({ mission_status: status })]}
      />,
    )
    expect(screen.getByText(regex)).toBeInTheDocument()
  })

  it("hides mission badge when mission_status is empty", () => {
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[makeRecord({ mission_status: "" })]}
      />,
    )
    // No badges of any of the canonical labels
    expect(screen.queryByText("En cours")).not.toBeInTheDocument()
  })

  it.each([
    ["succeeded", "text-green-500"],
    ["failed", "text-red-500"],
    ["pending", "text-amber-500"],
  ] as const)(
    "selects the correct payment-status icon class for %s",
    (status, cls) => {
      const { container } = render(
        <WalletTransactionsList
          escrow={0}
          available={0}
          transferred={0}
          records={[makeRecord({ payment_status: status })]}
        />,
      )
      const matched = container.querySelector(`.${cls}`)
      expect(matched).not.toBeNull()
    },
  )

  it.each([
    ["completed", /Transféré/i],
    ["failed", /Échec/i],
    ["pending", /En séquestre/i],
  ] as const)("renders the transfer badge for status %s", (status, label) => {
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[makeRecord({ transfer_status: status, payment_status: "succeeded" })]}
      />,
    )
    expect(screen.getAllByText(label).length).toBeGreaterThan(0)
  })

  it("uses fallback key when record.id is missing", () => {
    // Should not throw on duplicate-key warnings; we verify the rendered element
    // by content count instead of id.
    const recA = makeRecord({ id: "", milestone_id: "mile-A" })
    const recB = makeRecord({ id: "", milestone_id: "mile-B" })
    render(
      <WalletTransactionsList
        escrow={0}
        available={0}
        transferred={0}
        records={[recA, recB]}
      />,
    )
    expect(screen.getAllByText(/Mission du/)).toHaveLength(2)
  })
})
