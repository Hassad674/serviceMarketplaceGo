import { describe, expect, it } from "vitest"
import { render } from "@testing-library/react"

import { Portrait, PORTRAIT_PALETTE_COUNT } from "../portrait"

describe("Portrait", () => {
  it("renders an SVG with the expected silhouette regardless of id", () => {
    const { container } = render(<Portrait id={0} />)
    const svg = container.querySelector("svg")
    expect(svg).not.toBeNull()
    // 4 silhouette parts: neck rect, shoulders path, head ellipse, hair path.
    expect(container.querySelectorAll("svg > *")).toHaveLength(4)
  })

  it("cycles deterministically through palettes via id % count", () => {
    const ids = [0, 6, 12, 18]
    const renderings = ids.map((id) => {
      const { container } = render(<Portrait id={id} />)
      const wrapper = container.firstElementChild as HTMLElement
      return wrapper.style.background
    })
    // Same palette for ids that share a residue.
    expect(new Set(renderings).size).toBe(1)
  })

  it("handles negative ids by wrapping around", () => {
    const { container: a } = render(<Portrait id={-1} />)
    const { container: b } = render(<Portrait id={PORTRAIT_PALETTE_COUNT - 1} />)
    const aBg = (a.firstElementChild as HTMLElement).style.background
    const bBg = (b.firstElementChild as HTMLElement).style.background
    expect(aBg).toBe(bBg)
  })

  it("respects the size prop on both wrapper and svg", () => {
    const { container } = render(<Portrait id={0} size={96} />)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.style.width).toBe("96px")
    expect(wrapper.style.height).toBe("96px")
    const svg = container.querySelector("svg")!
    expect(svg.getAttribute("width")).toBe("96")
    expect(svg.getAttribute("height")).toBe("96")
  })

  it("defaults to full radius (round avatar)", () => {
    const { container } = render(<Portrait id={0} />)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.style.borderRadius).toBe("9999px")
  })

  it("accepts a custom px radius for square-ish avatars", () => {
    const { container } = render(<Portrait id={0} rounded={14} />)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.style.borderRadius).toBe("14px")
  })

  it("forwards aria-label for accessibility (defaults to French)", () => {
    const { container } = render(<Portrait id={0} alt="Photo de Élise" />)
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.getAttribute("role")).toBe("img")
    expect(wrapper.getAttribute("aria-label")).toBe("Photo de Élise")
  })
})
