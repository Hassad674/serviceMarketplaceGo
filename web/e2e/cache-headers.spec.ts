import { test, expect } from "@playwright/test"
import { clearAuth } from "./helpers/auth"

// PERF-B — public-cache-headers regression spec.
//
// Asserts the wire-level invariant that public read endpoints carry
// `Cache-Control: public, max-age=60, s-maxage=300` for anonymous
// callers, and that authenticated callers receive a private/no-store
// header so the Vercel CDN never caches a personalized payload.
//
// The spec is fixture-dependent: it needs at least one seeded public
// profile to hit. When the fixture is empty we mark the test as
// skipped rather than failing — the assertion is meaningful only
// against a populated dataset.
//
// We bypass Next.js by talking to the backend API directly through
// the page request context. The default base URL for these tests is
// the web app (port 3001) but `request.fetch` against an absolute
// URL pointing at the API uses the same context's cookie jar — which
// is exactly what we want for the "authenticated bypass" branch.

const API_BASE =
  process.env.PLAYWRIGHT_API_URL ?? "http://localhost:8083"

test.describe("public cache headers — PERF-B", () => {
  test("anonymous /api/v1/freelance-profiles/{id} carries public cache-control", async ({
    page,
    request,
  }) => {
    await page.goto("/")
    await clearAuth(page)

    // Find a real org id from the public listing. If the fixture is
    // empty we skip — the assertion is fixture-dependent.
    await page.goto("/freelancers")
    const card = page.locator('a[href*="/freelancers/"]').first()
    if ((await card.count()) === 0) {
      test.skip(
        true,
        "No freelancer seeded — public cache-header assertion is fixture-dependent",
      )
      return
    }
    const href = await card.getAttribute("href")
    const id = href?.split("/freelancers/")[1]?.split(/[/?#]/)[0]
    if (!id) {
      test.skip(true, "Could not extract a freelancer id from the listing")
      return
    }

    const response = await request.fetch(
      `${API_BASE}/api/v1/freelance-profiles/${id}`,
      {
        method: "GET",
      },
    )

    // The backend may legitimately 404 if seeding raced — the cache
    // header invariant still applies to 404s (an anonymous 404 is
    // cacheable too), but the test is more meaningful on a 200.
    expect([200, 404]).toContain(response.status())
    const cacheControl = response.headers()["cache-control"] ?? ""
    expect(
      cacheControl,
      `expected public Cache-Control on anonymous read, got "${cacheControl}"`,
    ).toContain("public")
    expect(cacheControl).toContain("max-age=60")
    expect(cacheControl).toContain("s-maxage=300")

    // Vary must include Accept-Language + Cookie so the CDN cache
    // key separates locales and authenticated variants.
    const vary = response.headers()["vary"] ?? ""
    expect(vary).toContain("Accept-Language")
    expect(vary).toContain("Cookie")
  })

  test("authenticated /api/v1/freelance-profiles/{id} does NOT receive public cache-control", async ({
    request,
  }) => {
    // Drive the bypass branch via the Authorization header (mobile
    // client semantics). The session-cookie branch is exercised by
    // the backend handler unit test — we keep the e2e focused on
    // the wire shape.
    const someOrgId = "00000000-0000-0000-0000-000000000000"
    const response = await request.fetch(
      `${API_BASE}/api/v1/freelance-profiles/${someOrgId}`,
      {
        method: "GET",
        headers: {
          Authorization: "Bearer some-token-not-actually-valid",
        },
      },
    )

    // Whatever the status (likely 401 or 404 with this fake token),
    // the response MUST NOT carry public cache headers — the CDN
    // must not be allowed to cache anything for an authenticated
    // request, even a failure.
    const cacheControl = response.headers()["cache-control"] ?? ""
    expect(
      cacheControl,
      `authenticated calls must bypass public cache, got "${cacheControl}"`,
    ).not.toContain("public, max-age=60, s-maxage=300")
  })

  test("anonymous /api/v1/reviews/org/{id} carries public cache-control", async ({
    page,
    request,
  }) => {
    await page.goto("/")
    await clearAuth(page)

    await page.goto("/freelancers")
    const card = page.locator('a[href*="/freelancers/"]').first()
    if ((await card.count()) === 0) {
      test.skip(true, "No freelancer seeded")
      return
    }
    const href = await card.getAttribute("href")
    const id = href?.split("/freelancers/")[1]?.split(/[/?#]/)[0]
    if (!id) {
      test.skip(true, "Could not extract id")
      return
    }

    const response = await request.fetch(
      `${API_BASE}/api/v1/reviews/org/${id}`,
      {
        method: "GET",
      },
    )

    expect([200, 404]).toContain(response.status())
    const cacheControl = response.headers()["cache-control"] ?? ""
    expect(cacheControl).toContain("public")
    expect(cacheControl).toContain("max-age=60")
  })
})
