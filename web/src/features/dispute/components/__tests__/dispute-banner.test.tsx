import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { DisputeBanner } from "../dispute-banner"
import type {
  DisputeResponse,
  DisputeStatus,
  CounterProposalResponse,
} from "../../types"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, args?: Record<string, unknown>) => {
    if (args) {
      return `${key}::${JSON.stringify(args)}`
    }
    return key
  },
}))

function makeCP(overrides: Partial<CounterProposalResponse> = {}): CounterProposalResponse {
  return {
    id: "cp-1",
    proposer_id: "user-other",
    amount_client: 1000,
    amount_provider: 2000,
    message: "split it",
    status: "pending",
    responded_at: null,
    created_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

function makeDispute(overrides: Partial<DisputeResponse> = {}): DisputeResponse {
  return {
    id: "d-1",
    proposal_id: "p-1",
    conversation_id: "c-1",
    initiator_id: "user-me",
    respondent_id: "user-other",
    client_id: "user-me",
    provider_id: "user-other",
    reason: "non_delivery",
    description: "",
    requested_amount: 50000,
    proposal_amount: 50000,
    status: "open" as DisputeStatus,
    resolution_type: null,
    resolution_amount_client: null,
    resolution_amount_provider: null,
    resolution_note: null,
    initiator_role: "client",
    evidence: [],
    counter_proposals: [],
    cancellation_requested_by: null,
    cancellation_requested_at: null,
    escalated_at: null,
    resolved_at: null,
    created_at: new Date().toISOString(),
    ...overrides,
  }
}

describe("DisputeBanner — base render", () => {
  it("renders the alert with the right status icon for open", () => {
    render(
      <DisputeBanner dispute={makeDispute()} currentUserId="user-me" />,
    )
    expect(screen.getByRole("alert")).toBeInTheDocument()
  })

  it("renders status text matching the dispute status", () => {
    render(
      <DisputeBanner dispute={makeDispute({ status: "negotiation" })} currentUserId="user-me" />,
    )
    expect(screen.getByText("status.negotiation")).toBeInTheDocument()
  })

  it("renders status text for 'resolved'", () => {
    render(
      <DisputeBanner
        dispute={makeDispute({
          status: "resolved",
          resolution_note: "All good",
          resolution_amount_client: 25000,
          resolution_amount_provider: 25000,
          resolved_at: "2026-04-15T00:00:00Z",
        })}
        currentUserId="user-me"
      />,
    )
    expect(screen.getByText("status.resolved")).toBeInTheDocument()
    expect(screen.getByText("All good")).toBeInTheDocument()
  })

  it("renders the daysLeft hint for open disputes within 7 days", () => {
    const created = new Date(Date.now() - 1000 * 60 * 60 * 24 * 2).toISOString()
    render(
      <DisputeBanner
        dispute={makeDispute({ created_at: created })}
        currentUserId="user-me"
      />,
    )
    expect(screen.getByText(/daysLeft/)).toBeInTheDocument()
  })

  it("renders the escalation soon hint when more than 7 days elapsed", () => {
    const old = new Date(Date.now() - 1000 * 60 * 60 * 24 * 10).toISOString()
    render(
      <DisputeBanner
        dispute={makeDispute({ created_at: old })}
        currentUserId="user-me"
      />,
    )
    expect(screen.getByText("escalationSoon")).toBeInTheDocument()
  })

  it("renders the escalated negotiation explainer for status=escalated", () => {
    render(
      <DisputeBanner
        dispute={makeDispute({ status: "escalated" })}
        currentUserId="user-me"
      />,
    )
    expect(
      screen.getByText("escalatedNegotiationStillOpen"),
    ).toBeInTheDocument()
  })
})

describe("DisputeBanner — counter proposal block", () => {
  it("shows the lastProposal block when there is a pending CP from the OTHER party", () => {
    const dispute = makeDispute({
      counter_proposals: [makeCP({ proposer_id: "user-other" })],
      status: "negotiation",
    })
    render(
      <DisputeBanner dispute={dispute} currentUserId="user-me" />,
    )
    expect(screen.getByText("lastProposal")).toBeInTheDocument()
  })

  it("renders accept/reject buttons when handlers are passed and CP from other", () => {
    const dispute = makeDispute({
      counter_proposals: [makeCP({ proposer_id: "user-other" })],
      status: "negotiation",
    })
    const onAccept = vi.fn()
    const onReject = vi.fn()
    render(
      <DisputeBanner
        dispute={dispute}
        currentUserId="user-me"
        onAcceptProposal={onAccept}
        onRejectProposal={onReject}
      />,
    )
    fireEvent.click(screen.getByText("acceptCounter"))
    expect(onAccept).toHaveBeenCalledWith("cp-1")
    fireEvent.click(screen.getByText("rejectCounter"))
    expect(onReject).toHaveBeenCalledWith("cp-1")
  })

  it("does NOT render accept/reject when CP is from the current user", () => {
    const dispute = makeDispute({
      counter_proposals: [makeCP({ proposer_id: "user-me" })],
      status: "negotiation",
    })
    render(
      <DisputeBanner
        dispute={dispute}
        currentUserId="user-me"
        onAcceptProposal={vi.fn()}
        onRejectProposal={vi.fn()}
      />,
    )
    expect(screen.queryByText("acceptCounter")).not.toBeInTheDocument()
  })

  it("shows the 'last proposal refused' feedback when the most recent CP was rejected and was mine", () => {
    const dispute = makeDispute({
      counter_proposals: [
        makeCP({
          proposer_id: "user-me",
          status: "rejected",
        }),
      ],
      status: "negotiation",
    })
    render(<DisputeBanner dispute={dispute} currentUserId="user-me" />)
    expect(screen.getByText("yourLastProposalRefused")).toBeInTheDocument()
  })
})

describe("DisputeBanner — counter-propose action", () => {
  it("renders counter-propose button and calls handler", () => {
    const onCounter = vi.fn()
    render(
      <DisputeBanner
        dispute={makeDispute()}
        currentUserId="user-me"
        onCounterPropose={onCounter}
      />,
    )
    fireEvent.click(screen.getByText("counterPropose"))
    expect(onCounter).toHaveBeenCalled()
  })
})

describe("DisputeBanner — cancellation flow", () => {
  it("renders cancel button only when no cancellation request is pending", () => {
    const onCancel = vi.fn()
    render(
      <DisputeBanner
        dispute={makeDispute()}
        currentUserId="user-me"
        onCancel={onCancel}
      />,
    )
    fireEvent.click(screen.getByText("cancel"))
    expect(onCancel).toHaveBeenCalled()
  })

  it("hides the cancel button when a cancellation request is pending", () => {
    render(
      <DisputeBanner
        dispute={makeDispute({
          cancellation_requested_by: "user-me",
          cancellation_requested_at: "2026-04-15T00:00:00Z",
        })}
        currentUserId="user-me"
        onCancel={vi.fn()}
      />,
    )
    expect(screen.queryByText("cancel")).not.toBeInTheDocument()
  })

  it("shows cancellation request waiting message to the requester", () => {
    render(
      <DisputeBanner
        dispute={makeDispute({
          cancellation_requested_by: "user-me",
          cancellation_requested_at: "2026-04-15T00:00:00Z",
        })}
        currentUserId="user-me"
      />,
    )
    expect(screen.getByText("cancellationRequestPending")).toBeInTheDocument()
    expect(screen.getByText("cancellationRequestWaiting")).toBeInTheDocument()
  })

  it("shows accept/refuse cancellation buttons to the respondent", () => {
    const onAccept = vi.fn()
    const onRefuse = vi.fn()
    render(
      <DisputeBanner
        dispute={makeDispute({
          cancellation_requested_by: "user-other",
          cancellation_requested_at: "2026-04-15T00:00:00Z",
        })}
        currentUserId="user-me"
        onAcceptCancellation={onAccept}
        onRefuseCancellation={onRefuse}
      />,
    )
    fireEvent.click(screen.getByText("acceptCancellation"))
    expect(onAccept).toHaveBeenCalled()
    fireEvent.click(screen.getByText("refuseCancellation"))
    expect(onRefuse).toHaveBeenCalled()
  })

  it("treats undefined cancellation_requested_by as no request", () => {
    const dispute = makeDispute()
    delete (dispute as { cancellation_requested_by?: string | null })
      .cancellation_requested_by
    render(
      <DisputeBanner dispute={dispute} currentUserId="user-me" />,
    )
    expect(
      screen.queryByText("cancellationRequestPending"),
    ).not.toBeInTheDocument()
  })
})

describe("DisputeBanner — terminal states hide actions", () => {
  it("hides the action row when status=resolved", () => {
    render(
      <DisputeBanner
        dispute={makeDispute({ status: "resolved" })}
        currentUserId="user-me"
        onCounterPropose={vi.fn()}
      />,
    )
    expect(screen.queryByText("counterPropose")).not.toBeInTheDocument()
  })

  it("hides the action row when status=cancelled", () => {
    render(
      <DisputeBanner
        dispute={makeDispute({ status: "cancelled" })}
        currentUserId="user-me"
        onCounterPropose={vi.fn()}
      />,
    )
    expect(screen.queryByText("counterPropose")).not.toBeInTheDocument()
  })
})
