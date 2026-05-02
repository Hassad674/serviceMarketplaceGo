import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { ProposalCard } from "../proposal-card"
import type { ProposalMessageMetadata } from "../../types"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, params?: Record<string, string | number>) => {
    if (key === "proposalFrom" && params?.name) {
      return `From ${params.name}`
    }
    if (key === "counterProposal" && params?.version) {
      return `counterProposal`
    }
    return key
  },
}))

// Mock @i18n/navigation
const pushFn = vi.fn()
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: pushFn, back: vi.fn() }),
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  Handshake: (props: Record<string, unknown>) => <span data-testid="handshake-icon" {...props} />,
  CheckCircle2: (props: Record<string, unknown>) => <span data-testid="check-circle-icon" {...props} />,
  XCircle: (props: Record<string, unknown>) => <span data-testid="x-circle-icon" {...props} />,
  Clock: (props: Record<string, unknown>) => <span data-testid="clock-icon" {...props} />,
  Calendar: (props: Record<string, unknown>) => <span data-testid="calendar-icon" {...props} />,
  Paperclip: (props: Record<string, unknown>) => <span data-testid="paperclip-icon" {...props} />,
  CreditCard: (props: Record<string, unknown>) => <span data-testid="credit-card-icon" {...props} />,
  Pencil: (props: Record<string, unknown>) => <span data-testid="pencil-icon" {...props} />,
  Loader2: (props: Record<string, unknown>) => <span data-testid="loader-icon" {...props} />,
  DollarSign: (props: Record<string, unknown>) => <span data-testid="dollar-icon" {...props} />,
  Star: (props: Record<string, unknown>) => <span data-testid="star-icon" {...props} />,
  ExternalLink: (props: Record<string, unknown>) => <span data-testid="external-link-icon" {...props} />,
}))

// Mock proposal hooks
const acceptMutateFn = vi.fn()
const declineMutateFn = vi.fn()
vi.mock("@/shared/hooks/proposal/use-proposal-actions", () => ({
  useAcceptProposal: () => ({
    mutate: acceptMutateFn,
    isPending: false,
  }),
  useDeclineProposal: () => ({
    mutate: declineMutateFn,
    isPending: false,
  }),
}))

// Mock @/shared/lib/utils
vi.mock("@/shared/lib/utils", () => ({
  cn: (...classes: unknown[]) => classes.filter(Boolean).join(" "),
  formatCurrency: (amount: number) => `${amount.toFixed(2)} EUR`,
}))

function createMetadata(overrides: Partial<ProposalMessageMetadata> = {}): ProposalMessageMetadata {
  return {
    proposal_id: "proposal-1",
    proposal_title: "Website redesign",
    proposal_amount: 500000,
    proposal_status: "pending",
    proposal_deadline: null,
    proposal_sender_name: "Alice",
    proposal_documents_count: 0,
    proposal_version: 1,
    proposal_parent_id: null,
    proposal_client_id: "client-1",
    proposal_provider_id: "provider-1",
    ...overrides,
  }
}

