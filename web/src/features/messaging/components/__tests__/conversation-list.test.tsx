import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { ConversationList } from "../conversation-list"
import type { Conversation } from "../../types"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  Search: (props: Record<string, unknown>) => <span data-testid="search-icon" {...props} />,
}))

function createConversation(overrides: Partial<Conversation> = {}): Conversation {
  return {
    id: "conv-1",
    other_user_id: "user-2",
    other_user_name: "Alice Smith",
    other_user_role: "provider",
    other_photo_url: "",
    last_message: "Hello there",
    last_message_at: "2026-03-26T10:00:00Z",
    unread_count: 0,
    last_message_seq: 5,
    online: false,
    ...overrides,
  }
}

function defaultProps(overrides: Partial<Parameters<typeof ConversationList>[0]> = {}) {
  return {
    conversations: [] as Conversation[],
    activeId: null as string | null,
    roleFilter: "all",
    searchQuery: "",
    onSelect: vi.fn(),
    onRoleFilterChange: vi.fn(),
    onSearchChange: vi.fn(),
    ...overrides,
  }
}

describe("ConversationList", () => {
  it("renders the title", () => {
    render(<ConversationList {...defaultProps()} />)

    expect(screen.getByText("title")).toBeDefined()
  })

  it("renders role filter tabs", () => {
    render(<ConversationList {...defaultProps()} />)

    expect(screen.getByText("allRoles")).toBeDefined()
    expect(screen.getByText("agency")).toBeDefined()
    expect(screen.getByText("freelancer")).toBeDefined()
    expect(screen.getByText("enterprise")).toBeDefined()
  })

  it("renders search input", () => {
    render(<ConversationList {...defaultProps()} />)

    expect(screen.getByPlaceholderText("searchPlaceholder")).toBeDefined()
  })

  it("shows empty state when no conversations", () => {
    render(<ConversationList {...defaultProps()} />)

    expect(screen.getByText("noConversations")).toBeDefined()
  })

  it("renders conversations with other_user_name", () => {
    const conversations = [
      createConversation({ id: "conv-1", other_user_name: "Alice Smith" }),
      createConversation({ id: "conv-2", other_user_name: "Bob Jones" }),
    ]
    render(<ConversationList {...defaultProps({ conversations })} />)

    expect(screen.getByText("Alice Smith")).toBeDefined()
    expect(screen.getByText("Bob Jones")).toBeDefined()
  })

  it("shows last message preview", () => {
    const conversations = [
      createConversation({ last_message: "Hey, how are you?" }),
    ]
    render(<ConversationList {...defaultProps({ conversations })} />)

    expect(screen.getByText("Hey, how are you?")).toBeDefined()
  })

  it("shows unread count badge when > 0", () => {
    const conversations = [
      createConversation({ unread_count: 3 }),
    ]
    render(<ConversationList {...defaultProps({ conversations })} />)

    expect(screen.getByText("3")).toBeDefined()
  })

  it("does not show unread badge when count is 0", () => {
    const conversations = [
      createConversation({ unread_count: 0 }),
    ]
    render(<ConversationList {...defaultProps({ conversations })} />)

    // No element with a number badge
    expect(screen.queryByText("0")).toBeNull()
  })

  it("shows online indicator when user is online", () => {
    const conversations = [
      createConversation({ online: true }),
    ]
    const { container } = render(<ConversationList {...defaultProps({ conversations })} />)

    // Online indicator uses bg-emerald-500 class
    const indicator = container.querySelector(".bg-emerald-500")
    expect(indicator).not.toBeNull()
  })

  it("does not show online indicator when user is offline", () => {
    const conversations = [
      createConversation({ online: false }),
    ]
    const { container } = render(<ConversationList {...defaultProps({ conversations })} />)

    const indicator = container.querySelector(".bg-emerald-500")
    expect(indicator).toBeNull()
  })

  it("renders initials when no photo_url", () => {
    const conversations = [
      createConversation({ other_user_name: "Alice Smith", other_photo_url: "" }),
    ]
    render(<ConversationList {...defaultProps({ conversations })} />)

    expect(screen.getByText("AS")).toBeDefined()
  })

  it("calls onSelect when conversation clicked", () => {
    const onSelect = vi.fn()
    const conversations = [
      createConversation({ id: "conv-123" }),
    ]
    render(<ConversationList {...defaultProps({ conversations, onSelect })} />)

    fireEvent.click(screen.getByText("Alice Smith"))

    expect(onSelect).toHaveBeenCalledWith("conv-123")
  })

  it("calls onRoleFilterChange when role tab clicked", () => {
    const onRoleFilterChange = vi.fn()
    render(<ConversationList {...defaultProps({ onRoleFilterChange })} />)

    fireEvent.click(screen.getByText("agency"))

    expect(onRoleFilterChange).toHaveBeenCalledWith("agency")
  })

  it("calls onSearchChange when search input changes", () => {
    const onSearchChange = vi.fn()
    render(<ConversationList {...defaultProps({ onSearchChange })} />)

    const input = screen.getByPlaceholderText("searchPlaceholder")
    fireEvent.change(input, { target: { value: "Alice" } })

    expect(onSearchChange).toHaveBeenCalledWith("Alice")
  })

  it("filters conversations by role", () => {
    const conversations = [
      createConversation({ id: "conv-1", other_user_name: "Alice", other_user_role: "provider" }),
      createConversation({ id: "conv-2", other_user_name: "Bob", other_user_role: "agency" }),
    ]
    render(
      <ConversationList {...defaultProps({ conversations, roleFilter: "provider" })} />,
    )

    expect(screen.getByText("Alice")).toBeDefined()
    expect(screen.queryByText("Bob")).toBeNull()
  })

  it("filters conversations by search query", () => {
    const conversations = [
      createConversation({ id: "conv-1", other_user_name: "Alice Smith" }),
      createConversation({ id: "conv-2", other_user_name: "Bob Jones" }),
    ]
    render(
      <ConversationList {...defaultProps({ conversations, searchQuery: "alice" })} />,
    )

    expect(screen.getByText("Alice Smith")).toBeDefined()
    expect(screen.queryByText("Bob Jones")).toBeNull()
  })

  it("shows no-messages text when last_message is null", () => {
    const conversations = [
      createConversation({ last_message: null }),
    ]
    render(<ConversationList {...defaultProps({ conversations })} />)

    expect(screen.getByText("noMessages")).toBeDefined()
  })

  it("highlights active conversation", () => {
    const conversations = [
      createConversation({ id: "conv-1" }),
    ]
    const { container } = render(
      <ConversationList {...defaultProps({ conversations, activeId: "conv-1" })} />,
    )

    // Active conversation gets bg-gray-50 class
    const buttons = container.querySelectorAll("button")
    const activeButton = Array.from(buttons).find((b) =>
      b.className.includes("bg-gray-50"),
    )
    expect(activeButton).toBeDefined()
  })
})
