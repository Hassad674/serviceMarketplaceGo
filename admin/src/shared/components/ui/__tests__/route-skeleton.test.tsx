import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { RouteSkeleton } from "../route-skeleton"

describe("RouteSkeleton — ADMIN-PERF-01", () => {
  it("renders an accessible status region", () => {
    render(<RouteSkeleton />)
    const status = screen.getByRole("status")
    expect(status).toBeInTheDocument()
    expect(status.getAttribute("aria-live")).toBe("polite")
    expect(status.getAttribute("aria-label")).toBe("Loading admin section")
  })

  it("contains an sr-only fallback text", () => {
    render(<RouteSkeleton />)
    expect(screen.getByText(/Loading/i)).toBeInTheDocument()
  })

  it("renders multiple skeleton blocks for header + filters + table rows", () => {
    const { container } = render(<RouteSkeleton />)
    const skeletons = container.querySelectorAll(".animate-shimmer")
    // Header (1) + filters (2) + 8 table rows = 11 skeleton bars.
    expect(skeletons.length).toBeGreaterThanOrEqual(10)
  })
})
