import { describe, it, expect, vi, beforeAll, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { MessageArea } from "../message-area"
import type { Message } from "../../types"

// Stub IntersectionObserver before any rendering (JSDOM does not have it)
class MockIntersectionObserver {
  observe = vi.fn()
  unobserve = vi.fn()
  disconnect = vi.fn()
  constructor(_callback: IntersectionObserverCallback, _options?: IntersectionObserverInit) {}
}
vi.stubGlobal("IntersectionObserver", MockIntersectionObserver)

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock @i18n/navigation
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), back: vi.fn() }),
  Link: ({ children, ...props }: Record<string, unknown>) => <a {...props}>{children as React.ReactNode}</a>,
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  MessageSquare: (props: Record<string, unknown>) => <span data-testid="message-square-icon" {...props} />,
  MoreHorizontal: (props: Record<string, unknown>) => <span data-testid="more-icon" {...props} />,
  Pencil: (props: Record<string, unknown>) => <span data-testid="pencil-icon" {...props} />,
  Trash2: (props: Record<string, unknown>) => <span data-testid="trash-icon" {...props} />,
  Clock: (props: Record<string, unknown>) => <span data-testid="clock-icon" {...props} />,
  Check: (props: Record<string, unknown>) => <span data-testid="check-icon" {...props} />,
  CheckCheck: (props: Record<string, unknown>) => <span data-testid="checkcheck-icon" {...props} />,
  Download: (props: Record<string, unknown>) => <span data-testid="download-icon" {...props} />,
  FileText: (props: Record<string, unknown>) => <span data-testid="filetext-icon" {...props} />,
  CheckCircle2: (props: Record<string, unknown>) => <span data-testid="check-circle-icon" {...props} />,
  XCircle: (props: Record<string, unknown>) => <span data-testid="x-circle-icon" {...props} />,
  CreditCard: (props: Record<string, unknown>) => <span data-testid="credit-card-icon" {...props} />,
  DollarSign: (props: Record<string, unknown>) => <span data-testid="dollar-icon" {...props} />,
  Reply: (props: Record<string, unknown>) => <span data-testid="reply-icon" {...props} />,
  RotateCcw: (props: Record<string, unknown>) => <span data-testid="rotate-icon" {...props} />,
}))

// Mock sub-components that are imported
vi.mock("../message-status-icon", () => ({
  MessageStatusIcon: ({ status }: { status: string }) => (
    <span data-testid={`status-${status}`} />
  ),
}))

vi.mock("../file-message", () => ({
  FileMessage: () => <span data-testid="file-message" />,
}))

vi.mock("../message-context-menu", () => ({
  MessageContextMenu: ({ onEdit, onDelete }: { onEdit: () => void; onDelete: () => void }) => (
    <span data-testid="context-menu">
      <button onClick={onEdit} data-testid="edit-btn">edit</button>
      <button onClick={onDelete} data-testid="delete-btn">delete</button>
    </span>
  ),
}))

vi.mock("../proposal-card", () => ({
  ProposalCard: () => <span data-testid="proposal-card" />,
}))

function createMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: "msg-1",
    conversation_id: "conv-1",
    sender_id: "user-1",
    content: "Hello world",
    type: "text",
    metadata: null,
    seq: 1,
    status: "sent",
    edited_at: null,
    deleted_at: null,
    created_at: new Date().toISOString(),
    ...overrides,
  }
}

function defaultProps(overrides: Partial<Parameters<typeof MessageArea>[0]> = {}) {
  return {
    messages: [] as Message[],
    currentUserId: "user-1",
    isLoading: false,
    hasMore: false,
    onLoadMore: vi.fn(),
    onEdit: vi.fn(),
    onDelete: vi.fn(),
    onReply: vi.fn(),
    conversationId: "conv-1",
    ...overrides,
  }
}

// Mock scrollTo for JSDOM
beforeAll(() => {
  Element.prototype.scrollTo = vi.fn()
})

