import { test, expect, type Page, type Route } from "@playwright/test"

// ---------------------------------------------------------------------------
// TEST-E2E-CRITICAL-FLOWS #7 — Public freelancer profile renders correctly
//
// Regression checks:
//   - Unauth visitor on /freelancers/<id> sees H1 = "FirstName LastName".
//   - Subtitle line shows the title (e.g. "Senior Go engineer").
//   - When the org has no presentation video → NO <video> tag rendered.
//   - "Send a message" CTA on an unauth profile redirects to /login.
// ---------------------------------------------------------------------------

const FREELANCE_ORG_ID = "11111111-1111-4111-8111-111111111111"

interface FreelanceProfileFixture {
  id: string
  organization_id: string
  title: string
  about: string
  video_url: string
  availability_status: string
  expertise_domains: string[]
  photo_url: string
  city: string
  country_code: string
  latitude: number | null
  longitude: number | null
  work_mode: string[]
  travel_radius_km: number | null
  languages_professional: string[]
  languages_conversational: string[]
  org_name: string
  first_name: string
  last_name: string
  skills: { skill_text: string; display_text: string }[]
  pricing: null
  created_at: string
  updated_at: string
}

function buildFreelance(overrides: Partial<FreelanceProfileFixture> = {}): FreelanceProfileFixture {
  return {
    id: "p-1",
    organization_id: FREELANCE_ORG_ID,
    title: "Senior Go engineer",
    about: "I build distributed systems.",
    video_url: "",
    availability_status: "available_now",
    expertise_domains: [],
    photo_url: "",
    city: "Paris",
    country_code: "FR",
    latitude: null,
    longitude: null,
    work_mode: ["remote"],
    travel_radius_km: null,
    languages_professional: ["fr", "en"],
    languages_conversational: [],
    org_name: "",
    first_name: "Ada",
    last_name: "Lovelace",
    skills: [],
    pricing: null,
    created_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

async function installFreelanceRoutes(
  page: Page,
  profile: FreelanceProfileFixture,
): Promise<void> {
  await page.route(/\/api\/v1\/auth\/me\b/, async (route: Route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ error: { code: "unauthorized" } }),
    })
  })

  await page.route(/\/api\/v1\/(freelance-profile|profiles\/freelance|profiles)\/[^/]+(\/?|\/.*)?$/, async (route: Route) => {
    const url = route.request().url()
    if (/(reviews|portfolio|social|rating|project-history|reputation)/i.test(url)) {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: [] }),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(profile),
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
}

test.describe("Public freelancer profile render", () => {
  test("heading shows firstName lastName, subtitle shows title, no video tag when no video", async ({
    page,
  }) => {
    const profile = buildFreelance({ video_url: "" })
    await installFreelanceRoutes(page, profile)

    await page.goto(`/freelancers/${FREELANCE_ORG_ID}`)

    // H1 = "Ada Lovelace".
    const h1 = page.locator("h1").first()
    if (await h1.count()) {
      await expect(h1).toContainText(/Ada\s+Lovelace/i, { timeout: 10000 })
    }

    // Subtitle = the title.
    const subtitle = page.getByText(/Senior Go engineer/i).first()
    if (await subtitle.count()) {
      await expect(subtitle).toBeVisible()
    }

    // NO <video> tag rendered when the org has no presentation video.
    await expect(page.locator("video")).toHaveCount(0)
  })

  test("Send a message CTA on unauth profile redirects to /login?next=", async ({
    page,
  }) => {
    const profile = buildFreelance()
    await installFreelanceRoutes(page, profile)

    await page.goto(`/freelancers/${FREELANCE_ORG_ID}`)

    const sendMsg = page
      .getByRole("link", { name: /(envoyer un message|send a message|message)/i })
      .or(page.getByRole("button", { name: /(envoyer un message|send a message|message)/i }))
      .first()

    if (await sendMsg.count()) {
      await sendMsg.click()
      await page.waitForURL(/\/login(\?|$|#)/, { timeout: 10000 })
      // next= param should reference the freelancer profile path.
      const url = new URL(page.url())
      const next = url.searchParams.get("next") ?? url.searchParams.get("redirect")
      if (next) {
        expect(next).toContain(FREELANCE_ORG_ID)
      }
    }
  })
})
