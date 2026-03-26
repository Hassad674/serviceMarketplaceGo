import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { MessageStatusIcon } from "../message-status-icon"

// Mock lucide-react icons
vi.mock("lucide-react", () => ({
  Clock: (props: Record<string, unknown>) => (
    <span data-testid="clock-icon" {...props} />
  ),
  Check: (props: Record<string, unknown>) => (
    <span data-testid="check-icon" {...props} />
  ),
  CheckCheck: (props: Record<string, unknown>) => (
    <span data-testid="checkcheck-icon" {...props} />
  ),
}))

describe("MessageStatusIcon", () => {
  it("renders clock icon for sending status", () => {
    render(<MessageStatusIcon status="sending" />)
    expect(screen.getByLabelText("Sending")).toBeDefined()
  })

  it("renders check icon for sent status", () => {
    render(<MessageStatusIcon status="sent" />)
    expect(screen.getByLabelText("Sent")).toBeDefined()
  })

  it("renders double check icon for delivered status", () => {
    render(<MessageStatusIcon status="delivered" />)
    expect(screen.getByLabelText("Delivered")).toBeDefined()
  })

  it("renders blue double check icon for read status", () => {
    const { container } = render(<MessageStatusIcon status="read" />)
    const icon = screen.getByLabelText("Read")
    expect(icon).toBeDefined()
    // Read status has blue color class
    expect(container.querySelector(".text-blue-300")).not.toBeNull()
  })
})
