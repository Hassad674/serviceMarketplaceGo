import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { DisputeResolutionCard } from "../dispute-resolution-card"
import type { DisputeResponse } from "../../types"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, args?: Record<string, unknown>) => {
    if (args) {
      return `${key}::${JSON.stringify(args)}`
    }
    return key
  },
}))

function makeDispute(overrides: Partial<DisputeResponse> = {}): DisputeResponse {
  return {
    id: "d-1",
    proposal_id: "p-1",
    conversation_id: "c-1",
    initiator_id: "user-c",
    respondent_id: "user-p",
    client_id: "user-c",
    provider_id: "user-p",
    reason: "non_delivery",
    description: "",
    requested_amount: 50000,
    proposal_amount: 50000,
    status: "resolved",
    resolution_type: "split",
    resolution_amount_client: 30000,
    resolution_amount_provider: 20000,
    resolution_note: "Decided 60/40",
    initiator_role: "client",
    evidence: [],
    counter_proposals: [],
    cancellation_requested_by: null,
    cancellation_requested_at: null,
    escalated_at: null,
    resolved_at: "2026-04-15T00:00:00Z",
    created_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

describe("DisputeResolutionCard — resolved state", () => {
  it("renders the decision title", () => {
    render(
      <DisputeResolutionCard dispute={makeDispute()} currentUserId="user-c" />,
    )
    expect(screen.getByText("decisionTitle")).toBeInTheDocument()
  })

  it("highlights the client cell when current user is the client", () => {
    render(
      <DisputeResolutionCard dispute={makeDispute()} currentUserId="user-c" />,
    )
    // resolution_amount_client = 30000 → 60% of 50000 → 60% display
    expect(screen.getByText(/decisionYourShare/)).toHaveTextContent("60")
  })

  it("highlights the provider cell when current user is the provider", () => {
    render(
      <DisputeResolutionCard dispute={makeDispute()} currentUserId="user-p" />,
    )
    expect(screen.getByText(/decisionYourShare/)).toHaveTextContent("40")
  })

  it("renders the resolution note", () => {
    render(
      <DisputeResolutionCard
        dispute={makeDispute({ resolution_note: "Custom decision text" })}
        currentUserId="user-c"
      />,
    )
    expect(screen.getByText("Custom decision text")).toBeInTheDocument()
  })

  it("does not render the message block when no note", () => {
    render(
      <DisputeResolutionCard
        dispute={makeDispute({ resolution_note: null })}
        currentUserId="user-c"
      />,
    )
    expect(screen.queryByText("decisionMessage")).not.toBeInTheDocument()
  })

  it("renders the resolved date when present", () => {
    render(
      <DisputeResolutionCard
        dispute={makeDispute({ resolved_at: "2026-04-30T00:00:00Z" })}
        currentUserId="user-c"
      />,
    )
    expect(screen.getByText(/decisionRenderedOn/)).toBeInTheDocument()
  })

  it("handles zero total gracefully (no division by zero)", () => {
    render(
      <DisputeResolutionCard
        dispute={makeDispute({
          resolution_amount_client: 0,
          resolution_amount_provider: 0,
        })}
        currentUserId="user-c"
      />,
    )
    // Should not crash; render with 0%
    expect(screen.getByText("decisionTitle")).toBeInTheDocument()
  })
})

describe("DisputeResolutionCard — cancelled state", () => {
  it("renders the cancelled title", () => {
    render(
      <DisputeResolutionCard
        dispute={makeDispute({ status: "cancelled" })}
        currentUserId="user-c"
      />,
    )
    expect(screen.getByText("disputeCancelledTitle")).toBeInTheDocument()
    expect(screen.getByText("disputeCancelledSubtitle")).toBeInTheDocument()
  })
})

describe("DisputeResolutionCard — non-terminal states", () => {
  it("renders nothing when status=open", () => {
    const { container } = render(
      <DisputeResolutionCard
        dispute={makeDispute({ status: "open" })}
        currentUserId="user-c"
      />,
    )
    expect(container.firstChild).toBeNull()
  })

  it("renders nothing when status=negotiation", () => {
    const { container } = render(
      <DisputeResolutionCard
        dispute={makeDispute({ status: "negotiation" })}
        currentUserId="user-c"
      />,
    )
    expect(container.firstChild).toBeNull()
  })

  it("renders nothing when status=escalated", () => {
    const { container } = render(
      <DisputeResolutionCard
        dispute={makeDispute({ status: "escalated" })}
        currentUserId="user-c"
      />,
    )
    expect(container.firstChild).toBeNull()
  })
})
