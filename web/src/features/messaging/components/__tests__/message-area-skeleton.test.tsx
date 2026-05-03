/**
 * message-area-skeleton.test.tsx
 *
 * Smoke tests for the loading skeleton rendered while messages fetch.
 * The component is purely visual but we lock its DOM contract so the
 * F.3.2 sweep cannot delete or rewire it without flagging the change.
 */
import { describe, it, expect } from "vitest"
import { render } from "@testing-library/react"
import { MessageAreaSkeleton } from "../message-area-skeleton"

describe("MessageAreaSkeleton", () => {
  it("renders 5 placeholder bubbles", () => {
    const { container } = render(<MessageAreaSkeleton />)
    const bubbles = container.querySelectorAll(".animate-pulse")
    expect(bubbles).toHaveLength(5)
  })

  it("alternates left/right alignment to mimic a real conversation", () => {
    const { container } = render(<MessageAreaSkeleton />)
    const rows = container.querySelectorAll('[class*="justify-"]')
    // Row 1 = left (justify-start), row 2 = right (justify-end), …
    expect(rows[0].className).toContain("justify-start")
    expect(rows[1].className).toContain("justify-end")
  })

  it("uses pulsing rose tint for outgoing rows and gray for incoming", () => {
    const { container } = render(<MessageAreaSkeleton />)
    const bubbles = container.querySelectorAll(".animate-pulse")
    // Outgoing bubbles (even indices) carry rose-200; incoming (odd) gray.
    expect(bubbles[1].className).toContain("rose-200")
    expect(bubbles[0].className).toContain("gray-200")
  })

  it("returns a stable render that does not crash without props", () => {
    expect(() => render(<MessageAreaSkeleton />)).not.toThrow()
  })
})
