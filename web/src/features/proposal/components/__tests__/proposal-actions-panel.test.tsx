/**
 * proposal-actions-panel.test.tsx
 *
 * Component tests for the proposal actions panel — the right-rail CTA
 * group that decides which buttons to render based on the proposal
 * status, the current milestone state, and whether the viewer is the
 * client/provider/sender/recipient.
 *
 * The component has 9 distinct status branches (pending, accepted,
 * paid, active*3 sub-states, completion_requested, completed, declined,
 * withdrawn, disputed). We exercise every branch the user touches and
 * confirm the right click handlers fire.
 */
import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { ActionsPanel } from "../proposal-actions-panel"
import type {
  MilestoneResponse,
  ProposalResponse,
  ProposalStatus,
} from "../../types"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock("@i18n/navigation", () => ({
  Link: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}))

vi.mock("@/shared/hooks/use-permissions", () => ({
  useHasPermission: () => true,
}))

function makeProposal(status: ProposalStatus): ProposalResponse {
  return {
    id: "p-1",
    conversation_id: "c-1",
    sender_id: "u-1",
    recipient_id: "u-2",
    title: "Test",
    description: "",
    amount: 100000,
    deadline: null,
    status,
    parent_id: null,
    version: 1,
    client_id: "u-2",
    provider_id: "u-1",
    client_name: "Client",
    provider_name: "Provider",
    active_dispute_id: null,
    documents: [],
    payment_mode: "one_time",
    milestones: [],
    accepted_at: null,
    paid_at: null,
    created_at: "2026-04-01T00:00:00Z",
  }
}

function makeMilestone(
  status: MilestoneResponse["status"] = "pending_funding",
): MilestoneResponse {
  return {
    id: "m-1",
    sequence: 1,
    title: "M1",
    description: "",
    amount: 100000,
    status,
    version: 0,
  }
}

function defaultProps(): React.ComponentProps<typeof ActionsPanel> {
  return {
    proposal: makeProposal("pending"),
    currentMilestone: makeMilestone("pending_funding"),
    isRecipient: false,
    isSender: false,
    isClient: false,
    isProvider: false,
    isMutating: false,
    acceptPending: false,
    declinePending: false,
    requestCompletionPending: false,
    completePending: false,
    rejectCompletionPending: false,
    onAccept: vi.fn(),
    onDecline: vi.fn(),
    onModify: vi.fn(),
    onPay: vi.fn(),
    onRequestCompletion: vi.fn(),
    onCompleteProposal: vi.fn(),
    onRejectCompletion: vi.fn(),
  }
}

describe("ActionsPanel — pending state", () => {
  it("recipient sees accept/modify/decline buttons", () => {
    render(<ActionsPanel {...defaultProps()} isRecipient={true} />)
    expect(screen.getByRole("button", { name: /accept/i })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /modify/i })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: /decline/i })).toBeInTheDocument()
  })

  it("non-recipient (sender) sees no action buttons", () => {
    render(<ActionsPanel {...defaultProps()} isSender={true} />)
    expect(screen.queryByRole("button", { name: /accept/i })).toBeNull()
    expect(screen.queryByRole("button", { name: /decline/i })).toBeNull()
  })

  it("clicking accept fires onAccept", () => {
    const props = defaultProps()
    props.isRecipient = true
    render(<ActionsPanel {...props} />)
    fireEvent.click(screen.getByRole("button", { name: /accept/i }))
    expect(props.onAccept).toHaveBeenCalledTimes(1)
  })

  it("clicking decline fires onDecline", () => {
    const props = defaultProps()
    props.isRecipient = true
    render(<ActionsPanel {...props} />)
    fireEvent.click(screen.getByRole("button", { name: /decline/i }))
    expect(props.onDecline).toHaveBeenCalledTimes(1)
  })

  it("clicking modify fires onModify", () => {
    const props = defaultProps()
    props.isRecipient = true
    render(<ActionsPanel {...props} />)
    fireEvent.click(screen.getByRole("button", { name: /modify/i }))
    expect(props.onModify).toHaveBeenCalledTimes(1)
  })

  it("buttons are disabled when isMutating=true", () => {
    const props = defaultProps()
    props.isRecipient = true
    props.isMutating = true
    render(<ActionsPanel {...props} />)
    expect(screen.getByRole("button", { name: /accept/i })).toBeDisabled()
    expect(screen.getByRole("button", { name: /decline/i })).toBeDisabled()
  })
})

