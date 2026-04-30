import { test, expect } from "@playwright/test"

/**
 * SEC-07: brute-force protection on /api/v1/auth/login.
 *
 * The backend locks an email after 5 failed attempts in a 15-minute
 * window for 30 minutes. These tests drive the HTTP API directly
 * (rather than through the form UI) so the assertions stay focused on
 * the security behavior — the form rendering is covered by the
 * existing auth.spec.ts.
 */

const TARGET_EMAIL = `bf-${Date.now()}@playwright.com`

test.describe("SEC-07 login brute-force protection", () => {
  test("locks an email after 5 wrong-password attempts", async ({ request }) => {
    // 5 deliberately wrong-password attempts. The backend may return
    // 401 for each (the email may not exist, that is fine — the
    // counter still bumps) or 401 + 401 + ... + 429 if a previous
    // test in the same window already used the same address. Either
    // way, the 6th attempt MUST be 429.
    for (let i = 0; i < 5; i++) {
      const r = await request.post("/api/v1/auth/login", {
        data: { email: TARGET_EMAIL, password: `wrong-${i}` },
        headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
        failOnStatusCode: false,
      })
      // No assertion on the first 5 — depending on prior state they
      // may already start at 429 if a long-running window is open.
      // We only care about the post-threshold behaviour.
      expect([401, 422, 429]).toContain(r.status())
    }

    const blocked = await request.post("/api/v1/auth/login", {
      data: { email: TARGET_EMAIL, password: "any-password" },
      headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
      failOnStatusCode: false,
    })
    expect(blocked.status()).toBe(429)
    const retry = blocked.headers()["retry-after"]
    expect(retry).toBeDefined()
    expect(Number(retry)).toBeGreaterThan(0)
  })
})
