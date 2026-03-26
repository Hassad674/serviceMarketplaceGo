import { describe, it, expect, vi, beforeAll } from "vitest"
import { render, screen } from "@testing-library/react"
import { MessageArea } from "../message-area"
import type { Message } from "../../types"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
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
    ...overrides,
  }
}

// Mock scrollTo for JSDOM (not available by default)
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
})
