import { describe, expect, it } from "vitest"
import { render } from "@testing-library/react"
import { Sparkline } from "../sparkline"

describe("Sparkline", () => {
  it("renders a dashed baseline when fewer than 2 points", () => {
    const { container } = render(<Sparkline values={[]} />)
    expect(container.querySelector("line")).toBeTruthy()
    expect(container.querySelector("path")).toBeNull()
  })

  it("renders a line + area path when given a series", () => {
    const { container } = render(<Sparkline values={[1, 4, 2, 8, 5]} />)
    const paths = container.querySelectorAll("path")
    // expect both the area gradient path AND the stroke path
    expect(paths.length).toBeGreaterThanOrEqual(2)
  })

  it("forwards an aria-label when provided", () => {
    const { container } = render(<Sparkline values={[1, 2, 3]} ariaLabel="Views" />)
    const svg = container.querySelector("svg")
    expect(svg?.getAttribute("aria-label")).toBe("Views")
    expect(svg?.getAttribute("role")).toBe("img")
  })
})
