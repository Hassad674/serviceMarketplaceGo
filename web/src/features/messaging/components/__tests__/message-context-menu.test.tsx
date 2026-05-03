/**
 * message-context-menu.test.tsx
 *
 * Tests the per-message dropdown menu (reply / edit / delete / report).
 * Covers:
 *   - menu opens on the trigger click
 *   - menu closes on outside click
 *   - clicking an action fires the corresponding callback AND closes the menu
 *   - actions only render when the matching prop is provided
 */
import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { MessageContextMenu } from "../message-context-menu"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

describe("MessageContextMenu — open/close", () => {
  it("renders the trigger button collapsed", () => {
    render(<MessageContextMenu onReply={vi.fn()} />)
    expect(
      screen.getByRole("button", { name: "Message options" }),
    ).toBeInTheDocument()
    // The dropdown is collapsed initially.
    expect(screen.queryByText("reply")).toBeNull()
  })

  it("opens the dropdown on trigger click", () => {
    render(<MessageContextMenu onReply={vi.fn()} />)
    fireEvent.click(screen.getByRole("button", { name: "Message options" }))
    expect(screen.getByText("reply")).toBeInTheDocument()
  })

  it("closes the dropdown on a second trigger click", () => {
    render(<MessageContextMenu onReply={vi.fn()} />)
    const trigger = screen.getByRole("button", { name: "Message options" })
    fireEvent.click(trigger)
    fireEvent.click(trigger)
    expect(screen.queryByText("reply")).toBeNull()
  })

  it("closes on outside click", () => {
    render(
      <div>
        <MessageContextMenu onReply={vi.fn()} />
        <button data-testid="outside">Outside</button>
      </div>,
    )
    fireEvent.click(screen.getByRole("button", { name: "Message options" }))
    expect(screen.getByText("reply")).toBeInTheDocument()
    fireEvent.mouseDown(screen.getByTestId("outside"))
    expect(screen.queryByText("reply")).toBeNull()
  })
})

describe("MessageContextMenu — action callbacks", () => {
  it("clicking reply fires onReply and closes the menu", () => {
    const onReply = vi.fn()
    render(<MessageContextMenu onReply={onReply} />)
    fireEvent.click(screen.getByRole("button", { name: "Message options" }))
    fireEvent.click(screen.getByText("reply"))
    expect(onReply).toHaveBeenCalledTimes(1)
    expect(screen.queryByText("reply")).toBeNull()
  })

  it("clicking edit fires onEdit and closes the menu", () => {
    const onEdit = vi.fn()
    render(<MessageContextMenu onEdit={onEdit} />)
    fireEvent.click(screen.getByRole("button", { name: "Message options" }))
    fireEvent.click(screen.getByText("editMessage"))
    expect(onEdit).toHaveBeenCalledTimes(1)
  })

  it("clicking delete fires onDelete", () => {
    const onDelete = vi.fn()
    render(<MessageContextMenu onDelete={onDelete} />)
    fireEvent.click(screen.getByRole("button", { name: "Message options" }))
    fireEvent.click(screen.getByText("deleteMessage"))
    expect(onDelete).toHaveBeenCalledTimes(1)
  })

  it("clicking report fires onReport", () => {
    const onReport = vi.fn()
    render(<MessageContextMenu onReport={onReport} />)
    fireEvent.click(screen.getByRole("button", { name: "Message options" }))
    fireEvent.click(screen.getByText("report"))
    expect(onReport).toHaveBeenCalledTimes(1)
  })
})

describe("MessageContextMenu — conditional action rendering", () => {
  it("renders all 4 actions when every callback is provided", () => {
    render(
      <MessageContextMenu
        onReply={vi.fn()}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onReport={vi.fn()}
      />,
    )
    fireEvent.click(screen.getByRole("button", { name: "Message options" }))
    expect(screen.getByText("reply")).toBeInTheDocument()
    expect(screen.getByText("editMessage")).toBeInTheDocument()
    expect(screen.getByText("deleteMessage")).toBeInTheDocument()
    expect(screen.getByText("report")).toBeInTheDocument()
  })

  it("hides reply when onReply is not provided", () => {
    render(<MessageContextMenu onEdit={vi.fn()} />)
    fireEvent.click(screen.getByRole("button", { name: "Message options" }))
    expect(screen.queryByText("reply")).toBeNull()
    expect(screen.getByText("editMessage")).toBeInTheDocument()
  })

  it("hides edit when onEdit is not provided", () => {
    render(<MessageContextMenu onReply={vi.fn()} />)
    fireEvent.click(screen.getByRole("button", { name: "Message options" }))
    expect(screen.queryByText("editMessage")).toBeNull()
  })

  it("renders nothing inside the dropdown when no callbacks are provided", () => {
    render(<MessageContextMenu />)
    fireEvent.click(screen.getByRole("button", { name: "Message options" }))
    // Dropdown opens but is empty.
    expect(screen.queryByText("reply")).toBeNull()
    expect(screen.queryByText("editMessage")).toBeNull()
    expect(screen.queryByText("deleteMessage")).toBeNull()
    expect(screen.queryByText("report")).toBeNull()
  })
})
