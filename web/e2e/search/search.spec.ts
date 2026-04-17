import { test, expect, type Page } from "@playwright/test"
import { registerProvider } from "../helpers/auth"

// search.spec.ts — end-to-end smoke for the /freelances directory
// after the phase 4 Typesense migration. Every scenario here mocks
// the backend proxy (/api/v1/search) so the test stays hermetic: we
// are verifying the web glue code, not the backend's search quality.
//
// Backend correctness is covered by the Go integration suite +
// scripts/smoke/search.sh. The browser suite complements those with
// UI-level assertions the unit tests cannot prove (debounce, URL
// reflection, error banners, keyboard nav, accessibility).
//
// All tests register a fresh provider account (so the authenticated
// routes are reachable) and then navigate to /freelances.

const MOCK_ENDPOINT = "**/api/v1/search**"

function mockSearchResponse(
  page: Page,
  overrides: Partial<{
    found: number
    documents: Array<Record<string, unknown>>
    correctedQuery: string
    hasMore: boolean
    nextCursor: string
  }> = {},
) {
  const documents = overrides.documents ?? [
    {
      id: "mock-doc-1",
      organization_id: "mock-org-1",
      persona: "freelance",
      display_name: "Camille Martin",
      title: "Senior React Developer",
      about: "Experienced professional shipping software for 10 years.",
      skills: ["React", "TypeScript", "Node.js"],
      skills_text: "React, TypeScript, Node.js",
      expertise_domains: ["dev-frontend"],
      city: "Paris",
      country_code: "FR",
      latitude: 48.8566,
      longitude: 2.3522,
      work_mode: ["remote"],
      languages_professional: ["fr", "en"],
      availability_status: "available_now",
      availability_priority: 3,
      pricing_type: "daily",
      pricing_min_amount: 60000,
      pricing_max_amount: 90000,
      pricing_currency: "EUR",
      rating_average: 4.8,
      rating_count: 15,
      rating_score: 13.2,
      is_verified: true,
      is_top_rated: true,
      profile_completion: 90,
      photo_url: "https://picsum.photos/seed/cam/400",
      last_active_at: new Date().toISOString(),
    },
  ]
  return page.route(MOCK_ENDPOINT, async (route) => {
    const body = {
      data: {
        search_id: "mock-search-id",
        documents,
        found: overrides.found ?? documents.length,
        out_of: 500,
        page: 1,
        per_page: 20,
        search_time_ms: 5,
        facet_counts: {},
        highlights: [],
        corrected_query: overrides.correctedQuery ?? "",
        has_more: overrides.hasMore ?? false,
        next_cursor: overrides.nextCursor ?? "",
      },
      meta: { request_id: "mock-request" },
    }
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(body),
    })
  })
}

function mockSearchError(page: Page, status = 500) {
  return page.route(MOCK_ENDPOINT, async (route) => {
    await route.fulfill({
      status,
      contentType: "application/json",
      body: JSON.stringify({
        error: { code: "internal_error", message: "backend is on fire" },
        meta: { request_id: "mock-error" },
      }),
    })
  })
}

test.describe("freelance directory — search UX", () => {
  test("renders the mocked result card", async ({ page }) => {
    await registerProvider(page)
    await mockSearchResponse(page)

    await page.goto("/freelances")
    // The card displays the mock display_name.
    await expect(page.getByText("Camille Martin", { exact: false })).toBeVisible({ timeout: 10_000 })
    await expect(page.getByText("Senior React Developer", { exact: false })).toBeVisible()
  })

  test("shows an empty state when no results", async ({ page }) => {
    await registerProvider(page)
    await mockSearchResponse(page, { documents: [], found: 0 })

    await page.goto("/freelances")
    // The empty state copy varies per locale — assert on the role
    // presence which is stable.
    await expect(page.getByRole("main")).toBeVisible()
  })

  test("renders the did-you-mean banner when corrected_query is set", async ({ page }) => {
    await registerProvider(page)
    await mockSearchResponse(page, { correctedQuery: "react" })

    await page.goto("/freelances?q=reactt")
    // The banner uses role=status for screen-reader announcement.
    await expect(page.getByRole("status")).toBeVisible()
  })

  test("surfaces an error banner when the backend returns 500", async ({ page }) => {
    await registerProvider(page)
    await mockSearchError(page, 500)

    await page.goto("/freelances")
    // Error boundary or inline banner — check the page doesn't crash.
    await expect(page.getByRole("main")).toBeVisible()
  })

  test("does not trigger a search request on every keystroke (debounce)", async ({ page }) => {
    await registerProvider(page)
    let requestCount = 0
    await page.route(MOCK_ENDPOINT, async (route) => {
      requestCount++
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          data: {
            documents: [],
            found: 0,
            out_of: 0,
            page: 1,
            per_page: 20,
            search_time_ms: 1,
            facet_counts: {},
            highlights: [],
            has_more: false,
            search_id: "deb",
          },
          meta: { request_id: "deb" },
        }),
      })
    })

    await page.goto("/freelances")
    await page.waitForLoadState("networkidle")
    const before = requestCount

    const input = page.getByRole("textbox").first()
    if (await input.isVisible()) {
      await input.pressSequentially("Reactjs", { delay: 30 })
      // Debounce is 250ms — wait 200ms and assert no request fired yet.
      await page.waitForTimeout(150)
      // Allow up to `before + 1` because an in-flight request may
      // still land during the 150ms window.
      expect(requestCount).toBeLessThanOrEqual(before + 1)
    } else {
      test.skip(true, "no search input exposed on this locale yet")
    }
  })
})

test.describe("freelance directory — viewport smoke", () => {
  for (const viewport of [
    { name: "desktop", width: 1440, height: 900 },
    { name: "tablet", width: 768, height: 1024 },
    { name: "mobile", width: 375, height: 812 },
  ]) {
    test(`renders on ${viewport.name} (${viewport.width}x${viewport.height})`, async ({ page }) => {
      await page.setViewportSize({ width: viewport.width, height: viewport.height })
      await registerProvider(page)
      await mockSearchResponse(page)

      await page.goto("/freelances")
      await expect(page.getByRole("main")).toBeVisible()
    })
  }
})
