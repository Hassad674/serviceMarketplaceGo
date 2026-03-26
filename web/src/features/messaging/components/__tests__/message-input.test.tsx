import { describe, it, expect, vi } from "vitest"
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
}))

// Mock messaging-api
vi.mock("../../api/messaging-api", () => ({
  getPresignedURL: vi.fn(),
}))

function defaultProps() {
  return {
    onSend: vi.fn(),
    onSendFile: vi.fn(),
    onTyping: vi.fn(),
    isSending: false,
  }
}

describe("MessageInput", () => {
  it("renders input field", () => {
    render(<MessageInput {...defaultProps()} />)

    const input = screen.getByPlaceholderText("writeMessage")
    expect(input).toBeDefined()
  })

  it("send button disabled when empty", () => {
    render(<MessageInput {...defaultProps()} />)

    const sendButton = screen.getByRole("button", { name: "sendMessage" })
    expect(sendButton).toHaveProperty("disabled", true)
  })

  it("send button enabled when text entered", () => {
    render(<MessageInput {...defaultProps()} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "Hello!" } })

    const sendButton = screen.getByRole("button", { name: "sendMessage" })
    expect(sendButton).toHaveProperty("disabled", false)
  })

  it("calls onSend with message text on submit", () => {
    const props = defaultProps()
    render(<MessageInput {...props} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "Hello world" } })

    const sendButton = screen.getByRole("button", { name: "sendMessage" })
    fireEvent.click(sendButton)

    expect(props.onSend).toHaveBeenCalledWith("Hello world")
  })

  it("clears input after sending", () => {
    const props = defaultProps()
    render(<MessageInput {...props} />)

    const input = screen.getByPlaceholderText("writeMessage") as HTMLInputElement
    fireEvent.change(input, { target: { value: "Hello" } })
    fireEvent.click(screen.getByRole("button", { name: "sendMessage" }))

    expect(input.value).toBe("")
  })

  it("does not send whitespace-only messages", () => {
    const props = defaultProps()
    render(<MessageInput {...props} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "   " } })

    const sendButton = screen.getByRole("button", { name: "sendMessage" })
    expect(sendButton).toHaveProperty("disabled", true)
  })

  it("trims message before sending", () => {
    const props = defaultProps()
    render(<MessageInput {...props} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "  Hello world  " } })
    fireEvent.click(screen.getByRole("button", { name: "sendMessage" }))

    expect(props.onSend).toHaveBeenCalledWith("Hello world")
  })

  it("sends on Enter key press", () => {
    const props = defaultProps()
    render(<MessageInput {...props} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "Enter test" } })
    fireEvent.keyDown(input, { key: "Enter", code: "Enter" })

    expect(props.onSend).toHaveBeenCalledWith("Enter test")
  })

  it("renders attachment button", () => {
    render(<MessageInput {...defaultProps()} />)

    expect(screen.getByRole("button", { name: "fileUpload" })).toBeDefined()
  })
})
