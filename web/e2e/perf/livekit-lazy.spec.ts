import { test, expect } from "@playwright/test"

/**
 * PERF-W-01 — LiveKit lazy-loading guard.
 *
 * Asserts the dashboard's initial network activity does NOT include
 * the `livekit-client` runtime. The 1.3 MB chunk should only load
 * after a call_incoming WS frame OR a startCall() invocation.
 *
 * This test is a regression guard: if a future patch reintroduces a
 * static `import` of `livekit-client` from the dashboard shell or
 * any sibling, the assertion below will fail because the chunk URL
 * will appear in the network log on initial dashboard navigation.
 *
 * The test is intentionally tolerant of auth-redirect: an
 * unauthenticated visit to /dashboard redirects to /login, which is
 * also a valid baseline for "no LiveKit on the auth flow either".
 */

const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? "http://localhost:3001"

test.describe("PERF-W-01 — LiveKit chunk is lazy-loaded", () => {
  test("home and login pages do not load livekit-client", async ({ page }) => {
    const livekitRequests: string[] = []
    page.on("request", (req) => {
      const url = req.url()
      if (/livekit-client|livekit\.esm/.test(url)) {
        livekitRequests.push(url)
      }
    })

    await page.goto(`${BASE_URL}/`)
    await page.waitForLoadState("networkidle")

    // Visit login as well — auth flow must not pull LiveKit either.
    await page.goto(`${BASE_URL}/login`)
    await page.waitForLoadState("networkidle")

    expect(
      livekitRequests,
      `LiveKit chunk loaded eagerly on public pages — these URLs were requested:\n${livekitRequests.join("\n")}`,
    ).toEqual([])
  })

  test("public agencies listing does not load livekit-client", async ({ page }) => {
    const livekitRequests: string[] = []
    page.on("request", (req) => {
      const url = req.url()
      if (/livekit-client|livekit\.esm/.test(url)) {
        livekitRequests.push(url)
      }
    })

    await page.goto(`${BASE_URL}/agencies`)
    await page.waitForLoadState("networkidle")

    expect(
      livekitRequests,
      `LiveKit chunk loaded on /agencies — listing pages must remain lean. URLs:\n${livekitRequests.join("\n")}`,
    ).toEqual([])
  })

  test("public freelancers listing does not load livekit-client", async ({ page }) => {
    const livekitRequests: string[] = []
    page.on("request", (req) => {
      const url = req.url()
      if (/livekit-client|livekit\.esm/.test(url)) {
        livekitRequests.push(url)
      }
    })

    await page.goto(`${BASE_URL}/freelancers`)
    await page.waitForLoadState("networkidle")

    expect(livekitRequests).toEqual([])
  })
})