describe("ProposalCard", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders proposal title", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_title: "Website redesign" })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.getByText("Website redesign")).toBeDefined()
  })

  it("renders formatted amount", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_amount: 500000 })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    // 500000 centimes = 5000.00 EUR
    expect(screen.getByText("5000.00 EUR")).toBeDefined()
  })

  it("renders status badge", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_status: "pending" })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.getByText("pending")).toBeDefined()
  })

  it("shows Accept and Decline buttons when pending and recipient", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_status: "pending" })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.getByText("accept")).toBeDefined()
    expect(screen.getByText("decline")).toBeDefined()
  })

  it("hides Accept and Decline buttons when not pending", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_status: "accepted" })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.queryByText("accept")).toBeNull()
    expect(screen.queryByText("decline")).toBeNull()
  })

  it("hides Accept and Decline buttons when own proposal (sender)", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_status: "pending" })}
        isOwn={true}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.queryByText("accept")).toBeNull()
    expect(screen.queryByText("decline")).toBeNull()
  })

  it("shows Pay button when accepted and user is client", () => {
    render(
      <ProposalCard
        metadata={createMetadata({
          proposal_status: "accepted",
          proposal_client_id: "user-1",
        })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.getByText("pay")).toBeDefined()
  })

  it("hides Pay button when accepted but user is not client", () => {
    render(
      <ProposalCard
        metadata={createMetadata({
          proposal_status: "accepted",
          proposal_client_id: "someone-else",
        })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.queryByText("pay")).toBeNull()
  })

  it("shows Modify button when pending and own proposal (sender)", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_status: "pending" })}
        isOwn={true}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.getByText("modify")).toBeDefined()
  })

  it("hides Modify button when not pending", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_status: "accepted" })}
        isOwn={true}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.queryByText("modify")).toBeNull()
  })

  it("shows counter-proposal label for version > 1", () => {
    render(
      <ProposalCard
        metadata={createMetadata({
          proposal_status: "pending",
          proposal_version: 2,
          proposal_parent_id: "parent-1",
        })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.getByText("counterProposal")).toBeDefined()
  })

  it("shows proposalFrom label for version 1", () => {
    render(
      <ProposalCard
        metadata={createMetadata({
          proposal_status: "pending",
          proposal_version: 1,
          proposal_parent_id: null,
          proposal_sender_name: "Alice",
        })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.getByText("From Alice")).toBeDefined()
  })

  it("calls accept mutation when Accept clicked", () => {
    render(
      <ProposalCard
        metadata={createMetadata({
          proposal_id: "proposal-42",
          proposal_status: "pending",
        })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    fireEvent.click(screen.getByText("accept"))
    expect(acceptMutateFn).toHaveBeenCalledWith("proposal-42")
  })

  it("calls decline mutation when Decline clicked", () => {
    render(
      <ProposalCard
        metadata={createMetadata({
          proposal_id: "proposal-42",
          proposal_status: "pending",
        })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    fireEvent.click(screen.getByText("decline"))
    expect(declineMutateFn).toHaveBeenCalledWith("proposal-42")
  })

  it("navigates to modify page when Modify clicked", () => {
    render(
      <ProposalCard
        metadata={createMetadata({
          proposal_id: "proposal-42",
          proposal_status: "pending",
          proposal_provider_id: "prov-1",
          proposal_client_id: "client-1",
        })}
        isOwn={true}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    fireEvent.click(screen.getByText("modify"))
    expect(pushFn).toHaveBeenCalledOnce()
    const calledUrl = pushFn.mock.calls[0][0] as string
    expect(calledUrl).toContain("modify=proposal-42")
    expect(calledUrl).toContain("conversation=conv-1")
  })

  it("navigates to pay page when Pay clicked", () => {
    render(
      <ProposalCard
        metadata={createMetadata({
          proposal_id: "proposal-42",
          proposal_status: "accepted",
          proposal_client_id: "user-1",
        })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    fireEvent.click(screen.getByText("pay"))
    expect(pushFn).toHaveBeenCalledWith("/projects/pay?proposal=proposal-42")
  })

  it("renders deadline when provided", () => {
    render(
      <ProposalCard
        metadata={createMetadata({
          proposal_deadline: "2026-06-15T00:00:00Z",
        })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    // The date is formatted with fr-FR Intl, should contain "15"
    expect(screen.getByText(/15/)).toBeDefined()
  })

  it("renders documents count when non-zero", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_documents_count: 3 })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.getByText("3")).toBeDefined()
  })

  it("shows counter-proposal label for version > 1", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_version: 3 })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    // Version > 1 renders counter-proposal label instead of proposalFrom
    expect(screen.getByText("counterProposal")).toBeDefined()
  })

  it("shows proposalFrom label for version 1 (not counter-proposal)", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_version: 1, proposal_sender_name: "Test" })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.getByText("From Test")).toBeDefined()
  })

  it("renders sender name", () => {
    render(
      <ProposalCard
        metadata={createMetadata({ proposal_sender_name: "Bob Jones" })}
        isOwn={false}
        currentUserId="user-1"
        conversationId="conv-1"
      />,
    )

    expect(screen.getByText("From Bob Jones")).toBeDefined()
  })
})
