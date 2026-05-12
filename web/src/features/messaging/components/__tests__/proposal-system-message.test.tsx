import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import {
  ProposalSystemMessage,
  PaymentRequestedMessage,
  CompletionRequestedMessage,
  EvaluationRequestMessage,
} from "../proposal-system-message"
import type { ProposalMessageMetadata } from "../../types"

// next-intl mock — key + interpolated params returned verbatim.
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

const pushFn = vi.fn()
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: pushFn, back: vi.fn() }),
}))

// Reduce icon noise; expose only the data attributes we need.
vi.mock("lucide-react", () => {
  const Stub =
    (name: string) =>
    (props: Record<string, unknown>) =>
      <span data-testid={`icon-${name}`} {...props} />
  return {
    AlertTriangle: Stub("alert"),
    CheckCircle2: Stub("check"),
    XCircle: Stub("x"),
    DollarSign: Stub("dollar"),
    CreditCard: Stub("credit"),
    Clock: Stub("clock"),
    RotateCcw: Stub("rotate"),
    Pencil: Stub("pencil"),
    Scale: Stub("scale"),
    ShieldAlert: Stub("shield"),
    Star: Stub("star"),
    Trophy: Stub("trophy"),
    ArrowRight: Stub("arrow"),
  }
})

vi.mock("@/shared/lib/utils", () => ({
  cn: (...classes: unknown[]) => classes.filter(Boolean).join(" "),
  formatCurrency: (amount: number) => `${amount.toFixed(2)} EUR`,
}))

