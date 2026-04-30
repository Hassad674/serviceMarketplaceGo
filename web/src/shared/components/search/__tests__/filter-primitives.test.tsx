import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import {
  CheckboxRow,
  NumberInput,
  PillButton,
  SectionShell,
  toggle,
} from "../filter-primitives"

describe("toggle helper", () => {
  it("adds an absent value", () => {
    expect(toggle([1, 2], 3)).toEqual([1, 2, 3])
  })
  it("removes a present value", () => {
    expect(toggle([1, 2, 3], 2)).toEqual([1, 3])
  })
  it("preserves identity for an empty list", () => {
    expect(toggle([] as number[], 1)).toEqual([1])
  })
})

describe("SectionShell", () => {
  it("renders the title as an h3", () => {
    render(
      <SectionShell title="My section">
        <span>child</span>
      </SectionShell>,
    )
    expect(screen.getByRole("heading", { name: "My section" })).toBeInTheDocument()
    expect(screen.getByText("child")).toBeInTheDocument()
  })
})

describe("PillButton", () => {
  it("renders the label", () => {
    render(<PillButton label="Pill" selected={false} onClick={() => {}} />)
    expect(screen.getByRole("button", { name: "Pill" })).toBeInTheDocument()
  })

  it("aria-pressed mirrors selected", () => {
    const { rerender } = render(
      <PillButton label="X" selected={false} onClick={() => {}} />,
    )
    expect(screen.getByRole("button")).toHaveAttribute("aria-pressed", "false")
    rerender(<PillButton label="X" selected={true} onClick={() => {}} />)
    expect(screen.getByRole("button")).toHaveAttribute("aria-pressed", "true")
  })

  it("calls onClick", () => {
    const onClick = vi.fn()
    render(<PillButton label="X" selected={false} onClick={onClick} />)
    fireEvent.click(screen.getByRole("button"))
    expect(onClick).toHaveBeenCalledOnce()
  })
})

describe("NumberInput", () => {
  it("renders the placeholder + aria-label", () => {
    render(
      <NumberInput
        value={null}
        onChange={() => {}}
        placeholder="hi"
        ariaLabel="aria"
      />,
    )
    expect(screen.getByLabelText("aria")).toBeInTheDocument()
  })

  it("returns null on empty input", () => {
    const onChange = vi.fn()
    // Start with a value so changing back to "" actually triggers a
    // change event (changing "" → "" is a no-op in React).
    render(
      <NumberInput
        value={42}
        onChange={onChange}
        placeholder="x"
        ariaLabel="aria"
      />,
    )
    fireEvent.change(screen.getByLabelText("aria"), { target: { value: "" } })
    expect(onChange).toHaveBeenCalledWith(null)
  })

  it("parses a valid number", () => {
    const onChange = vi.fn()
    render(
      <NumberInput
        value={null}
        onChange={onChange}
        placeholder="x"
        ariaLabel="aria"
      />,
    )
    fireEvent.change(screen.getByLabelText("aria"), { target: { value: "42" } })
    expect(onChange).toHaveBeenCalledWith(42)
  })
})

describe("CheckboxRow", () => {
  it("renders the label and checked state", () => {
    render(<CheckboxRow checked label="check me" onChange={() => {}} />)
    expect(screen.getByText("check me")).toBeInTheDocument()
    expect(screen.getByRole("checkbox")).toBeChecked()
  })

  it("calls onChange when toggled", () => {
    const onChange = vi.fn()
    render(<CheckboxRow checked={false} label="x" onChange={onChange} />)
    fireEvent.click(screen.getByRole("checkbox"))
    expect(onChange).toHaveBeenCalledOnce()
  })
})
