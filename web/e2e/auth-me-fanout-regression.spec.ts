import { test, expect } from "@playwright/test"
import { clearAuth } from "./helpers/auth"

// PERF-FIX-W-AUTH-ME-FANOUT regression.
//
// Before the fix, opening any authenticated-aware page while logged
// out (e.g. /freelancers/[id]) issued 100-200+ GET /api/v1/auth/me
// requests in under one second from the same browser tab. The burst
// tripped the global IP rate limit which then bricked every other
// API call the page legitimately needed, freezing the page on its
// loading skeleton forever.
//
// Root cause: the session query (`useUser` / `useOrganization` /
// `useSession`) shares a singleton ["session"] queryKey across ~30
// distinct consumers (PublicLayout, PostHogProvider, sidebar/header
// on the logged-in branch, ChatWidget, …). When the first /auth/me
// returned 401 the cache stayed in `{ data: undefined, status: "error" }`,
// and TanStack Query's default `retryOnMount: true` made every
// subsequent observer fire its own /auth/me request. The fix sets
// `retryOnMount: false` on the session query so the 401 verdict
// sticks for the lifetime of the cache; login / register flows
// explicitly invalidate ["session"] to refetch when the cookie is
// issued.
//
// This spec asserts the budget directly: a logged-out navigation to
// a public freelancer profile must not exceed a small constant
// number of /auth/me requests, regardless of how many session
// consumers the page mounts.
test.describe("auth/me fan-out regression", () => {
  test("issues fewer than 5 /auth/me requests on a logged-out /freelancers/[id] navigation", async ({ page }) => {
    // The fan-out budget. Production logs showed 100-200+; the fix
    // brings the count down to 1-2 (one for the first observer, at
    // most one extra for the React-strict-mode double-mount in dev
    // builds). 5 leaves a comfortable margin while still failing
    // loud if a regression slips a refetch storm back in.
    const FANOUT_BUDGET = 5

    let authMeCallCount = 0
    page.on("request", (request) => {
      const url = request.url()
      if (
        url.includes("/api/v1/auth/me") &&
        request.method() === "GET"
      ) {
        authMeCallCount += 1
      }
    })

    await page.goto("/")
    await clearAuth(page)

    // Visit the public freelancers listing first to land on a real
    // freelancer profile id. If the seeded fixture is empty we mark
    // the test as skipped — the assertion is fixture-dependent.
    await page.goto("/freelancers")
    const card = page.locator('a[href*="/freelancers/"]').first()
    if ((await card.count()) === 0) {
      test.skip(true, "No freelancer profile seeded — fixture-dependent assertion")
      return
    }

    // Reset the counter once we're about to navigate to the detail
    // page — the fan-out we care about is the one triggered by the
    // detail page mounting its session consumers, not the list.
    authMeCallCount = 0

    await card.click()
    await page.waitForURL(/\/freelancers\/[^/]+$/, { timeout: 10000 })

    // Give the page time to mount every layout, provider, sidebar
    // shell, chat widget, etc. — any fan-out would happen in the
    // first second after the route commit.
    await page.waitForTimeout(1500)

    expect(authMeCallCount).toBeLessThan(FANOUT_BUDGET)
  })
})