describe("MessageArea", () => {
  it("shows empty state when no messages", () => {
    render(<MessageArea {...defaultProps()} />)

    expect(screen.getByText("noMessages")).toBeDefined()
  })

  it("shows skeleton when loading", () => {
    const { container } = render(
      <MessageArea {...defaultProps({ isLoading: true })} />,
    )

    // Skeleton uses animate-pulse class (check class attribute string)
    const allDivs = container.querySelectorAll("div")
    const hasAnimatePulse = Array.from(allDivs).some((el) =>
      el.className.includes("animate-pulse"),
    )
    expect(hasAnimatePulse).toBe(true)
  })

  it("renders messages", () => {
    const messages = [
      createMessage({ id: "msg-1", content: "First message" }),
      createMessage({ id: "msg-2", content: "Second message", sender_id: "user-2" }),
    ]
    render(
      <MessageArea {...defaultProps({ messages })} />,
    )

    expect(screen.getByText("First message")).toBeDefined()
    expect(screen.getByText("Second message")).toBeDefined()
  })

  it("detects own messages by flex-row-reverse layout", () => {
    const messages = [
      createMessage({ id: "msg-1", sender_id: "user-1", content: "My message" }),
    ]
    const { container } = render(
      <MessageArea {...defaultProps({ messages, currentUserId: "user-1" })} />,
    )

    // Own messages use flex-row-reverse class
    const allDivs = container.querySelectorAll("div")
    const hasOwnLayout = Array.from(allDivs).some((el) =>
      el.className.includes("flex-row-reverse"),
    )
    expect(hasOwnLayout).toBe(true)
  })

  it("renders other user messages without flex-row-reverse", () => {
    const messages = [
      createMessage({ id: "msg-1", sender_id: "user-2", content: "Their message" }),
    ]
    const { container } = render(
      <MessageArea {...defaultProps({ messages, currentUserId: "user-1" })} />,
    )

    // Other user messages use flex-row (not flex-row-reverse)
    const groupDivs = Array.from(container.querySelectorAll("div")).filter((el) =>
      el.className.includes("group"),
    )
    expect(groupDivs.length).toBeGreaterThan(0)
    expect(groupDivs[0].className).toContain("flex-row")
    expect(groupDivs[0].className).not.toContain("flex-row-reverse")
  })

  it("renders deleted message placeholder", () => {
    const messages = [
      createMessage({
        id: "msg-1",
        content: "",
        deleted_at: "2026-03-26T10:00:00Z",
      }),
    ]
    render(<MessageArea {...defaultProps({ messages })} />)

    expect(screen.getByText("messageDeleted")).toBeDefined()
  })

  it("shows edited label for edited messages", () => {
    const messages = [
      createMessage({
        id: "msg-1",
        content: "Edited content",
        edited_at: "2026-03-26T10:05:00Z",
      }),
    ]
    render(<MessageArea {...defaultProps({ messages })} />)

    expect(screen.getByText("Edited content")).toBeDefined()
    // The edited label is rendered as "(messageEdited)"
    expect(screen.getByText(/messageEdited/)).toBeDefined()
  })

  it("renders file message component for file type", () => {
    const messages = [
      createMessage({
        id: "msg-1",
        type: "file",
        content: "document.pdf",
        metadata: {
          url: "https://example.com/file.pdf",
          filename: "document.pdf",
          size: 1024,
          mime_type: "application/pdf",
        },
      }),
    ]
    render(<MessageArea {...defaultProps({ messages })} />)

    expect(screen.getByTestId("file-message")).toBeDefined()
  })

  it("shows status icons on own messages", () => {
    const messages = [
      createMessage({ id: "msg-1", sender_id: "user-1", status: "sent" }),
    ]
    render(
      <MessageArea {...defaultProps({ messages, currentUserId: "user-1" })} />,
    )

    expect(screen.getByTestId("status-sent")).toBeDefined()
  })

  it("does not show status icons on other user messages", () => {
    const messages = [
      createMessage({ id: "msg-1", sender_id: "user-2", status: "sent" }),
    ]
    render(
      <MessageArea {...defaultProps({ messages, currentUserId: "user-1" })} />,
    )

    expect(screen.queryByTestId("status-sent")).toBeNull()
  })

  it("renders load more button when hasMore is true", () => {
    const messages = [
      createMessage({ id: "msg-1", content: "Hello" }),
    ]
    render(
      <MessageArea {...defaultProps({ messages, hasMore: true })} />,
    )

    expect(screen.getByText("loadMore")).toBeDefined()
  })

  it("does not render load more button when hasMore is false", () => {
    const messages = [
      createMessage({ id: "msg-1", content: "Hello" }),
    ]
    render(
      <MessageArea {...defaultProps({ messages, hasMore: false })} />,
    )

    expect(screen.queryByText("loadMore")).toBeNull()
  })

  it("calls onLoadMore when load more button clicked", () => {
    const onLoadMore = vi.fn()
    const messages = [
      createMessage({ id: "msg-1", content: "Hello" }),
    ]
    render(
      <MessageArea {...defaultProps({ messages, hasMore: true, onLoadMore })} />,
    )

    fireEvent.click(screen.getByText("loadMore"))

    expect(onLoadMore).toHaveBeenCalledOnce()
  })

  it("shows context menu for own non-temp messages", () => {
    const messages = [
      createMessage({ id: "msg-1", sender_id: "user-1", content: "My message" }),
    ]
    render(
      <MessageArea {...defaultProps({ messages, currentUserId: "user-1" })} />,
    )

    expect(screen.getByTestId("context-menu")).toBeDefined()
  })

  it("does not show context menu for temp messages", () => {
    const messages = [
      createMessage({ id: "temp-123", sender_id: "user-1", content: "Sending..." }),
    ]
    render(
      <MessageArea {...defaultProps({ messages, currentUserId: "user-1" })} />,
    )

    expect(screen.queryByTestId("context-menu")).toBeNull()
  })

  it("does not show context menu for other user messages", () => {
    const messages = [
      createMessage({ id: "msg-1", sender_id: "user-2", content: "Their message" }),
    ]
    render(
      <MessageArea {...defaultProps({ messages, currentUserId: "user-1" })} />,
    )

    expect(screen.queryByTestId("context-menu")).toBeNull()
  })

  it("treats optimistic sender as own message", () => {
    const messages = [
      createMessage({ id: "temp-123", sender_id: "optimistic", content: "Sending..." }),
    ]
    const { container } = render(
      <MessageArea {...defaultProps({ messages, currentUserId: "user-1" })} />,
    )

    // Optimistic messages appear as own (flex-row-reverse)
    const allDivs = container.querySelectorAll("div")
    const hasOwnLayout = Array.from(allDivs).some((el) =>
      el.className.includes("flex-row-reverse"),
    )
    expect(hasOwnLayout).toBe(true)
  })
})
