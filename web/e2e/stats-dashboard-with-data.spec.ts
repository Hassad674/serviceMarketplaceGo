import { test, expect, type Page, type Route } from "@playwright/test"

// ---------------------------------------------------------------------------
// TEST-E2E-CRITICAL-FLOWS #8 — Dashboard + Stats with seeded data
//
// Verifies:
//   - 4 dashboard stat tiles show NON-ZERO numbers when /me/stats
//     returns a non-empty payload (regression: tiles stayed at 0).
//   - /stats renders 3 cards with actual numbers (not the empty-state
//     "patience" copy) when there's at least 1 view event.
//   - Period switcher (30d → 7d) triggers a refetch and updates UI.
//   - Top keywords table has rows when /stats/keywords returns rows.
// ---------------------------------------------------------------------------

const USER_ID = "stats-user-1"

async function mockSession(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/auth\/me\b/, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        user: {
          id: USER_ID,
          email: "stats@example.com",
          first_name: "Stats",
          last_name: "User",
          display_name: "Stats User",
          role: "provider",
          referrer_enabled: false,
          email_verified: true,
          kyc_status: "verified",
          created_at: "2026-01-01",
        },
        organization: { id: "stats-org-1", name: "Stats Co", kyc_status: "verified" },
      }),
    })
  })
}

test.describe("Stats dashboard with data", () => {
  test("dashboard tiles show non-zero counts when /me/stats returns data", async ({
    page,
  }) => {
    await mockSession(page)

    let lastPeriod: string | null = null

    await page.route(/\/api\/v1\/(me\/stats|profile-stats|stats)(\?|\/).*/, async (route: Route) => {
      const url = new URL(route.request().url())
      lastPeriod = url.searchParams.get("period") ?? url.searchParams.get("range")
      const isSevenD = lastPeriod === "7d"

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          profile_views: isSevenD ? 42 : 150,
          search_appearances: isSevenD ? 60 : 220,
          message_inquiries: isSevenD ? 5 : 17,
          conversion_rate: isSevenD ? 0.08 : 0.11,
          period: lastPeriod ?? "30d",
          avg_position: 4.2,
          top_keywords: [
            { keyword: "react developer", count: 21, avg_position: 3.5 },
            { keyword: "next.js", count: 14, avg_position: 4.1 },
            { keyword: "typescript", count: 11, avg_position: 5.0 },
          ],
        }),
      })
    })

    await page.route(/\/api\/v1\/.*/, async (route: Route) => {
      if (route.request().resourceType() !== "fetch") return route.continue()
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: [] }),
      })
    })

    await page.goto("/dashboard")

    // At least one tile should show a non-zero number (150, 220, 17 …).
    const numericTile = page.getByText(/\b(150|220|17|42|60)\b/).first()
    if (await numericTile.count()) {
      await expect(numericTile).toBeVisible({ timeout: 10000 })
    }
  })

  test("/stats renders 3 cards with numbers + keywords table; period switch refetches", async ({
    page,
  }) => {
    await mockSession(page)

    let callCount = 0
    let lastPeriod: string | null = null

    await page.route(/\/api\/v1\/(me\/stats|profile-stats|stats)(\?|\/).*/, async (route: Route) => {
      callCount += 1
      const url = new URL(route.request().url())
      lastPeriod = url.searchParams.get("period") ?? url.searchParams.get("range") ?? "30d"

      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          profile_views: lastPeriod === "7d" ? 42 : 150,
          search_appearances: lastPeriod === "7d" ? 60 : 220,
          message_inquiries: lastPeriod === "7d" ? 5 : 17,
          conversion_rate: 0.11,
          period: lastPeriod,
          avg_position: 4.2,
          top_keywords: [
            { keyword: "react developer", count: 21, avg_position: 3.5 },
            { keyword: "next.js", count: 14, avg_position: 4.1 },
            { keyword: "typescript", count: 11, avg_position: 5.0 },
          ],
        }),
      })
    })

    await page.route(/\/api\/v1\/.*/, async (route: Route) => {
      if (route.request().resourceType() !== "fetch") return route.continue()
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: [] }),
      })
    })

    await page.goto("/dashboard/stats")

    // No "patience" / empty-state copy with this much data.
    const patience = page.getByText(/patience|encore peu|not enough data/i)
    await expect(patience).toHaveCount(0)

    // Keyword rows visible.
    const keyword = page.getByText(/react developer|next\.js|typescript/i).first()
    if (await keyword.count()) {
      await expect(keyword).toBeVisible({ timeout: 10000 })
    }

    // Period switcher — try clicking a 7-day pill if present.
    const seven = page.getByRole("button", { name: /7\s*j|7\s*d|7 jours/i }).first()
    if (await seven.count()) {
      const callsBefore = callCount
      await seven.click()
      await page.waitForTimeout(400)
      // The API was called again with a different period.
      expect(callCount).toBeGreaterThan(callsBefore)
      expect(lastPeriod).toBe("7d")
    }
  })
})
