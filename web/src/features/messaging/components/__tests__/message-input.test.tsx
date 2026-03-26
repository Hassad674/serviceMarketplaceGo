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
  Smile: (props: Record<string, unknown>) => <span data-testid="smile-icon" {...props} />,
  Mic: (props: Record<string, unknown>) => <span data-testid="mic-icon" {...props} />,
  Send: (props: Record<string, unknown>) => <span data-testid="send-icon" {...props} />,
}))

describe("MessageInput", () => {
  it("renders input field", () => {
    render(<MessageInput />)

    const input = screen.getByPlaceholderText("writeMessage")
    expect(input).toBeDefined()
  })

  it("send button disabled when empty", () => {
    render(<MessageInput />)

    const sendButton = screen.getByRole("button", { name: "Send message" })
    expect(sendButton).toHaveProperty("disabled", true)
  })

  it("send button enabled when text entered", () => {
    render(<MessageInput />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "Hello!" } })

    const sendButton = screen.getByRole("button", { name: "Send message" })
    expect(sendButton).toHaveProperty("disabled", false)
  })

  it("calls onSend with message text on submit", () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "Hello world" } })

    const sendButton = screen.getByRole("button", { name: "Send message" })
    fireEvent.click(sendButton)

    expect(onSend).toHaveBeenCalledWith("Hello world")
  })

  it("clears input after sending", () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const input = screen.getByPlaceholderText("writeMessage") as HTMLInputElement
    fireEvent.change(input, { target: { value: "Hello" } })
    fireEvent.click(screen.getByRole("button", { name: "Send message" }))

    expect(input.value).toBe("")
  })

  it("does not send whitespace-only messages", () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "   " } })

    const sendButton = screen.getByRole("button", { name: "Send message" })
    expect(sendButton).toHaveProperty("disabled", true)
  })

  it("trims message before sending", () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "  Hello world  " } })
    fireEvent.click(screen.getByRole("button", { name: "Send message" }))

    expect(onSend).toHaveBeenCalledWith("Hello world")
  })

  it("sends on Enter key press", () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const input = screen.getByPlaceholderText("writeMessage")
    fireEvent.change(input, { target: { value: "Enter test" } })
    fireEvent.keyDown(input, { key: "Enter", code: "Enter" })

    expect(onSend).toHaveBeenCalledWith("Enter test")
  })

  it("renders attachment and emoji buttons", () => {
    render(<MessageInput />)

    expect(screen.getByRole("button", { name: "Attach file" })).toBeDefined()
    expect(screen.getByRole("button", { name: "Add emoji" })).toBeDefined()
    expect(screen.getByRole("button", { name: "Voice note" })).toBeDefined()
  })
})
