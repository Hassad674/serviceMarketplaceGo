/**
 * robots.ts tests — PERF-W-04.
 *
 * Asserts the rule set blocks every authenticated path AND points
 * Google at the sitemap URL.
 */

import { describe, it, expect, vi } from "vitest"

vi.mock("@/config/site", () => ({
  siteConfig: { url: "https://example.com/" },
}))

import robots from "../robots"

describe("robots — PERF-W-04", () => {
  it("declares a single user-agent rule with allow / disallow", () => {
    const result = robots()
    expect(Array.isArray(result.rules)).toBe(true)
    const rules = result.rules as Array<{
      userAgent: string
      allow: string
      disallow: string[]
    }>
    expect(rules).toHaveLength(1)
    expect(rules[0].userAgent).toBe("*")
    expect(rules[0].allow).toBe("/")
  })

  it("disallows every authenticated route in PROTECTED_PATHS", () => {
    const result = robots()
    const rules = result.rules as Array<{ disallow: string[] }>
    const blocked = rules[0].disallow
    const expected = [
      "/api/",
      "/dashboard/",
      "/login",
      "/register",
      "/account",
      "/billing",
      "/wallet",
      "/messages",
      "/notifications",
      "/profile",
      "/payment-info",
      "/team",
      "/invoices",
      "/referral",
    ]
    for (const path of expected) {
      expect(
        blocked.some((d) => d === path || d === `${path}/`),
        `${path} should be in the disallow list`,
      ).toBe(true)
    }
  })

  it("does NOT disallow public listing routes", () => {
    const result = robots()
    const rules = result.rules as Array<{ disallow: string[] }>
    const blocked = rules[0].disallow
    expect(blocked).not.toContain("/agencies")
    expect(blocked).not.toContain("/freelancers")
    expect(blocked).not.toContain("/referrers")
    expect(blocked).not.toContain("/opportunities")
  })

  it("points Google at the sitemap", () => {
    const result = robots()
    expect(result.sitemap).toBe("https://example.com/sitemap.xml")
  })
})
