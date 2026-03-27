import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { ConversationHeader } from "../conversation-header"
import type { Conversation } from "../../types"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, params?: Record<string, string>) => {
    if (key === "typing" && params?.name) {
      return `${params.name} is typing`
    }
    return key
  },
}))

// Mock @i18n/navigation (used by ConversationHeader for "Start Project" button)
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), back: vi.fn() }),
  Link: ({ children, ...props }: Record<string, unknown>) => <a {...props}>{children as React.ReactNode}</a>,
}))

// Mock next/image
vi.mock("next/image", () => ({
  default: (props: Record<string, unknown>) => {
    // eslint-disable-next-line @next/next/no-img-element, jsx-a11y/alt-text
    return <img {...props} />
  },
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  ArrowLeft: (props: Record<string, unknown>) => <span data-testid="arrow-left-icon" {...props} />,
  Wifi: (props: Record<string, unknown>) => <span data-testid="wifi-icon" {...props} />,
  WifiOff: (props: Record<string, unknown>) => <span data-testid="wifi-off-icon" {...props} />,
  FileText: (props: Record<string, unknown>) => <span data-testid="file-text-icon" {...props} />,
}))

// Mock TypingIndicator
vi.mock("../typing-indicator", () => ({
  TypingIndicator: ({ userName }: { userName: string }) => (
    <span data-testid="typing-indicator">{userName} is typing</span>
  ),
}))

function createConversation(overrides: Partial<Conversation> = {}): Conversation {
  return {
    id: "conv-1",
    other_user_id: "user-2",
    other_user_name: "Alice Smith",
    other_user_role: "provider",
    other_photo_url: "",
    last_message: "Hello",
    last_message_at: "2026-03-26T10:00:00Z",
    unread_count: 0,
    last_message_seq: 5,
    online: false,
    ...overrides,
  }
}

describe("ConversationHeader", () => {
  it("renders other user name", () => {
    render(
      <ConversationHeader
        conversation={createConversation({ other_user_name: "Alice Smith" })}
        isConnected={true}
      />,
    )

    expect(screen.getByText("Alice Smith")).toBeDefined()
  })

  it("shows online status when user is online", () => {
    render(
      <ConversationHeader
        conversation={createConversation({ online: true })}
        isConnected={true}
      />,
    )

    // Both the status text and the sr-only text render "online"
    const elements = screen.getAllByText("online")
    expect(elements.length).toBeGreaterThanOrEqual(1)
  })

  it("shows offline status when user is offline", () => {
    render(
      <ConversationHeader
        conversation={createConversation({ online: false })}
        isConnected={true}
      />,
    )

    expect(screen.getByText("offline")).toBeDefined()
  })

  it("shows typing indicator when typing user name provided", () => {
    render(
      <ConversationHeader
        conversation={createConversation()}
        typingUserName="Alice Smith"
        isConnected={true}
      />,
    )

    expect(screen.getByTestId("typing-indicator")).toBeDefined()
    expect(screen.getByText("Alice Smith is typing")).toBeDefined()
  })

  it("does not show typing indicator when no typing user", () => {
    render(
      <ConversationHeader
        conversation={createConversation()}
        isConnected={true}
      />,
    )

    expect(screen.queryByTestId("typing-indicator")).toBeNull()
  })

  it("shows wifi icon when connected", () => {
    render(
      <ConversationHeader
        conversation={createConversation()}
        isConnected={true}
      />,
    )

    expect(screen.getByTestId("wifi-icon")).toBeDefined()
  })

  it("shows wifi-off icon when disconnected", () => {
    render(
      <ConversationHeader
        conversation={createConversation()}
        isConnected={false}
      />,
    )

    expect(screen.getByTestId("wifi-off-icon")).toBeDefined()
  })

  it("renders back button when onBack provided", () => {
    const onBack = vi.fn()
    render(
      <ConversationHeader
        conversation={createConversation()}
        onBack={onBack}
        isConnected={true}
      />,
    )

    const backButton = screen.getByLabelText("back")
    expect(backButton).toBeDefined()
  })

  it("calls onBack when back button clicked", () => {
    const onBack = vi.fn()
    render(
      <ConversationHeader
        conversation={createConversation()}
        onBack={onBack}
        isConnected={true}
      />,
    )

    fireEvent.click(screen.getByLabelText("back"))

    expect(onBack).toHaveBeenCalledOnce()
  })

  it("does not render back button when onBack not provided", () => {
    render(
      <ConversationHeader
        conversation={createConversation()}
        isConnected={true}
      />,
    )

    expect(screen.queryByLabelText("back")).toBeNull()
  })

  it("renders initials when no photo url", () => {
    render(
      <ConversationHeader
        conversation={createConversation({ other_user_name: "Bob Jones", other_photo_url: "" })}
        isConnected={true}
      />,
    )

    expect(screen.getByText("BJ")).toBeDefined()
  })

  it("shows online indicator dot when user is online", () => {
    const { container } = render(
      <ConversationHeader
        conversation={createConversation({ online: true })}
        isConnected={true}
      />,
    )

    const indicator = container.querySelector(".bg-emerald-500")
    expect(indicator).not.toBeNull()
  })
})
