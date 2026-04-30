import { test, expect } from "@playwright/test"

// ---------------------------------------------------------------------------
// CSP violation smoke — depends on Agent A's SecurityHeaders work.
//
// Once Agent A wires the SecurityHeaders middleware (SEC-03) and the
// web/next.config.ts headers, this test asserts that the response
// includes a Content-Security-Policy header that blocks third-party
// scripts. Until then, the test is a stub and skips itself so the suite
// stays green.
//
// To activate after Agent A merges:
//   1. Remove the test.skip line.
//   2. Update the expected CSP string to match the merged middleware.
// ---------------------------------------------------------------------------

test.describe("SEC-03 — CSP violation refusal (depends on Agent A)", () => {
  test.skip(
    true,
    "stub — activate once Agent A merges feat/security-headers-sessions",
  )

  test("CSP header present on / and blocks third-party scripts", async ({ page, request }) => {
    const resp = await request.get("/")
    const csp = resp.headers()["content-security-policy"]
    expect(csp).toBeTruthy()
    expect(csp).toContain("default-src 'self'")
    expect(csp).toContain("script-src 'self'")

    // Page-level smoke: load the home page and check the browser did
    // not surface a CSP violation in console.
    const violations: string[] = []
    page.on("console", (msg) => {
      if (msg.type() === "error" && msg.text().includes("Content Security Policy")) {
        violations.push(msg.text())
      }
    })
    await page.goto("/")
    await page.waitForLoadState("networkidle")
    expect(violations).toHaveLength(0)
  })
})
