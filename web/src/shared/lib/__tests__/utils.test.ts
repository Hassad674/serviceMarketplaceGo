import { describe, it, expect } from "vitest"
import { cn, formatDate, formatCurrency } from "../utils"

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("px-2", "py-1")).toBe("px-2 py-1")
  })

  it("handles conditional classes", () => {
    expect(cn("px-2", false && "hidden")).toBe("px-2")
  })

  it("deduplicates tailwind classes", () => {
    expect(cn("px-2", "px-4")).toBe("px-4")
  })
})

describe("formatCurrency", () => {
  it("formats euros in French locale", () => {
    const result = formatCurrency(1234.56)
    expect(result).toContain("1")
    expect(result).toContain("234")
    expect(result).toContain("\u20ac")
  })

  it("handles zero", () => {
    const result = formatCurrency(0)
    expect(result).toContain("0")
    expect(result).toContain("\u20ac")
  })
})

describe("formatDate", () => {
  it("formats date in French locale", () => {
    const result = formatDate("2026-03-16")
    expect(result).toContain("2026")
  })
})
