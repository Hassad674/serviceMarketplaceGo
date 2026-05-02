import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { MessageBubble } from "../message-bubble"
import type { Message, ProposalMessageMetadata } from "../../types"

// next-intl mock — keys returned verbatim
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), back: vi.fn() }),
  Link: ({
    children,
    ...rest
  }: {
    children: React.ReactNode
  }) => <a {...rest}>{children as React.ReactNode}</a>,
}))

vi.mock("../message-status-icon", () => ({
  MessageStatusIcon: ({ status }: { status: string }) => (
    <span data-testid={`status-${status}`} />
  ),
}))

vi.mock("../file-message", () => ({
  FileMessage: () => <span data-testid="file-message" />,
}))

vi.mock("../voice-message", () => ({
  VoiceMessage: () => <span data-testid="voice-message" />,
}))

vi.mock("../message-context-menu", () => ({
  MessageContextMenu: ({
    onEdit,
    onDelete,
    onReply,
    onReport,
  }: {
    onEdit?: () => void
    onDelete?: () => void
    onReply?: () => void
    onReport?: () => void
  }) => {
    if (!onEdit && !onDelete && !onReply && !onReport) return null
    return <span data-testid="context-menu" />
  },
}))

vi.mock("../proposal-card", () => ({
  ProposalCard: () => <span data-testid="proposal-card" />,
}))

vi.mock("../proposal-system-message", () => ({
  ProposalSystemMessage: ({ type }: { type: string }) => (
    <span data-testid={`proposal-system-${type}`} />
  ),
  PaymentRequestedMessage: () => (
    <span data-testid="payment-requested-message" />
  ),
  CompletionRequestedMessage: () => (
    <span data-testid="completion-requested-message" />
  ),
  EvaluationRequestMessage: () => (
    <span data-testid="evaluation-request-message" />
  ),
}))

vi.mock("../dispute-system-message", () => ({
  DisputeSystemBubble: ({ type }: { type: string }) => (
    <span data-testid={`dispute-${type}`} />
  ),
}))

// `ReferralSystemMessage` moved to shared (P9 — consumed cross-feature
// by messaging). Mock the shared path that message-bubble now imports.
vi.mock("@/shared/components/referral/referral-system-message", () => ({
  ReferralSystemMessage: ({ type }: { type: string }) => (
    <span data-testid={`referral-${type}`} />
  ),
}))

function makeMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: "m",
    conversation_id: "c",
    sender_id: "u",
    content: "",
    type: "text",
    metadata: null,
    seq: 1,
    status: "sent",
    edited_at: null,
    deleted_at: null,
    created_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

function makeProposalMeta(
  overrides: Partial<ProposalMessageMetadata> = {},
): ProposalMessageMetadata {
  return {
    proposal_id: "p1",
    proposal_title: "T",
    proposal_amount: 0,
    proposal_status: "pending",
    proposal_deadline: null,
    proposal_sender_name: "S",
    proposal_documents_count: 0,
    proposal_version: 1,
    proposal_parent_id: null,
    proposal_client_id: "c1",
    proposal_provider_id: "pr1",
    ...overrides,
  }
}

function defaultState(overrides: Partial<Parameters<typeof MessageBubble>[0]["state"]> = {}) {
  return {
    isOwn: true,
    currentUserId: "u",
    conversationId: "c",
    supersededProposalIds: new Set<string>(),
    ...overrides,
  }
}

function defaultActions() {
  return {
    onEdit: vi.fn(),
    onDelete: vi.fn(),
    onReply: vi.fn(),
    onReport: vi.fn(),
    onReview: vi.fn(),
  }
}

describe("MessageBubble — proposal_sent / proposal_modified", () => {
  it("renders ProposalCard for proposal_sent", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "proposal_sent",
          metadata: makeProposalMeta(),
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(screen.getByTestId("proposal-card")).toBeInTheDocument()
  })

  it("flags superseded proposals with the i18n key", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "proposal_sent",
          metadata: makeProposalMeta({ proposal_id: "p-old" }),
        })}
        state={defaultState({
          supersededProposalIds: new Set(["p-old"]),
        })}
        actions={defaultActions()}
      />,
    )
    expect(screen.getByText(/supersededByVersion/)).toBeInTheDocument()
  })
})

