import { describe, it, expect, vi, beforeEach } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { MessageInput } from "../message-input"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  Paperclip: (props: Record<string, unknown>) => <span data-testid="paperclip-icon" {...props} />,
  Send: (props: Record<string, unknown>) => <span data-testid="send-icon" {...props} />,
  Loader2: (props: Record<string, unknown>) => <span data-testid="loader-icon" {...props} />,
  FileText: (props: Record<string, unknown>) => <span data-testid="filetext-icon" {...props} />,
  X: (props: Record<string, unknown>) => <span data-testid="x-icon" {...props} />,
  Mic: (props: Record<string, unknown>) => <span data-testid="mic-icon" {...props} />,
  Square: (props: Record<string, unknown>) => <span data-testid="square-icon" {...props} />,
  Plus: (props: Record<string, unknown>) => <span data-testid="plus-icon" {...props} />,
  Trash2: (props: Record<string, unknown>) => <span data-testid="trash-icon" {...props} />,
}))

// Mock i18n navigation
vi.mock("@i18n/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}))

// Mock the messaging API (for getPresignedURL used in file upload)
vi.mock("../../api/messaging-api", () => ({
  getPresignedURL: vi.fn(),
}))

function defaultProps(overrides: Partial<Parameters<typeof MessageInput>[0]> = {}) {
  return {
    conversationId: "conv-123",
    otherUserId: "user-456",
    onSend: vi.fn(),
    onSendFile: vi.fn(),
    onTyping: vi.fn(),
    isSending: false,
    ...overrides,
  }
}

describe("MessageInput", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("renders text input", () => {
    render(<MessageInput {...defaultProps()} />)

    expect(screen.getByPlaceholderText("writeMessage")).toBeDefined()
  })

  it("renders send button", () => {
    render(<MessageInput {...defaultProps()} />)

    expect(screen.getByLabelText("sendMessage")).toBeDefined()
  })

  it("renders file attachment button", () => {
    render(<MessageInput {...defaultProps()} />)

    // The button has aria-label="fileUpload"
    const button = screen.getByRole("button", { name: "fileUpload" })
    expect(button).toBeDefined()
  })

  it("send button is disabled when input is empty", () => {
    render(<MessageInput {...defaultProps()} />)

    const sendButton = screen.getByLabelText("sendMessage")
    expect(sendButton).toHaveProperty("disabled", true)
  })

  it("send button enables when text entered", () => {
    render(<MessageInput {...defaultProps()} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "Hello!" } })

    const sendButton = screen.getByLabelText("sendMessage")
    expect(sendButton).toHaveProperty("disabled", false)
  })

  it("send button stays disabled with whitespace-only input", () => {
    render(<MessageInput {...defaultProps()} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "   " } })

    const sendButton = screen.getByLabelText("sendMessage")
    expect(sendButton).toHaveProperty("disabled", true)
  })

  it("calls onSend with trimmed content on form submit", () => {
    const onSend = vi.fn()
    render(<MessageInput {...defaultProps({ onSend })} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "  Hello!  " } })
    fireEvent.submit(input.closest("form")!)

    expect(onSend).toHaveBeenCalledWith("Hello!", undefined)
  })

  it("clears input after successful send", () => {
    const onSend = vi.fn()
    render(<MessageInput {...defaultProps({ onSend })} />)

    const input = screen.getByPlaceholderText("writeMessage") as HTMLInputElement
    fireEvent.change(input, { target: { value: "Hello!" } })
    fireEvent.submit(input.closest("form")!)

    expect(input.value).toBe("")
  })

  it("calls onSend on Enter key press", () => {
    const onSend = vi.fn()
    render(<MessageInput {...defaultProps({ onSend })} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "Hello!" } })
    fireEvent.keyDown(input, { key: "Enter", shiftKey: false })

    expect(onSend).toHaveBeenCalledWith("Hello!", undefined)
  })

  it("does not call onSend when input is empty", () => {
    const onSend = vi.fn()
    render(<MessageInput {...defaultProps({ onSend })} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.submit(input.closest("form")!)

    expect(onSend).not.toHaveBeenCalled()
  })

  it("does not call onSend when isSending is true", () => {
    const onSend = vi.fn()
    render(<MessageInput {...defaultProps({ onSend, isSending: true })} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "Hello!" } })
    fireEvent.submit(input.closest("form")!)

    expect(onSend).not.toHaveBeenCalled()
  })

  it("calls onTyping when typing (first keystroke)", () => {
    const onTyping = vi.fn()
    render(<MessageInput {...defaultProps({ onTyping })} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "H" } })

    expect(onTyping).toHaveBeenCalledOnce()
  })

  it("disables input when isSending is true", () => {
    render(<MessageInput {...defaultProps({ isSending: true })} />)

    const input = screen.getByPlaceholderText("writeMessage") as HTMLInputElement
    expect(input.disabled).toBe(true)
  })

  it("shows loader icon when isSending", () => {
    render(<MessageInput {...defaultProps({ isSending: true })} />)

    expect(screen.getByTestId("loader-icon")).toBeDefined()
  })
})
