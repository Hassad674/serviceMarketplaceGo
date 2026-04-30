import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { StarRating } from "../star-rating"

describe("StarRating", () => {
  it("renders 5 stars", () => {
    render(<StarRating rating={3} onRatingChange={vi.fn()} />)
    expect(screen.getAllByRole("radio")).toHaveLength(5)
  })

  it("renders the label when provided", () => {
    render(<StarRating rating={3} label="Quality" />)
    expect(screen.getByText("Quality")).toBeInTheDocument()
  })

  it("renders no label when not provided", () => {
    render(<StarRating rating={3} />)
    expect(screen.queryByText(/quality/i)).toBeNull()
  })

  it("calls onRatingChange with the star number when clicked", () => {
    const onChange = vi.fn()
    render(<StarRating rating={0} onRatingChange={onChange} />)
    fireEvent.click(screen.getByLabelText("4 stars"))
    expect(onChange).toHaveBeenCalledWith(4)
  })

  it("treats 1 differently in aria-label (singular)", () => {
    render(<StarRating rating={0} onRatingChange={vi.fn()} />)
    expect(screen.getByLabelText("1 star")).toBeInTheDocument()
  })

  it("uses radiogroup role when interactive", () => {
    render(<StarRating rating={0} onRatingChange={vi.fn()} />)
    const group = document.querySelector("[role='radiogroup']")
    expect(group).not.toBeNull()
  })

  it("sets aria-checked on the active star", () => {
    render(<StarRating rating={3} onRatingChange={vi.fn()} />)
    expect(screen.getByLabelText("3 stars").getAttribute("aria-checked")).toBe("true")
    expect(screen.getByLabelText("4 stars").getAttribute("aria-checked")).toBe("false")
  })

  it("renders read-only stars without buttons being interactive", () => {
    const onChange = vi.fn()
    render(<StarRating rating={3} onRatingChange={onChange} readOnly />)
    fireEvent.click(screen.getByLabelText("4 stars"))
    expect(onChange).not.toHaveBeenCalled()
  })

  it("disables buttons when readOnly", () => {
    const { container } = render(<StarRating rating={3} readOnly />)
    const buttons = container.querySelectorAll("button")
    expect(buttons).toHaveLength(5)
    buttons.forEach((btn) => {
      expect((btn as HTMLButtonElement).disabled).toBe(true)
    })
  })

  it("highlights stars up to the current rating", () => {
    const { container } = render(<StarRating rating={3} />)
    const filled = container.querySelectorAll(".fill-amber-400")
    expect(filled.length).toBe(3)
  })

  it("highlights all 5 stars when rating=5", () => {
    const { container } = render(<StarRating rating={5} />)
    const filled = container.querySelectorAll(".fill-amber-400")
    expect(filled.length).toBe(5)
  })

  it("respects size prop (lg)", () => {
    const { container } = render(<StarRating rating={3} size="lg" />)
    expect(container.querySelectorAll(".h-6.w-6").length).toBe(5)
  })

  it("respects size prop (sm)", () => {
    const { container } = render(<StarRating rating={3} size="sm" />)
    expect(container.querySelectorAll(".h-4.w-4").length).toBe(5)
  })

  it("hovers preview the rating temporarily", () => {
    const { container } = render(<StarRating rating={2} onRatingChange={vi.fn()} />)
    const fourthStar = screen.getByLabelText("4 stars")
    fireEvent.mouseEnter(fourthStar)
    const filledAfterHover = container.querySelectorAll(".fill-amber-400")
    expect(filledAfterHover.length).toBe(4)
    fireEvent.mouseLeave(fourthStar.parentElement!)
    const filledAfterLeave = container.querySelectorAll(".fill-amber-400")
    expect(filledAfterLeave.length).toBe(2)
  })

  it("does not preview hover when readOnly", () => {
    const { container } = render(<StarRating rating={2} readOnly />)
    const fourthStar = screen.getByLabelText("4 stars")
    fireEvent.mouseEnter(fourthStar)
    const filled = container.querySelectorAll(".fill-amber-400")
    expect(filled.length).toBe(2) // unchanged
  })
})
