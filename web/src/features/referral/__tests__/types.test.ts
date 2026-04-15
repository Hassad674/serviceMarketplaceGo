import { describe, expect, it } from "vitest"

import { formatRatePct, statusTone } from "../types"
import type { ReferralStatus } from "../types"

describe("formatRatePct", () => {
  it("returns the placeholder when the rate is undefined (client pre-activation)", () => {
    expect(formatRatePct(undefined)).toBe("—")
  })

  it("returns an integer percentage with no decimals when the rate is a whole number", () => {
    expect(formatRatePct(5)).toBe("5%")
    expect(formatRatePct(0)).toBe("0%")
  })

  it("returns two decimals when the rate is fractional", () => {
    expect(formatRatePct(3.5)).toBe("3.50%")
    expect(formatRatePct(7.25)).toBe("7.25%")
  })
})

describe("statusTone", () => {
  const pending: ReferralStatus[] = [
    "pending_provider",
    "pending_referrer",
    "pending_client",
  ]
  const failure: ReferralStatus[] = ["rejected", "expired", "cancelled"]

  it("groups pending_* statuses under the pending tone", () => {
    for (const s of pending) expect(statusTone(s)).toBe("pending")
  })

  it("returns active for the active status", () => {
    expect(statusTone("active")).toBe("active")
  })

  it("groups rejection-like terminal statuses as terminal-failure", () => {
    for (const s of failure) expect(statusTone(s)).toBe("terminal-failure")
  })

  it("returns terminal-success for terminated", () => {
    expect(statusTone("terminated")).toBe("terminal-success")
  })
})
