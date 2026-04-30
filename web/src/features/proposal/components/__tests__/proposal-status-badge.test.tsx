import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { StatusBadge, DetailSkeleton } from "../proposal-status-badge"

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string) => key,
}))

describe("StatusBadge", () => {
  it("renders pending status", () => {
    render(<StatusBadge status="pending" />)
    expect(screen.getByText("pending")).toBeInTheDocument()
  })

  it("renders accepted status", () => {
    render(<StatusBadge status="accepted" />)
    expect(screen.getByText("accepted")).toBeInTheDocument()
  })

  it("renders declined status", () => {
    render(<StatusBadge status="declined" />)
    expect(screen.getByText("declined")).toBeInTheDocument()
  })

  it("renders withdrawn status", () => {
    render(<StatusBadge status="withdrawn" />)
    expect(screen.getByText("withdrawn")).toBeInTheDocument()
  })

  it("renders paid status", () => {
    render(<StatusBadge status="paid" />)
    expect(screen.getByText("paid")).toBeInTheDocument()
  })

  it("renders active status", () => {
    render(<StatusBadge status="active" />)
    expect(screen.getByText("active")).toBeInTheDocument()
  })

  it("renders completion_requested status", () => {
    render(<StatusBadge status="completion_requested" />)
    expect(screen.getByText("completionRequested")).toBeInTheDocument()
  })

  it("renders completed status", () => {
    render(<StatusBadge status="completed" />)
    expect(screen.getByText("completed")).toBeInTheDocument()
  })

  it("falls back to pending config for unknown status", () => {
    // @ts-expect-error testing unknown status fallback
    render(<StatusBadge status="unknown_status" />)
    expect(screen.getByText("pending")).toBeInTheDocument()
  })

  it("applies amber colour for pending", () => {
    const { container } = render(<StatusBadge status="pending" />)
    const span = container.querySelector("span")
    expect(span?.className).toMatch(/amber/)
  })

  it("applies green colour for accepted", () => {
    const { container } = render(<StatusBadge status="accepted" />)
    const span = container.querySelector("span")
    expect(span?.className).toMatch(/green/)
  })

  it("applies red colour for declined", () => {
    const { container } = render(<StatusBadge status="declined" />)
    const span = container.querySelector("span")
    expect(span?.className).toMatch(/red/)
  })
})

describe("DetailSkeleton", () => {
  it("renders the skeleton scaffold", () => {
    const { container } = render(<DetailSkeleton />)
    expect(container.querySelectorAll(".animate-pulse").length).toBeGreaterThan(
      3,
    )
  })
})
