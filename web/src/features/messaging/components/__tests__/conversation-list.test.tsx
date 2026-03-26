import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { ConversationList } from "../conversation-list"
import type { Conversation, ConversationRole } from "../../types"

// Mock next-intl — return the key as the translated string
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock lucide-react icons as simple spans
vi.mock("lucide-react", () => ({
  Search: (props: Record<string, unknown>) => <span data-testid="search-icon" {...props} />,
}))

function createConversation(overrides: Partial<Conversation> = {}): Conversation {
  return {
    id: "conv-1",
    name: "Alice Martin",
    role: "freelancer",
    lastMessage: "Sounds good!",
    lastMessageAt: "10:30",
    avatar: null,
    unread: 0,
    online: false,
    ...overrides,
  }
}

function defaultProps(overrides: Record<string, unknown> = {}) {
  return {
    conversations: [
      createConversation({ id: "conv-1", name: "Alice Martin", role: "freelancer" as ConversationRole }),
      createConversation({ id: "conv-2", name: "Bob Agency", role: "agency" as ConversationRole }),
      createConversation({ id: "conv-3", name: "Corp Enterprise", role: "enterprise" as ConversationRole }),
    ],
    activeId: null,
    roleFilter: "all" as "all" | ConversationRole,
    searchQuery: "",
    onSelect: vi.fn(),
    onRoleFilterChange: vi.fn(),
    onSearchChange: vi.fn(),
    ...overrides,
  }
}

describe("ConversationList", () => {
  it("renders conversation items", () => {
    render(<ConversationList {...defaultProps()} />)

    expect(screen.getByText("Alice Martin")).toBeDefined()
    expect(screen.getByText("Bob Agency")).toBeDefined()
    expect(screen.getByText("Corp Enterprise")).toBeDefined()
  })

  it("filters by role when tab clicked", () => {
    const onRoleFilterChange = vi.fn()
    render(
      <ConversationList
        {...defaultProps({ onRoleFilterChange })}
      />,
    )

    // Click the "agency" role filter tab
    const agencyTab = screen.getByText("agency")
    fireEvent.click(agencyTab)

    expect(onRoleFilterChange).toHaveBeenCalledWith("agency")
  })

  it("shows unread badge", () => {
    const conversations = [
      createConversation({ id: "conv-1", name: "Alice", unread: 5 }),
    ]
    render(
      <ConversationList {...defaultProps({ conversations })} />,
    )

    expect(screen.getByText("5")).toBeDefined()
  })

  it("search filters conversations by name", () => {
    const conversations = [
      createConversation({ id: "conv-1", name: "Alice Martin" }),
      createConversation({ id: "conv-2", name: "Bob Smith" }),
    ]
    render(
      <ConversationList
        {...defaultProps({ conversations, searchQuery: "Alice" })}
      />,
    )

    expect(screen.getByText("Alice Martin")).toBeDefined()
    expect(screen.queryByText("Bob Smith")).toBeNull()
  })

  it("shows online indicator", () => {
    const conversations = [
      createConversation({ id: "conv-1", name: "Online User", online: true }),
    ]

    const { container } = render(
      <ConversationList {...defaultProps({ conversations })} />,
    )

    // Online indicator is a span with bg-emerald-500 class
    const onlineIndicators = container.querySelectorAll(".bg-emerald-500")
    expect(onlineIndicators.length).toBeGreaterThan(0)
  })

  it("shows empty state when no conversations match", () => {
    render(
      <ConversationList
        {...defaultProps({ conversations: [] })}
      />,
    )

    expect(screen.getByText("noConversations")).toBeDefined()
  })

  it("calls onSelect when conversation clicked", () => {
    const onSelect = vi.fn()
    const conversations = [
      createConversation({ id: "conv-42", name: "Clickable User" }),
    ]
    render(
      <ConversationList {...defaultProps({ conversations, onSelect })} />,
    )

    fireEvent.click(screen.getByText("Clickable User"))

    expect(onSelect).toHaveBeenCalledWith("conv-42")
  })

  it("calls onSearchChange when typing in search input", () => {
    const onSearchChange = vi.fn()
    render(
      <ConversationList {...defaultProps({ onSearchChange })} />,
    )

    const searchInput = screen.getByPlaceholderText("searchPlaceholder")
    fireEvent.change(searchInput, { target: { value: "test" } })

    expect(onSearchChange).toHaveBeenCalledWith("test")
  })

  it("shows role filter tabs", () => {
    render(<ConversationList {...defaultProps()} />)

    expect(screen.getByText("allRoles")).toBeDefined()
    expect(screen.getByText("agency")).toBeDefined()
    expect(screen.getByText("freelancer")).toBeDefined()
    expect(screen.getByText("enterprise")).toBeDefined()
  })

  it("filters conversations by role when roleFilter is set", () => {
    render(
      <ConversationList
        {...defaultProps({ roleFilter: "agency" })}
      />,
    )

    expect(screen.queryByText("Alice Martin")).toBeNull()
    expect(screen.getByText("Bob Agency")).toBeDefined()
    expect(screen.queryByText("Corp Enterprise")).toBeNull()
  })
})