function makeMeta(
  overrides: Partial<ProposalMessageMetadata> = {},
): ProposalMessageMetadata {
  return {
    proposal_id: "prop-1",
    proposal_title: "Website redesign",
    proposal_amount: 500_000,
    proposal_status: "accepted",
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

describe("ProposalSystemMessage — 'Payer maintenant' CTA regression", () => {
  beforeEach(() => {
    pushFn.mockReset()
  })

  it("renders the CTA when type=accepted, viewer is client, status=accepted", () => {
    render(
      <ProposalSystemMessage
        type="proposal_accepted"
        metadata={makeMeta({
          proposal_client_id: "client-1",
          proposal_status: "accepted",
        })}
        currentUserId="client-1"
      />,
    )
    expect(
      screen.getByTestId("proposal-accepted-pay-cta"),
    ).toBeInTheDocument()
    expect(screen.getByText("payNow")).toBeInTheDocument()
  })

  it("clicking the CTA pushes /projects/pay?proposal=<id>", () => {
    render(
      <ProposalSystemMessage
        type="proposal_accepted"
        metadata={makeMeta({
          proposal_id: "prop-42",
          proposal_client_id: "client-1",
          proposal_status: "accepted",
        })}
        currentUserId="client-1"
      />,
    )
    fireEvent.click(screen.getByTestId("proposal-accepted-pay-cta"))
    expect(pushFn).toHaveBeenCalledTimes(1)
    expect(pushFn).toHaveBeenCalledWith("/projects/pay?proposal=prop-42")
  })

  it("hides the CTA when the viewer is the provider (not the client)", () => {
    render(
      <ProposalSystemMessage
        type="proposal_accepted"
        metadata={makeMeta({
          proposal_client_id: "client-1",
          proposal_provider_id: "provider-1",
          proposal_status: "accepted",
        })}
        currentUserId="provider-1"
      />,
    )
    expect(
      screen.queryByTestId("proposal-accepted-pay-cta"),
    ).not.toBeInTheDocument()
  })

  it("hides the CTA when currentUserId is missing", () => {
    render(
      <ProposalSystemMessage
        type="proposal_accepted"
        metadata={makeMeta({
          proposal_client_id: "client-1",
          proposal_status: "accepted",
        })}
      />,
    )
    expect(
      screen.queryByTestId("proposal-accepted-pay-cta"),
    ).not.toBeInTheDocument()
  })

  it.each([
    "paid",
    "completed",
    "active",
    "completion_requested",
    "withdrawn",
    "declined",
    "pending",
  ] as const)(
    "hides the CTA when status is %s (no longer 'accepted')",
    (status) => {
      render(
        <ProposalSystemMessage
          type="proposal_accepted"
          metadata={makeMeta({
            proposal_client_id: "client-1",
            proposal_status: status,
          })}
          currentUserId="client-1"
        />,
      )
      expect(
        screen.queryByTestId("proposal-accepted-pay-cta"),
      ).not.toBeInTheDocument()
    },
  )

  it.each([
    "proposal_declined",
    "proposal_paid",
    "proposal_completed",
    "proposal_modified",
    "proposal_cancelled",
    "proposal_auto_closed",
  ])(
    "hides the CTA on other system message types (%s) even when viewer is client",
    (type) => {
      render(
        <ProposalSystemMessage
          type={type}
          metadata={makeMeta({
            proposal_client_id: "client-1",
            proposal_status: "accepted",
          })}
          currentUserId="client-1"
        />,
      )
      expect(
        screen.queryByTestId("proposal-accepted-pay-cta"),
      ).not.toBeInTheDocument()
    },
  )

  it("renders the title + subtitle for the accepted bubble", () => {
    render(
      <ProposalSystemMessage
        type="proposal_accepted"
        metadata={makeMeta({
          proposal_title: "Refonte vitrine",
          proposal_amount: 250_000,
          proposal_client_id: "client-1",
        })}
        currentUserId="client-1"
      />,
    )
    expect(screen.getByText("systemAccepted")).toBeInTheDocument()
    expect(
      screen.getByText("Refonte vitrine — 2500.00 EUR"),
    ).toBeInTheDocument()
  })

  it("returns null for unknown system message types", () => {
    const { container } = render(
      <ProposalSystemMessage
        type="totally_unknown_type"
        metadata={makeMeta()}
        currentUserId="client-1"
      />,
    )
    expect(container.firstChild).toBeNull()
  })
})

describe("PaymentRequestedMessage (regression guard)", () => {
  beforeEach(() => {
    pushFn.mockReset()
  })

  it("renders Pay CTA for the client and navigates to /projects/pay", () => {
    render(
      <PaymentRequestedMessage
        metadata={makeMeta({
          proposal_id: "p-9",
          proposal_client_id: "client-1",
        })}
        currentUserId="client-1"
      />,
    )
    const btn = screen.getByText("payNow")
    fireEvent.click(btn)
    expect(pushFn).toHaveBeenCalledWith("/projects/pay?proposal=p-9")
  })

  it("hides Pay CTA when the viewer is not the client", () => {
    render(
      <PaymentRequestedMessage
        metadata={makeMeta({
          proposal_client_id: "client-1",
        })}
        currentUserId="provider-1"
      />,
    )
    expect(screen.queryByText("payNow")).not.toBeInTheDocument()
  })
})

describe("CompletionRequestedMessage (regression guard)", () => {
  beforeEach(() => {
    pushFn.mockReset()
  })

  it("shows the view-details CTA only to the client", () => {
    render(
      <CompletionRequestedMessage
        metadata={makeMeta({
          proposal_id: "p-3",
          proposal_client_id: "client-1",
        })}
        currentUserId="client-1"
      />,
    )
    fireEvent.click(screen.getByText("viewDetails"))
    expect(pushFn).toHaveBeenCalledWith("/projects/p-3")
  })

  it("hides the view-details CTA for non-clients", () => {
    render(
      <CompletionRequestedMessage
        metadata={makeMeta({
          proposal_client_id: "client-1",
        })}
        currentUserId="provider-1"
      />,
    )
    expect(screen.queryByText("viewDetails")).not.toBeInTheDocument()
  })
})

describe("EvaluationRequestMessage (regression guard)", () => {
  it("does not render the review CTA when org ids are missing", () => {
    render(
      <EvaluationRequestMessage
        metadata={makeMeta()}
        onReview={vi.fn()}
      />,
    )
    expect(screen.queryByText("leaveReview")).not.toBeInTheDocument()
  })

  it("renders + invokes onReview when org ids are present", () => {
    const onReview = vi.fn()
    render(
      <EvaluationRequestMessage
        metadata={makeMeta({
          proposal_id: "p-7",
          proposal_title: "Refonte",
          proposal_client_organization_id: "co-1",
          proposal_provider_organization_id: "po-1",
        })}
        onReview={onReview}
      />,
    )
    fireEvent.click(screen.getByText("leaveReview"))
    expect(onReview).toHaveBeenCalledWith("p-7", "Refonte", {
      clientOrganizationId: "co-1",
      providerOrganizationId: "po-1",
    })
  })
})
