import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { PaymentModeToggle } from "../payment-mode-toggle"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

describe("PaymentModeToggle", () => {
  it("renders both tabs", () => {
    render(<PaymentModeToggle value="one_time" onChange={vi.fn()} />)
    expect(screen.getByText("oneTime")).toBeInTheDocument()
    expect(screen.getByText("milestone")).toBeInTheDocument()
  })

  it("highlights one_time as selected when value=one_time", () => {
    render(<PaymentModeToggle value="one_time" onChange={vi.fn()} />)
    const oneTime = screen.getByText("oneTime").closest("button")!
    expect(oneTime.getAttribute("aria-selected")).toBe("true")
    const milestone = screen.getByText("milestone").closest("button")!
    expect(milestone.getAttribute("aria-selected")).toBe("false")
  })

  it("highlights milestone as selected when value=milestone", () => {
    render(<PaymentModeToggle value="milestone" onChange={vi.fn()} />)
    const milestone = screen.getByText("milestone").closest("button")!
    expect(milestone.getAttribute("aria-selected")).toBe("true")
  })

  it("calls onChange with one_time when oneTime tab is clicked", () => {
    const onChange = vi.fn()
    render(<PaymentModeToggle value="milestone" onChange={onChange} />)
    fireEvent.click(screen.getByText("oneTime"))
    expect(onChange).toHaveBeenCalledWith("one_time")
  })

  it("calls onChange with milestone when milestone tab is clicked", () => {
    const onChange = vi.fn()
    render(<PaymentModeToggle value="one_time" onChange={onChange} />)
    fireEvent.click(screen.getByText("milestone"))
    expect(onChange).toHaveBeenCalledWith("milestone")
  })

  it("renders the appropriate hint for one_time", () => {
    render(<PaymentModeToggle value="one_time" onChange={vi.fn()} />)
    expect(screen.getByText("oneTimeHint")).toBeInTheDocument()
  })

  it("renders the appropriate hint for milestone", () => {
    render(<PaymentModeToggle value="milestone" onChange={vi.fn()} />)
    expect(screen.getByText("milestoneHint")).toBeInTheDocument()
  })

  it("disables both buttons when disabled=true", () => {
    render(<PaymentModeToggle value="one_time" onChange={vi.fn()} disabled />)
    const oneTime = screen.getByText("oneTime").closest("button")!
    const milestone = screen.getByText("milestone").closest("button")!
    expect(oneTime.disabled).toBe(true)
    expect(milestone.disabled).toBe(true)
  })

  it("does not call onChange when disabled", () => {
    const onChange = vi.fn()
    render(
      <PaymentModeToggle value="one_time" onChange={onChange} disabled />,
    )
    fireEvent.click(screen.getByText("milestone"))
    expect(onChange).not.toHaveBeenCalled()
  })

  it("has tablist role on the container", () => {
    render(<PaymentModeToggle value="one_time" onChange={vi.fn()} />)
    expect(screen.getByRole("tablist")).toBeInTheDocument()
  })
})
