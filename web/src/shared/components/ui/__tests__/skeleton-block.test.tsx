import { describe, it, expect } from "vitest"
import { render } from "@testing-library/react"
import { SkeletonBlock } from "../skeleton-block"

describe("SkeletonBlock", () => {
  it("renders an aria-hidden div with the shimmer overlay", () => {
    const { container } = render(<SkeletonBlock />)
    const root = container.firstElementChild
    expect(root).toBeTruthy()
    expect(root?.getAttribute("aria-hidden")).toBe("true")
    expect(root?.querySelector(".animate-shimmer")).toBeTruthy()
  })

  it("merges a custom className", () => {
    const { container } = render(<SkeletonBlock className="h-9 w-1/2" />)
    const root = container.firstElementChild as HTMLElement
    expect(root.className).toContain("h-9")
    expect(root.className).toContain("w-1/2")
  })
})
