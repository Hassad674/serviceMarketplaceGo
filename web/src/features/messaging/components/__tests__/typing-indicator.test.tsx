import { describe, it, expect, vi } from "vitest"
import { render, screen } from "@testing-library/react"
import { TypingIndicator } from "../typing-indicator"

// Mock next-intl
vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, params?: Record<string, string>) => {
    if (key === "typing" && params?.name) {
      return `${params.name} is typing`
    }
    return key
  },
}))

describe("TypingIndicator", () => {
  it("renders with user name", () => {
    render(<TypingIndicator userName="Alice" />)

    expect(screen.getByText("Alice is typing")).toBeDefined()
  })

  it("renders animated dots", () => {
    const { container } = render(<TypingIndicator userName="Bob" />)

    const dots = container.querySelectorAll(".animate-bounce")
    expect(dots.length).toBe(3)
  })

  it("dots have staggered animation delays", () => {
    const { container } = render(<TypingIndicator userName="Charlie" />)

    const dots = container.querySelectorAll(".animate-bounce")
    // Check that different delay classes are present
    const delays = Array.from(dots).map((d) => d.className)
    expect(delays.some((d) => d.includes("[animation-delay:0ms]"))).toBe(true)
    expect(delays.some((d) => d.includes("[animation-delay:150ms]"))).toBe(true)
    expect(delays.some((d) => d.includes("[animation-delay:300ms]"))).toBe(true)
  })
})
