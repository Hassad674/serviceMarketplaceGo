/**
 * file-message.download.test.tsx
 *
 * Coverage gap fill for `FileMessage`. The existing
 * `file-message.test.tsx` covers the rendering surface; this file
 * exercises the click-to-download path (fetch -> blob -> anchor -> click)
 * and the fallback behaviour when fetch fails.
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { render, screen, fireEvent, waitFor } from "@testing-library/react"
import { FileMessage } from "../file-message"
import type { FileMetadata } from "../../types"

vi.mock("next/image", () => ({
  default: (props: Record<string, unknown>) => {
    // eslint-disable-next-line @next/next/no-img-element, jsx-a11y/alt-text
    return <img {...props} />
  },
}))

vi.mock("lucide-react", () => ({
  FileText: (props: Record<string, unknown>) => <span data-testid="filetext-icon" {...props} />,
  Download: (props: Record<string, unknown>) => <span data-testid="download-icon" {...props} />,
}))

function createMetadata(overrides: Partial<FileMetadata> = {}): FileMetadata {
  return {
    url: "https://storage.example.com/file.pdf",
    filename: "document.pdf",
    size: 1024,
    mime_type: "application/pdf",
    ...overrides,
  }
}

const mockFetch = vi.fn()
const originalCreateObjectURL = URL.createObjectURL
const originalRevokeObjectURL = URL.revokeObjectURL
const originalOpen = window.open

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch)
  URL.createObjectURL = vi.fn().mockReturnValue("blob:http://localhost/x")
  URL.revokeObjectURL = vi.fn()
  window.open = vi.fn() as unknown as typeof window.open
})

afterEach(() => {
  vi.unstubAllGlobals()
  URL.createObjectURL = originalCreateObjectURL
  URL.revokeObjectURL = originalRevokeObjectURL
  window.open = originalOpen
  vi.clearAllMocks()
})

describe("FileMessage — download interaction", () => {
  it("clicking a non-image file fetches the URL and triggers a download", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      blob: async () => new Blob(["pdf-bytes"]),
    })
    render(<FileMessage metadata={createMetadata()} isOwn={false} />)
    fireEvent.click(screen.getByRole("button"))
    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith("https://storage.example.com/file.pdf")
    })
    expect(URL.createObjectURL).toHaveBeenCalled()
    expect(URL.revokeObjectURL).toHaveBeenCalled()
  })

  it("falls back to window.open when fetch fails", async () => {
    mockFetch.mockRejectedValueOnce(new Error("network down"))
    render(<FileMessage metadata={createMetadata()} isOwn={false} />)
    fireEvent.click(screen.getByRole("button"))
    await waitFor(() => {
      expect(window.open).toHaveBeenCalledWith(
        "https://storage.example.com/file.pdf",
        "_blank",
      )
    })
  })

  it("renders distinct background classes for own vs other messages", () => {
    const { rerender } = render(
      <FileMessage metadata={createMetadata()} isOwn={false} />,
    )
    const otherClasses = screen.getByRole("button").className
    rerender(<FileMessage metadata={createMetadata()} isOwn={true} />)
    const ownClasses = screen.getByRole("button").className
    expect(otherClasses).not.toBe(ownClasses)
    expect(ownClasses).toMatch(/rose/)
  })

  it("renders the download caption under image previews", () => {
    render(
      <FileMessage
        metadata={createMetadata({ mime_type: "image/png", filename: "pic.png", size: 2048 })}
        isOwn={false}
      />,
    )
    // The image branch shows a clickable button with the filename and size.
    expect(screen.getByText("pic.png", { exact: false })).toBeInTheDocument()
    expect(screen.getByText(/2\.0 KB/)).toBeInTheDocument()
  })
})