describe("MessageBubble — system messages", () => {
  it.each([
    "proposal_accepted",
    "proposal_declined",
    "proposal_paid",
    "proposal_completed",
    "proposal_modified",
    "milestone_released",
    "milestone_auto_approved",
  ])("renders ProposalSystemMessage for %s", (type) => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: type as Message["type"],
          metadata: makeProposalMeta(),
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    if (type === "proposal_modified") {
      // proposal_modified renders ProposalCard via the first branch
      // (it falls into the "proposal_sent || proposal_modified" path)
      expect(screen.getByTestId("proposal-card")).toBeInTheDocument()
    } else {
      expect(screen.getByTestId(`proposal-system-${type}`)).toBeInTheDocument()
    }
  })

  it("renders PaymentRequestedMessage for proposal_payment_requested", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "proposal_payment_requested",
          metadata: makeProposalMeta(),
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(screen.getByTestId("payment-requested-message")).toBeInTheDocument()
  })

  it("renders CompletionRequestedMessage for proposal_completion_requested", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "proposal_completion_requested",
          metadata: makeProposalMeta(),
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(
      screen.getByTestId("completion-requested-message"),
    ).toBeInTheDocument()
  })

  it("renders EvaluationRequestMessage for evaluation_request", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "evaluation_request",
          metadata: makeProposalMeta(),
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(
      screen.getByTestId("evaluation-request-message"),
    ).toBeInTheDocument()
  })
})

describe("MessageBubble — call system messages", () => {
  it("renders the call_ended bubble with the formatted duration", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "call_ended",
          metadata: { duration: 125 } as never,
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    // Translation key + formatted duration "2:05"
    expect(screen.getByText(/2:05/)).toBeInTheDocument()
  })

  it("renders the call_missed bubble", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "call_missed",
          metadata: null,
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(screen.getByText(/callMissed/)).toBeInTheDocument()
  })
})

describe("MessageBubble — dispute system messages", () => {
  it("renders DisputeSystemBubble for dispute_opened", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "dispute_opened",
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(screen.getByTestId("dispute-dispute_opened")).toBeInTheDocument()
  })
})

describe("MessageBubble — referral system messages", () => {
  it("renders ReferralSystemMessage for referral_intro_sent", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "referral_intro_sent" as Message["type"],
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(
      screen.getByTestId("referral-referral_intro_sent"),
    ).toBeInTheDocument()
  })
})

describe("MessageBubble — deleted messages", () => {
  it("shows the deleted placeholder for soft-deleted messages", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          deleted_at: "2026-04-01T01:00:00Z",
          type: "text",
          content: "",
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(screen.getByText(/messageDeleted/)).toBeInTheDocument()
  })

  it("renders the placeholder right-aligned for own messages", () => {
    const { container } = render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          deleted_at: "2026-04-01T01:00:00Z",
        })}
        state={defaultState({ isOwn: true })}
        actions={defaultActions()}
      />,
    )
    expect(container.querySelector(".justify-end")).not.toBeNull()
  })

  it("renders the placeholder left-aligned for others' messages", () => {
    const { container } = render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          deleted_at: "2026-04-01T01:00:00Z",
        })}
        state={defaultState({ isOwn: false })}
        actions={defaultActions()}
      />,
    )
    expect(container.querySelector(".justify-start")).not.toBeNull()
  })
})

describe("MessageBubble — text fallback", () => {
  it("falls through to TextMessageBubble for plain text messages", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "text",
          content: "Hello",
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(screen.getByText("Hello")).toBeInTheDocument()
  })

  it("renders FileMessage for file-typed messages", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "file",
          content: "x.pdf",
          metadata: { filename: "x.pdf" } as never,
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(screen.getByTestId("file-message")).toBeInTheDocument()
  })

  it("renders VoiceMessage for voice-typed messages", () => {
    render(
      <MessageBubble
        message={makeMessage({
          id: "1",
          type: "voice",
          metadata: { duration: 12 } as never,
        })}
        state={defaultState()}
        actions={defaultActions()}
      />,
    )
    expect(screen.getByTestId("voice-message")).toBeInTheDocument()
  })
})
