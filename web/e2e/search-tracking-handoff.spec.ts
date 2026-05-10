import { test, expect } from "@playwright/test"

// search-tracking-handoff.spec.ts — locks in the search → profile
// click handoff contract. Clicking a result on /freelancers (or any
// public listing) must navigate to the destination profile with
// `?q=<lowercased trimmed query>&pos=<1-based rank>`. The backend's
// public-profile tracking middleware reads those query params to
// populate /me/stats/keywords + visibility records.
//
// Public listings are anonymous-accessible — no login fixture needed.

test.describe("Search result tracking handoff", () => {
  test("clicking a card preserves no q/pos in the unscoped browse", async ({ page }) => {
    await page.goto("/en/freelancers")
    // Wait for at least one card to render. The selector is the link
    // wrapping each <article role=article>.
    const firstLink = page.locator("article a").first()
    await expect(firstLink).toBeVisible({ timeout: 15000 })
    const href = await firstLink.getAttribute("href")
    expect(href).toBeTruthy()
    // No query was typed → no `?q=`. Position MAY still be present
    // when the card is rendered with an index, but the query param
    // must be absent.
    expect(href ?? "").not.toContain("q=")
  })

  test("clicking a result after typing a query appends ?q=&pos=", async ({ page }) => {
    await page.goto("/en/freelancers")
    // Type a query and submit
    const input = page.getByRole("searchbox")
    await input.fill("Developer")
    await input.press("Enter")
    // Wait for results to refresh
    const firstLink = page.locator("article a").first()
    await expect(firstLink).toBeVisible({ timeout: 15000 })
    const href = await firstLink.getAttribute("href")
    expect(href).toBeTruthy()
    expect(href).toContain("q=developer")
    expect(href).toContain("pos=1")
  })
})
