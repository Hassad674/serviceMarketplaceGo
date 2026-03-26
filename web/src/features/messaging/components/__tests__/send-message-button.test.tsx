import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { SendMessageButton } from "../send-message-button"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  MessageSquare: (props: Record<string, unknown>) => <span data-testid="message-icon" {...props} />,
  Send: (props: Record<string, unknown>) => <span data-testid="send-icon" {...props} />,
  X: (props: Record<string, unknown>) => <span data-testid="close-icon" {...props} />,
}))

const mockMutate = vi.fn()
const mockIsPending = false

// Mock the hook
vi.mock("../../hooks/use-start-conversation", () => ({
  useStartConversation: () => ({
    mutate: mockMutate,
    isPending: mockIsPending,
  }),
}))

describe("SendMessageButton", () => {
  it("renders the initial button with message icon", () => {
    render(<SendMessageButton targetUserId="user-123" />)

    const button = screen.getByText("startConversation")
    expect(button).toBeDefined()
    expect(screen.getByTestId("message-icon")).toBeDefined()
  })

  it("opens message form when button clicked", () => {
    render(<SendMessageButton targetUserId="user-123" />)

    fireEvent.click(screen.getByText("startConversation"))

    // Form should appear with a textarea
    expect(screen.getByPlaceholderText("writeMessage")).toBeDefined()
  })

  it("has close button in expanded form", () => {
    render(<SendMessageButton targetUserId="user-123" />)

    fireEvent.click(screen.getByText("startConversation"))

    const closeButton = screen.getByLabelText("close")
    expect(closeButton).toBeDefined()
  })

  it("closes form when close button clicked", () => {
    render(<SendMessageButton targetUserId="user-123" />)

    // Open
    fireEvent.click(screen.getByText("startConversation"))
    expect(screen.getByPlaceholderText("writeMessage")).toBeDefined()

    // Close
    fireEvent.click(screen.getByLabelText("close"))

    // Should be back to button state
    expect(screen.getByText("startConversation")).toBeDefined()
  })

  it("send button is disabled when textarea is empty", () => {
    render(<SendMessageButton targetUserId="user-123" />)

    fireEvent.click(screen.getByText("startConversation"))

    // The send button in the form
    const sendButtons = screen.getAllByText("sendMessage")
    const formSendButton = sendButtons[sendButtons.length - 1]
    expect(formSendButton.closest("button")).toHaveProperty("disabled", true)
  })

  it("send button enables when text entered", () => {
    render(<SendMessageButton targetUserId="user-123" />)

    fireEvent.click(screen.getByText("startConversation"))

    const textarea = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(textarea, { target: { value: "Hello!" } })

    const sendButtons = screen.getAllByText("sendMessage")
    const formSendButton = sendButtons[sendButtons.length - 1]
    expect(formSendButton.closest("button")).toHaveProperty("disabled", false)
  })
})
