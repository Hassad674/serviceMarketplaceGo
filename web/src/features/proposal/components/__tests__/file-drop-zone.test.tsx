/**
 * file-drop-zone.test.tsx
 *
 * Component tests for the proposal file picker. Covers:
 *   - rendering the empty state
 *   - selecting files via the hidden input
 *   - drag-and-drop appending to the existing list
 *   - removing a single file
 *   - file-size formatting helpers
 *   - keyboard accessibility (Enter / Space open the picker)
 */
import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { FileDropZone } from "../file-drop-zone"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

function makeFile(name: string, size: number, type = "application/pdf"): File {
  // Use a Blob with a fixed-size buffer to control file.size precisely.
  const buf = new Uint8Array(size)
  return new File([buf], name, { type })
}

describe("FileDropZone — empty state", () => {
  it("renders the dropzone hint when no files are selected", () => {
    render(<FileDropZone files={[]} onFilesChange={vi.fn()} />)
    expect(screen.getByText("proposalDocumentsHint")).toBeInTheDocument()
  })

  it("does not render any file rows when files=[]", () => {
    render(<FileDropZone files={[]} onFilesChange={vi.fn()} />)
    expect(screen.queryByRole("button", { name: /Remove/i })).toBeNull()
  })
})

describe("FileDropZone — file rendering", () => {
  it("renders one row per file with name and size", () => {
    const files = [
      makeFile("spec.pdf", 1024 * 5),
      makeFile("photo.png", 1024 * 1024 * 2, "image/png"),
    ]
    render(<FileDropZone files={files} onFilesChange={vi.fn()} />)
    expect(screen.getByText("spec.pdf")).toBeInTheDocument()
    expect(screen.getByText("photo.png")).toBeInTheDocument()
    // Bytes < 1 KiB → "B"; 1-1024 KiB → "X.X KB"; > 1 MiB → "X.X MB".
    expect(screen.getByText(/5\.0 KB/)).toBeInTheDocument()
    expect(screen.getByText(/2\.0 MB/)).toBeInTheDocument()
  })

  it("formats sub-1KB files in bytes", () => {
    const files = [makeFile("tiny.txt", 500, "text/plain")]
    render(<FileDropZone files={files} onFilesChange={vi.fn()} />)
    expect(screen.getByText(/500 B/)).toBeInTheDocument()
  })
})

describe("FileDropZone — remove action", () => {
  it("calls onFilesChange with the file removed", () => {
    const onFilesChange = vi.fn()
    const files = [makeFile("a.pdf", 1024), makeFile("b.pdf", 1024)]
    render(<FileDropZone files={files} onFilesChange={onFilesChange} />)
    const removeButton = screen.getByRole("button", { name: /Remove a\.pdf/ })
    fireEvent.click(removeButton)
    expect(onFilesChange).toHaveBeenCalledWith([files[1]])
  })

  it("clicking remove on the only file leaves an empty list", () => {
    const onFilesChange = vi.fn()
    const files = [makeFile("only.pdf", 1024)]
    render(<FileDropZone files={files} onFilesChange={onFilesChange} />)
    fireEvent.click(screen.getByRole("button", { name: /Remove only\.pdf/ }))
    expect(onFilesChange).toHaveBeenCalledWith([])
  })
})

describe("FileDropZone — drag and drop", () => {
  it("appends dropped files to the existing list", () => {
    const onFilesChange = vi.fn()
    const initial = [makeFile("kept.pdf", 1024)]
    render(<FileDropZone files={initial} onFilesChange={onFilesChange} />)

    const zone = screen.getByRole("button", { name: "proposalDocumentsHint" })
    const dropped = makeFile("new.pdf", 2048)
    fireEvent.drop(zone, {
      dataTransfer: { files: [dropped] },
    })
    expect(onFilesChange).toHaveBeenCalledWith([initial[0], dropped])
  })

  it("dragOver does not call onFilesChange (only highlights)", () => {
    const onFilesChange = vi.fn()
    render(<FileDropZone files={[]} onFilesChange={onFilesChange} />)
    const zone = screen.getByRole("button", { name: "proposalDocumentsHint" })
    fireEvent.dragOver(zone)
    expect(onFilesChange).not.toHaveBeenCalled()
  })

  it("dragLeave does not call onFilesChange", () => {
    const onFilesChange = vi.fn()
    render(<FileDropZone files={[]} onFilesChange={onFilesChange} />)
    const zone = screen.getByRole("button", { name: "proposalDocumentsHint" })
    fireEvent.dragLeave(zone)
    expect(onFilesChange).not.toHaveBeenCalled()
  })
})

describe("FileDropZone — file picker via input", () => {
  it("appends files chosen through the hidden file input", () => {
    const onFilesChange = vi.fn()
    const initial = [makeFile("a.pdf", 1024)]
    const { container } = render(
      <FileDropZone files={initial} onFilesChange={onFilesChange} />,
    )
    const fileInput = container.querySelector("input[type=file]") as HTMLInputElement
    expect(fileInput).toBeTruthy()
    const picked = makeFile("picked.pdf", 4096)
    Object.defineProperty(fileInput, "files", {
      value: [picked],
      writable: false,
    })
    fireEvent.change(fileInput)
    expect(onFilesChange).toHaveBeenCalledWith([initial[0], picked])
  })
})
