import { test, expect } from "@playwright/test"

/**
 * PERF-W-04 — sitemap.xml + robots.txt regression suite.
 *
 * Asserts the routes:
 *   - GET /sitemap.xml returns XML with the static URLs
 *   - GET /robots.txt returns the disallow list
 */

const BASE_URL = process.env.PLAYWRIGHT_BASE_URL ?? "http://localhost:3001"

test.describe("PERF-W-04 — sitemap + robots", () => {
  test("/sitemap.xml exposes the static SEO surfaces", async ({ request }) => {
    const res = await request.get(`${BASE_URL}/sitemap.xml`)
    expect(res.status()).toBe(200)
    const ct = res.headers()["content-type"] || ""
    expect(ct.toLowerCase()).toContain("xml")
    const xml = await res.text()
    // The static block must always appear.
    for (const path of [
      "/agencies",
      "/freelancers",
      "/referrers",
      "/opportunities",
    ]) {
      expect(xml).toContain(path)
    }
    // Must declare urlset.
    expect(xml).toContain("<urlset")
  })

  test("/robots.txt declares the protected-path disallow list", async ({ request }) => {
    const res = await request.get(`${BASE_URL}/robots.txt`)
    expect(res.status()).toBe(200)
    const body = await res.text()
    expect(body).toContain("User-agent: *")
    for (const path of [
      "/dashboard/",
      "/api/",
      "/login",
      "/register",
      "/wallet",
      "/messages",
    ]) {
      expect(body).toContain(`Disallow: ${path}`)
    }
    // Sitemap pointer.
    expect(body).toContain("Sitemap:")
  })
})
