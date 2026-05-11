import { test, expect } from "@playwright/test"
import { registerProvider } from "./helpers/auth"

// stats-page-with-data.spec.ts — Bug 2 regression coverage. Asserts
// that when the backend returns non-zero total_views and
// search_appearances, the unit counts render in the metric strip and
// the legacy "Données insuffisantes — patiente ~7 jours" copy does
// NOT appear anywhere on the page. The patience copy remains
// acceptable for the avg_search_position card only when below the
// statistical-significance threshold (handled in unit tests).

test.describe("/stats — unit counts always render", () => {
  test("renders 150 / 75 / 4 spots with no global 'not enough data' banner", async ({
    page,
  }) => {
    await registerProvider(page)

    // Intercept the visibility stats endpoint and return a populated
    // payload covering all three unit metrics.
    await page.route("**/api/v1/me/stats/visibility**", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            organization_id: "org-e2e",
            period_days: 30,
            total_views: 150,
            unique_viewers: 90,
            search_appearances: 75,
            avg_search_position: 3.5,
            series: [
              { date: "2026-04-10T00:00:00Z", count: 12 },
              { date: "2026-04-11T00:00:00Z", count: 9 },
              { date: "2026-04-12T00:00:00Z", count: 14 },
            ],
          },
        }),
      }),
    )
    await page.route("**/api/v1/me/stats/keywords**", (route) =>
      route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: [] }),
      }),
    )

    await page.goto("/en/stats")
    const strip = page.getByTestId("stats-metric-strip")
    await expect(strip).toBeVisible({ timeout: 15000 })

    // Unit counts surface verbatim in the strip.
    await expect(strip).toContainText("150")
    await expect(strip).toContainText("75")

    // Position rendered via plural unit ("4 spots" for en locale).
    await expect(strip).toContainText(/4\s+spot/i)

    // The legacy "not enough data" copy MUST NOT appear at the page
    // level (it leaked from the position-only logic to all cards
    // before this fix).
    await expect(
      page.getByText(
        /Not enough data — give your profile a few days to build up/i,
      ),
    ).toHaveCount(0)
    await expect(
      page.getByText(
        /Données insuffisantes — patiente pendant que ton profil/i,
      ),
    ).toHaveCount(0)
  })
})
