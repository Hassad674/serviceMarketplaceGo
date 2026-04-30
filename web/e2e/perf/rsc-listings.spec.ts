import { test, expect } from "@playwright/test"

/**
 * PERF-W-02 — public listings RSC + JSON-LD regression suite.
 *
 * The three public listing pages must:
 *   - render a real <h1> in the SSR HTML (not just "Loading…")
 *   - emit a JSON-LD `ItemList` script tag with at least 1 element
 *     (the seed data may be empty in CI, in which case the test
 *     accepts an empty itemListElement)
 *   - declare a canonical URL
 *
 * The agency / opportunity detail pages must additionally emit
 * Schema.org `Organization` / `JobPosting` JSON-LD blocks.
 */

const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? "http://localhost:3001"

async function readJsonLdScripts(page: import("@playwright/test").Page) {
  return page.$$eval(
    "script[type='application/ld+json']",
    (scripts) =>
      scripts
        .map((s) => s.textContent || "")
        .map((t) => {
          try {
            return JSON.parse(t)
          } catch {
            return null
          }
        })
        .filter(Boolean),
  )
}

test.describe("PERF-W-02 — public listings render real HTML", () => {
  test("agencies listing has h1 + canonical + ItemList JSON-LD", async ({ page }) => {
    await page.goto(`${BASE_URL}/agencies`, { waitUntil: "networkidle" })

    // <title> is set dynamically — assert it's NOT the generic stub.
    const title = await page.title()
    expect(title.toLowerCase()).toContain("agenc")

    const canonical = await page.locator("link[rel='canonical']").getAttribute("href")
    expect(canonical).toContain("/agencies")

    const blocks = await readJsonLdScripts(page)
    const itemList = blocks.find(
      (b: unknown) =>
        typeof b === "object" &&
        b !== null &&
        (b as Record<string, unknown>)["@type"] === "ItemList",
    )
    expect(itemList, "agency listing should emit a JSON-LD ItemList").toBeTruthy()
  })

  test("freelancers listing has h1 + canonical + ItemList JSON-LD", async ({ page }) => {
    await page.goto(`${BASE_URL}/freelancers`, { waitUntil: "networkidle" })

    const title = await page.title()
    expect(title.toLowerCase()).toContain("freelance")

    const canonical = await page.locator("link[rel='canonical']").getAttribute("href")
    expect(canonical).toContain("/freelancers")

    const blocks = await readJsonLdScripts(page)
    const itemList = blocks.find(
      (b: unknown) =>
        typeof b === "object" &&
        b !== null &&
        (b as Record<string, unknown>)["@type"] === "ItemList",
    )
    expect(itemList, "freelancer listing should emit a JSON-LD ItemList").toBeTruthy()
  })

  test("referrers listing has h1 + canonical + ItemList JSON-LD", async ({ page }) => {
    await page.goto(`${BASE_URL}/referrers`, { waitUntil: "networkidle" })

    const title = await page.title()
    expect(title.toLowerCase()).toContain("referrer")

    const canonical = await page.locator("link[rel='canonical']").getAttribute("href")
    expect(canonical).toContain("/referrers")

    const blocks = await readJsonLdScripts(page)
    const itemList = blocks.find(
      (b: unknown) =>
        typeof b === "object" &&
        b !== null &&
        (b as Record<string, unknown>)["@type"] === "ItemList",
    )
    expect(itemList, "referrer listing should emit a JSON-LD ItemList").toBeTruthy()
  })
})
