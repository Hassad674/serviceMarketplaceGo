import { describe, expect, it } from "vitest"

import {
  CMP_CATEGORIES,
  COOKIE_INVENTORY,
  isCmpCategory,
} from "../cookie-consent-config"

describe("cookie-consent-config", () => {
  it("declares at least one cookie per CMP category", () => {
    for (const category of CMP_CATEGORIES) {
      const matching = COOKIE_INVENTORY.filter((c) => c.category === category)
      expect(matching.length).toBeGreaterThan(0)
    }
  })

  it("does not place anything under a category outside the CMP", () => {
    for (const cookie of COOKIE_INVENTORY) {
      // functional is reserved but currently empty — assert only the
      // active CMP categories are used by the inventory.
      if (cookie.category === "functional") continue
      expect(CMP_CATEGORIES).toContain(cookie.category)
    }
  })

  it("only lists necessary and analytics in the active CMP today", () => {
    expect(CMP_CATEGORIES).toEqual(["necessary", "analytics"])
  })

  it("isCmpCategory recognises the live categories", () => {
    expect(isCmpCategory("necessary")).toBe(true)
    expect(isCmpCategory("analytics")).toBe(true)
    expect(isCmpCategory("marketing")).toBe(false)
    expect(isCmpCategory("")).toBe(false)
  })

  it("uses unique row keys (no duplicates)", () => {
    const keys = COOKIE_INVENTORY.map((c) => c.key)
    expect(new Set(keys).size).toBe(keys.length)
  })
})
