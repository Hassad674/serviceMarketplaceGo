import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { SendMessageButton } from "../send-message-button"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock i18n navigation (@i18n/navigation wraps next/navigation)
const mockPush = vi.fn()
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: mockPush }),
}))

// Mock the media query hook — default to desktop; tests can override.
const mockIsDesktop = { current: true }
vi.mock("@/shared/hooks/use-media-query", () => ({
  useMediaQuery: () => mockIsDesktop.current,
}))

// Spy on the chat widget trigger.
const mockOpenChatWithOrg = vi.fn()
vi.mock("@/shared/components/chat-widget/use-chat-widget", () => ({
  openChatWithOrg: (...args: unknown[]) => mockOpenChatWithOrg(...args),
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  MessageSquare: (props: Record<string, unknown>) => <span data-testid="message-icon" {...props} />,
}))

describe("SendMessageButton", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockIsDesktop.current = true
  })

  it("renders the button with the start-conversation label", () => {
    render(<SendMessageButton targetOrgId="org-123" />)
    expect(screen.getByText("startConversation")).toBeDefined()
    expect(screen.getByTestId("message-icon")).toBeDefined()
  })

  it("opens the chat widget on desktop when clicked", () => {
    render(
      <SendMessageButton targetOrgId="org-123" targetDisplayName="Alice" />,
    )

    fireEvent.click(screen.getByText("startConversation"))

    expect(mockOpenChatWithOrg).toHaveBeenCalledWith("org-123", "Alice")
    expect(mockPush).not.toHaveBeenCalled()
  })

  it("navigates to the messages page on mobile when clicked", () => {
    mockIsDesktop.current = false

    render(
      <SendMessageButton targetOrgId="org-123" targetDisplayName="Alice" />,
    )

    fireEvent.click(screen.getByText("startConversation"))

    expect(mockPush).toHaveBeenCalledWith("/messages?to=org-123&name=Alice")
    expect(mockOpenChatWithOrg).not.toHaveBeenCalled()
  })

  it("falls back to an empty display name when none is provided", () => {
    render(<SendMessageButton targetOrgId="org-123" />)

    fireEvent.click(screen.getByText("startConversation"))

    expect(mockOpenChatWithOrg).toHaveBeenCalledWith("org-123", "")
  })
})
