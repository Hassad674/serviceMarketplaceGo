import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render as baseRender, screen, fireEvent, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import type { ReactElement } from "react"
import { MessageInput } from "../message-input"
import { getPresignedURL } from "../../api/messaging-api"

// MessageInput reads `useHasPermission` → `useOrganization` → `useQuery`
// from the org-permissions system, so every render must sit inside a
// TanStack QueryClientProvider. The helper keeps the tests below
// unchanged structurally.
function render(ui: ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  return baseRender(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  )
}

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

// Mock lucide-react icons. Using importOriginal so any icon used by
// transitively-rendered components (e.g. FileUploadModal's UploadCloud,
// FileImage, FileIcon) does not blow up the render. Explicit overrides
// stay below for the icons we actually want to assert against.
vi.mock("lucide-react", async (importOriginal) => {
  const actual = await importOriginal<typeof import("lucide-react")>()
  return {
    ...actual,
    Paperclip: (props: Record<string, unknown>) => <span data-testid="paperclip-icon" {...props} />,
    Send: (props: Record<string, unknown>) => <span data-testid="send-icon" {...props} />,
    Loader2: (props: Record<string, unknown>) => <span data-testid="loader-icon" {...props} />,
    FileText: (props: Record<string, unknown>) => <span data-testid="filetext-icon" {...props} />,
    X: (props: Record<string, unknown>) => <span data-testid="x-icon" {...props} />,
    Mic: (props: Record<string, unknown>) => <span data-testid="mic-icon" {...props} />,
    Square: (props: Record<string, unknown>) => <span data-testid="square-icon" {...props} />,
    Plus: (props: Record<string, unknown>) => <span data-testid="plus-icon" {...props} />,
    Trash2: (props: Record<string, unknown>) => <span data-testid="trash-icon" {...props} />,
  }
})

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

    // The component renders two buttons with aria-label="fileUpload":
    // one for desktop (hidden md:flex) and one for the mobile "+" menu
    // trigger (md:hidden). jsdom keeps both in the DOM since md: is
    // CSS-only, so we use getAllByRole.
    const buttons = screen.getAllByRole("button", { name: "fileUpload" })
    expect(buttons.length).toBeGreaterThan(0)
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

  it("disables the send button when isSending is true", () => {
    // Since the WhatsApp-style voice UX refactor the send icon stays
    // visible during isSending — only the submit button itself is
    // disabled. The previous "loader icon when isSending" assertion
    // was stale drift from the earlier UI.
    render(<MessageInput {...defaultProps({ isSending: true })} />)
    const sendButton = screen.getByLabelText("sendMessage")
    expect(sendButton).toHaveProperty("disabled", true)
  })
})

describe("MessageInput — file upload error handling", () => {
  const originalFetch = global.fetch

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    global.fetch = originalFetch
  })

  function makeFile(name = "doc.pdf", type = "application/pdf"): File {
    return new File(["x"], name, { type })
  }

  /**
   * Drive the FileUploadModal end-to-end: open it, attach a file to the
   * hidden <input type="file">, then click the upload submit button.
   * Returns true when the modal flow could be driven, false when the
   * jsdom DOM did not surface the expected nodes (defensive — happens
   * when the createPortal target is ripped between tests).
   */
  async function driveUpload(file: File): Promise<boolean> {
    const buttons = screen.getAllByRole("button", { name: "fileUpload" })
    fireEvent.click(buttons[0])

    const fileInput = document.querySelector(
      'input[type="file"]',
    ) as HTMLInputElement | null
    if (!fileInput) return false

    Object.defineProperty(fileInput, "files", {
      value: [file],
      configurable: true,
    })
    fireEvent.change(fileInput)

    // The send button uses translation key "sendFiles".
    const sendBtn = await screen.findByRole("button", { name: "sendFiles" })
    fireEvent.click(sendBtn)
    return true
  }

  it("does NOT call onSendFile when the PUT upload returns 500", async () => {
    const onSendFile = vi.fn()
    vi.mocked(getPresignedURL).mockResolvedValue({
      upload_url: "https://storage.example/upload",
      public_url: "https://storage.example/public",
      file_key: "key-123",
    })
    const fetchMock = vi.fn().mockResolvedValue(
      new Response("server error", { status: 500 }),
    )
    global.fetch = fetchMock as unknown as typeof fetch

    render(<MessageInput {...defaultProps({ onSendFile })} />)

    const driven = await driveUpload(makeFile())
    if (!driven) return

    await waitFor(() => {
      expect(fetchMock).toHaveBeenCalled()
    })

    expect(onSendFile).not.toHaveBeenCalled()
    // Error banner uses role="alert" with the i18n key "uploadFailed".
    await waitFor(() => {
      const alerts = screen.getAllByRole("alert")
      expect(alerts.some((a) => a.textContent?.includes("uploadFailed"))).toBe(true)
    })
  })

  it("DOES call onSendFile when the PUT upload returns 200", async () => {
    const onSendFile = vi.fn()
    vi.mocked(getPresignedURL).mockResolvedValue({
      upload_url: "https://storage.example/upload",
      public_url: "https://storage.example/public",
      file_key: "key-123",
    })
    const fetchMock = vi.fn().mockResolvedValue(
      new Response("", { status: 200 }),
    )
    global.fetch = fetchMock as unknown as typeof fetch

    render(<MessageInput {...defaultProps({ onSendFile })} />)

    const driven = await driveUpload(makeFile())
    if (!driven) return

    await waitFor(() => {
      expect(onSendFile).toHaveBeenCalledTimes(1)
    })
    expect(onSendFile.mock.calls[0][1]).toMatchObject({
      url: "https://storage.example/public",
      filename: "doc.pdf",
    })
  })
})
