import { test, expect, type Page, type Route } from "@playwright/test"

// ---------------------------------------------------------------------------
// /jobs/<id> Candidates panel — "View profile" persona-aware routing.
//
// Regression coverage for the bug where the "View profile" CTA on a
// candidate row hardcoded /freelancers/<applicant_id>, which 404'd
// because applicant_id is the user id (not the org id) AND because
// agency / referrer candidates need their own route prefix.
//
// Strategy: mock every API the /jobs/<id> page calls so the test runs
// without a backend. We assert the rendered <a> href on each persona
// row, which is the smallest contract that proves the bug is fixed.
// Non-goal: visiting the public profile page itself — that is covered
// elsewhere.
// ---------------------------------------------------------------------------

test.describe.configure({ mode: "parallel" })

const JOB_ID = "ad3cc084-79c5-4f7a-9605-38cc463913fa"

const ME_FIXTURE = {
  user: {
    id: "user-enterprise-1",
    email: "ent@test.local",
    role: "enterprise",
    name: "Test Enterprise",
  },
  organization: {
    id: "org-enterprise-1",
    name: "Acme",
    type: "enterprise",
    role: "owner",
  },
  permissions: ["jobs.create", "jobs.list", "messaging.send"],
}

const JOB_FIXTURE = {
  id: JOB_ID,
  creator_id: "user-enterprise-1",
  title: "Need a senior Go engineer",
  description: "Project description goes here.",
  skills: ["go", "postgres"],
  applicant_type: "all",
  budget_type: "one_shot",
  min_budget: 5000,
  max_budget: 10000,
  status: "open",
  created_at: "2026-04-01T00:00:00Z",
  updated_at: "2026-04-01T00:00:00Z",
  is_indefinite: false,
  description_type: "text",
}

// Three candidate rows — one of each persona — using DIFFERENT ids
// for applicant_id (user) and profile.organization_id (org). This is
// the critical fixture: if the link uses applicant_id we land on a 404
// (or the wrong profile); only profile.organization_id resolves.
const APPLICATIONS_FIXTURE = {
  data: [
    {
      application: {
        id: "app-agency-1",
        job_id: JOB_ID,
        applicant_id: "user-agency-NOT-org",
        applicant_kind: "agency",
        message: "We can help.",
        created_at: "2026-04-02T00:00:00Z",
      },
      profile: {
        organization_id: "org-agency-real",
        owner_user_id: "user-agency-NOT-org",
        name: "Acme Agency",
        org_type: "agency",
        title: "",
        photo_url: "",
        referrer_enabled: false,
        average_rating: 0,
        review_count: 0,
        skills: [],
        pricing: [],
        city: "",
        country_code: "",
        languages_professional: [],
        availability_status: "",
        total_earned: 0,
        completed_projects: 0,
      },
    },
    {
      application: {
        id: "app-referrer-1",
        job_id: JOB_ID,
        applicant_id: "user-referrer-NOT-org",
        applicant_kind: "referrer",
        message: "I know the perfect freelance.",
        created_at: "2026-04-02T01:00:00Z",
      },
      profile: {
        organization_id: "org-referrer-real",
        owner_user_id: "user-referrer-NOT-org",
        name: "Bob Referrer",
        org_type: "provider_personal",
        title: "Apporteur d'affaires",
        photo_url: "",
        referrer_enabled: true,
        average_rating: 0,
        review_count: 0,
        skills: [],
        pricing: [],
        city: "",
        country_code: "",
        languages_professional: [],
        availability_status: "",
        total_earned: 0,
        completed_projects: 0,
      },
    },
    {
      application: {
        id: "app-freelance-1",
        job_id: JOB_ID,
        applicant_id: "user-freelance-NOT-org",
        applicant_kind: "freelance",
        message: "I would love to apply.",
        created_at: "2026-04-02T02:00:00Z",
      },
      profile: {
        organization_id: "org-freelance-real",
        owner_user_id: "user-freelance-NOT-org",
        name: "Carol Dev",
        org_type: "provider_personal",
        title: "Senior Go engineer",
        photo_url: "",
        referrer_enabled: false,
        average_rating: 0,
        review_count: 0,
        skills: [],
        pricing: [],
        city: "",
        country_code: "",
        languages_professional: [],
        availability_status: "",
        total_earned: 0,
        completed_projects: 0,
      },
    },
  ],
  next_cursor: "",
  has_more: false,
}

async function mockRoutes(page: Page) {
  await page.route(/\/api\/v1\/auth\/me/, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(ME_FIXTURE),
    })
  })

  await page.route(new RegExp(`/api/v1/jobs/${JOB_ID}\\b(?!/applications)`), async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(JOB_FIXTURE),
    })
  })

  await page.route(/\/api\/v1\/jobs\/.+\/applications/, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(APPLICATIONS_FIXTURE),
    })
  })

  // Catch-all for any /api/v1/* the page tries (notifications, calls,
  // metrics) — return an empty payload so the page keeps rendering.
  await page.route(/\/api\/v1\/.*/, async (route: Route) => {
    if (route.request().resourceType() === "fetch") {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: [] }),
      })
      return
    }
    await route.continue()
  })
}

test.describe("Job candidates · View profile routing", () => {
  test("each persona row links to its persona-specific public route", async ({ page }) => {
    await mockRoutes(page)

    await page.goto(`/jobs/${JOB_ID}`)
    await page.waitForLoadState("networkidle")

    // Switch to the Candidates tab if needed — many designs default to
    // the description tab. We click the first tab whose label includes
    // a digit (the "Candidates (N)" pill).
    const tab = page.getByRole("button", { name: /candidat|candidate/i }).first()
    if (await tab.count()) {
      await tab.click().catch(() => {
        /* fine if tab already active */
      })
    }

    // Each "View profile" link must carry the correct route prefix +
    // the org id, NOT the applicant user id.
    const links = page.getByRole("link", { name: /view profile|voir le profil/i })
    const hrefs = await links.evaluateAll((nodes) =>
      nodes.map((n) => n.getAttribute("href") ?? ""),
    )

    expect(hrefs.length).toBeGreaterThanOrEqual(3)
    expect(hrefs.some((h) => h.includes("/agencies/org-agency-real"))).toBe(true)
    expect(hrefs.some((h) => h.includes("/referrers/org-referrer-real"))).toBe(true)
    expect(hrefs.some((h) => h.includes("/freelancers/org-freelance-real"))).toBe(true)

    // Regression: never the user id.
    for (const h of hrefs) {
      expect(h).not.toContain("user-agency-NOT-org")
      expect(h).not.toContain("user-referrer-NOT-org")
      expect(h).not.toContain("user-freelance-NOT-org")
    }
  })
})
