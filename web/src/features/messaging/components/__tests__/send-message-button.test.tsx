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

// Mock the session hook — tests override `data` to flip auth state.
const mockUserData: { current: unknown } = { current: { id: "viewer-1" } }
vi.mock("@/shared/hooks/use-user", () => ({
  useUser: () => ({ data: mockUserData.current }),
}))

// Spy on the chat widget trigger.
const mockOpenChatWithOrg = vi.fn()
vi.mock("@/shared/components/chat-widget/use-chat-widget", () => ({
  openChatWithOrg: (...args: unknown[]) => mockOpenChatWithOrg(...args),
}))

// Spy on the analytics lead-tracker — must NOT fire on unauth click.
const mockTrackLead = vi.fn()
vi.mock("@/shared/lib/analytics-events", () => ({
  trackLead: (...args: unknown[]) => mockTrackLead(...args),
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  MessageSquare: (props: Record<string, unknown>) => <span data-testid="message-icon" {...props} />,
}))

describe("SendMessageButton", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockIsDesktop.current = true
    mockUserData.current = { id: "viewer-1" }
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
    expect(mockTrackLead).toHaveBeenCalledWith({ profileId: "org-123", persona: "freelance" })
  })

  it("navigates to the messages page on mobile when clicked", () => {
    mockIsDesktop.current = false

    render(
      <SendMessageButton targetOrgId="org-123" targetDisplayName="Alice" />,
    )

    fireEvent.click(screen.getByText("startConversation"))

    expect(mockPush).toHaveBeenCalledWith("/messages?to=org-123&name=Alice")
    expect(mockOpenChatWithOrg).not.toHaveBeenCalled()
    expect(mockTrackLead).toHaveBeenCalledWith({ profileId: "org-123", persona: "freelance" })
  })

  it("falls back to an empty display name when none is provided", () => {
    render(<SendMessageButton targetOrgId="org-123" />)

    fireEvent.click(screen.getByText("startConversation"))

    expect(mockOpenChatWithOrg).toHaveBeenCalledWith("org-123", "")
  })

  // Bug B regression: an unauthenticated visitor clicking the
  // "Send a message" button on a public profile must be redirected
  // to /login with a `next` query param so they bounce back to the
  // same profile after signing in. Previously the desktop path
  // silently dropped the click (chat widget bootstrap failed) and
  // the mobile path leaned on middleware to redirect — both
  // delivered an inconsistent UX.
  describe("when the visitor is unauthenticated", () => {
    beforeEach(() => {
      mockUserData.current = undefined
      Object.defineProperty(window, "location", {
        writable: true,
        value: { ...window.location, pathname: "/agencies/org-123" },
      })
    })

    it("redirects to /login with a next param on desktop", () => {
      render(<SendMessageButton targetOrgId="org-123" targetDisplayName="Alice" />)

      fireEvent.click(screen.getByText("startConversation"))

      expect(mockPush).toHaveBeenCalledWith(
        `/login?next=${encodeURIComponent("/agencies/org-123")}`,
      )
      expect(mockOpenChatWithOrg).not.toHaveBeenCalled()
    })

    it("redirects to /login with a next param on mobile", () => {
      mockIsDesktop.current = false

      render(<SendMessageButton targetOrgId="org-123" targetDisplayName="Alice" />)

      fireEvent.click(screen.getByText("startConversation"))

      expect(mockPush).toHaveBeenCalledWith(
        `/login?next=${encodeURIComponent("/agencies/org-123")}`,
      )
      expect(mockOpenChatWithOrg).not.toHaveBeenCalled()
    })

    it("does NOT fire the GA4 lead event on the bounce-to-login click", () => {
      render(<SendMessageButton targetOrgId="org-123" targetDisplayName="Alice" />)

      fireEvent.click(screen.getByText("startConversation"))

      expect(mockTrackLead).not.toHaveBeenCalled()
    })
  })
})
