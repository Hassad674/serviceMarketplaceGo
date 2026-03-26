import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { FileMessage } from "../file-message"
import type { FileMetadata } from "../../types"

// Mock lucide-react icons
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

describe("FileMessage", () => {
  it("renders filename for non-image files", () => {
    render(<FileMessage metadata={createMetadata()} isOwn={false} />)

    expect(screen.getByText("document.pdf")).toBeDefined()
  })

  it("renders file size for non-image files", () => {
    render(
      <FileMessage metadata={createMetadata({ size: 1024 })} isOwn={false} />,
    )

    expect(screen.getByText("1.0 KB")).toBeDefined()
  })

  it("renders file size in MB for large files", () => {
    render(
      <FileMessage
        metadata={createMetadata({ size: 5 * 1024 * 1024 })}
        isOwn={false}
      />,
    )

    expect(screen.getByText("5.0 MB")).toBeDefined()
  })

  it("renders file size in bytes for small files", () => {
    render(
      <FileMessage metadata={createMetadata({ size: 512 })} isOwn={false} />,
    )

    expect(screen.getByText("512 B")).toBeDefined()
  })

  it("renders download icon for non-image files", () => {
    render(<FileMessage metadata={createMetadata()} isOwn={false} />)

    expect(screen.getByTestId("download-icon")).toBeDefined()
  })

  it("renders file text icon for non-image files", () => {
    render(<FileMessage metadata={createMetadata()} isOwn={false} />)

    expect(screen.getByTestId("filetext-icon")).toBeDefined()
  })

  it("renders image for image mime type", () => {
    render(
      <FileMessage
        metadata={createMetadata({
          mime_type: "image/png",
          filename: "photo.png",
        })}
        isOwn={false}
      />,
    )

    // Image renders an <img> tag with alt text
    const img = screen.getByAltText("photo.png")
    expect(img).toBeDefined()
    expect(img.tagName).toBe("IMG")
  })

  it("links to file URL", () => {
    render(
      <FileMessage
        metadata={createMetadata({ url: "https://storage.example.com/myfile.pdf" })}
        isOwn={false}
      />,
    )

    const link = screen.getByRole("link")
    expect(link).toHaveAttribute("href", "https://storage.example.com/myfile.pdf")
    expect(link).toHaveAttribute("target", "_blank")
    expect(link).toHaveAttribute("rel", "noopener noreferrer")
  })
})