describe("ActionsPanel — accepted state", () => {
  it("client sees the proceedToPayment button", () => {
    const props = defaultProps()
    props.proposal = makeProposal("accepted")
    props.isClient = true
    render(<ActionsPanel {...props} />)
    expect(
      screen.getByRole("button", { name: /proceedToPayment/i }),
    ).toBeInTheDocument()
  })

  it("clicking proceedToPayment fires onPay", () => {
    const props = defaultProps()
    props.proposal = makeProposal("accepted")
    props.isClient = true
    render(<ActionsPanel {...props} />)
    fireEvent.click(screen.getByRole("button", { name: /proceedToPayment/i }))
    expect(props.onPay).toHaveBeenCalledTimes(1)
  })

  it("provider does NOT see proceedToPayment", () => {
    const props = defaultProps()
    props.proposal = makeProposal("accepted")
    props.isProvider = true
    render(<ActionsPanel {...props} />)
    expect(screen.queryByRole("button", { name: /proceedToPayment/i })).toBeNull()
  })
})

describe("ActionsPanel — active state, milestone-aware CTAs", () => {
  it("client + pending_funding shows proceedToPayment", () => {
    const props = defaultProps()
    props.proposal = makeProposal("active")
    props.currentMilestone = makeMilestone("pending_funding")
    props.isClient = true
    render(<ActionsPanel {...props} />)
    expect(
      screen.getByRole("button", { name: /proceedToPayment/i }),
    ).toBeInTheDocument()
  })

  it("provider + funded shows terminateMission CTA", () => {
    const props = defaultProps()
    props.proposal = makeProposal("active")
    props.currentMilestone = makeMilestone("funded")
    props.isProvider = true
    render(<ActionsPanel {...props} />)
    expect(
      screen.getByRole("button", { name: /terminateMission/i }),
    ).toBeInTheDocument()
  })

  it("provider + pending_funding shows NO CTA (waits for client to fund)", () => {
    const props = defaultProps()
    props.proposal = makeProposal("active")
    props.currentMilestone = makeMilestone("pending_funding")
    props.isProvider = true
    render(<ActionsPanel {...props} />)
    expect(
      screen.queryByRole("button", { name: /terminateMission|proceedToPayment/i }),
    ).toBeNull()
  })
})

describe("ActionsPanel — completion_requested state", () => {
  it("client sees confirm + reject completion buttons", () => {
    const props = defaultProps()
    props.proposal = makeProposal("completion_requested")
    props.isClient = true
    render(<ActionsPanel {...props} />)
    expect(
      screen.getByRole("button", { name: /confirmCompletion/i }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole("button", { name: /rejectCompletion/i }),
    ).toBeInTheDocument()
  })

  it("clicking confirmCompletion fires onCompleteProposal", () => {
    const props = defaultProps()
    props.proposal = makeProposal("completion_requested")
    props.isClient = true
    render(<ActionsPanel {...props} />)
    fireEvent.click(screen.getByRole("button", { name: /confirmCompletion/i }))
    expect(props.onCompleteProposal).toHaveBeenCalledTimes(1)
  })

  it("clicking rejectCompletion fires onRejectCompletion", () => {
    const props = defaultProps()
    props.proposal = makeProposal("completion_requested")
    props.isClient = true
    render(<ActionsPanel {...props} />)
    fireEvent.click(screen.getByRole("button", { name: /rejectCompletion/i }))
    expect(props.onRejectCompletion).toHaveBeenCalledTimes(1)
  })
})

describe("ActionsPanel — terminal states", () => {
  it.each<ProposalStatus>(["completed", "declined", "withdrawn"])(
    "renders no action buttons for %s",
    (status) => {
      const props = defaultProps()
      props.proposal = makeProposal(status)
      render(<ActionsPanel {...props} />)
      expect(
        screen.queryByRole("button", {
          name: /accept|decline|modify|proceedToPayment|confirmCompletion|rejectCompletion|terminateMission/i,
        }),
      ).toBeNull()
    },
  )
})

describe("ActionsPanel — conversation link", () => {
  it("always renders the goToConversation link with the conversation id", () => {
    render(<ActionsPanel {...defaultProps()} />)
    const link = screen.getByRole("link", { name: /goToConversation/i })
    expect(link).toBeInTheDocument()
    expect(link.getAttribute("href")).toContain("c-1")
  })
})
