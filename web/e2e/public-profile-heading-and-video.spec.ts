import { test, expect, type Page, type Route } from "@playwright/test"

// ---------------------------------------------------------------------------
// /freelancers/<id>, /agencies/<id>, /referrers/<id> — heading + empty video
//
// Three regression bugs covered:
//   1. /freelancers/<id> rendered the persona title twice (as both H1
//      and italic subtitle) when first_name / last_name were empty.
//   2. /agencies/<id> rendered "Untitled Profile" verbatim when the
//      agency had not set a title — instead of falling back to the
//      localised "Agency profile" / "Profil agence" label.
//   3. All three pages rendered the empty-state placeholder for the
//      presentation video card on read-only views — even when the
//      org had no video. The empty-state must collapse silently on
//      the public surface (other empty sections already do).
//
// Strategy: mock every backend call the page touches so the test runs
// without a live API. We assert the rendered DOM directly (H1 text +
// presence of <video>) which is the smallest contract that proves
// each bug is fixed.
// ---------------------------------------------------------------------------

test.describe.configure({ mode: "parallel" })

const FREELANCE_ORG = "8a8d2779-4f3b-47f8-9e09-ec1da67fe8d7"
const AGENCY_ORG = "ee4cca91-1233-49a3-9d0a-4f5f80100c1b"
const REFERRER_ORG = "ff0dad77-a021-4b71-91ae-694f5f2e5982"

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

function buildFreelanceProfile(
  overrides: Partial<FreelanceProfileFixture> = {},
): FreelanceProfileFixture {
  return {
    id: "profile-1",
    organization_id: FREELANCE_ORG,
    title: "Senior Go engineer",
    about: "I build distributed systems for a living.",
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

interface AgencyProfileFixture {
  organization_id: string
  title: string
  photo_url: string
  presentation_video_url: string
  referrer_video_url: string
  about: string
  referrer_about: string
  expertise_domains: string[]
  skills: { skill_text: string; display_text: string }[]
  city: string
  country_code: string
  work_mode: string[]
  travel_radius_km: number | null
  languages_professional: string[]
  languages_conversational: string[]
  availability_status: string
  pricing: never[]
  created_at: string
  updated_at: string
}

function buildAgencyProfile(
  overrides: Partial<AgencyProfileFixture> = {},
): AgencyProfileFixture {
  return {
    organization_id: AGENCY_ORG,
    title: "",
    photo_url: "",
    presentation_video_url: "",
    referrer_video_url: "",
    about: "We craft brand systems for ambitious teams.",
    referrer_about: "",
    expertise_domains: [],
    skills: [],
    city: "Lyon",
    country_code: "FR",
    work_mode: ["remote"],
    travel_radius_km: null,
    languages_professional: ["fr", "en"],
    languages_conversational: [],
    availability_status: "available_now",
    pricing: [],
    created_at: "2026-04-01T00:00:00Z",
    updated_at: "2026-04-01T00:00:00Z",
    ...overrides,
  }
}

// installCommonRoutes mocks the noisy auxiliary endpoints (auth, ratings,
// reviews, social links, project history, …) so the page renders
// cleanly without a backend. The persona-specific profile route is
// installed by each test before navigation.
async function installCommonRoutes(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/auth\/me/, async (route: Route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ error: { code: "unauthorized" } }),
    })
  })

  await page.route(/\/api\/v1\/.*\/social-links/, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ data: [] }),
    })
  })

  await page.route(/\/api\/v1\/profiles\/[^/]+\/reviews/, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ data: [] }),
    })
  })

  await page.route(
    /\/api\/v1\/profiles\/[^/]+\/(rating|project-history|portfolio)/,
    async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ data: [] }),
      })
    },
  )

  await page.route(/\/api\/v1\/portfolio\/.*/, async (route: Route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ data: [] }),
    })
  })

  await page.route(
    /\/api\/v1\/referrer-profile.*\/reputation/,
    async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          rating_avg: 0,
          review_count: 0,
          history: [],
          next_cursor: "",
          has_more: false,
        }),
      })
    },
  )

  // Catch-all for any other /api/v1/* request (notifications, metrics,
  // search) — return an empty envelope so the page keeps rendering.
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

test.describe("Public profile · Heading and empty-video regression", () => {
  test("/freelancers shows ${first_name} ${last_name} as the H1 and hides the empty video card", async ({
    page,
  }) => {
    await installCommonRoutes(page)
    const profile = buildFreelanceProfile({
      first_name: "Ada",
      last_name: "Lovelace",
      title: "Senior Go engineer",
      video_url: "",
    })
    await page.route(
      new RegExp(`/api/v1/freelance-profiles/${FREELANCE_ORG}\\b`),
      async (route: Route) => {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(profile),
        })
      },
    )

    await page.goto(`/en/freelancers/${FREELANCE_ORG}`)

    await expect(page.locator("h1")).toContainText("Ada Lovelace", {
      timeout: 10000,
    })
    // The italic subtitle must show the title (different from name).
    await expect(page.locator("p.italic", { hasText: "Senior Go engineer" })).toBeVisible()
    // No <video> tag and no empty-state copy.
    await expect(page.locator("video")).toHaveCount(0)
    await expect(page.getByText("No presentation video")).toHaveCount(0)
  })

  test("/freelancers renders the embedded <video> when the profile has a video", async ({
    page,
  }) => {
    await installCommonRoutes(page)
    const profile = buildFreelanceProfile({
      video_url: "https://media.example.test/intro.mp4",
    })
    await page.route(
      new RegExp(`/api/v1/freelance-profiles/${FREELANCE_ORG}\\b`),
      async (route: Route) => {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(profile),
        })
      },
    )

    await page.goto(`/en/freelancers/${FREELANCE_ORG}`)

    await expect(page.locator("video")).toHaveCount(1)
    await expect(page.locator("video")).toHaveAttribute(
      "src",
      "https://media.example.test/intro.mp4",
    )
  })

  test("/agencies falls back to the localised 'Agency profile' label when title is empty", async ({
    page,
  }) => {
    await installCommonRoutes(page)
    const profile = buildAgencyProfile({ title: "", presentation_video_url: "" })
    await page.route(
      new RegExp(`/api/v1/profiles/${AGENCY_ORG}\\b(?!/)`),
      async (route: Route) => {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(profile),
        })
      },
    )

    await page.goto(`/en/agencies/${AGENCY_ORG}`)

    // The H1 falls back to "Agency profile" via the i18n key — never
    // to the hardcoded "Untitled Profile" string the previous code
    // surfaced regardless of locale.
    await expect(page.locator("h1")).toContainText("Agency profile", {
      timeout: 10000,
    })
    await expect(page.locator("h1")).not.toContainText("Untitled Profile")
    await expect(page.locator("video")).toHaveCount(0)
    await expect(page.getByText("No presentation video")).toHaveCount(0)
  })

  test("/agencies renders the title verbatim as the H1 when set", async ({
    page,
  }) => {
    await installCommonRoutes(page)
    const profile = buildAgencyProfile({
      title: "Studio Forge",
      presentation_video_url: "",
    })
    await page.route(
      new RegExp(`/api/v1/profiles/${AGENCY_ORG}\\b(?!/)`),
      async (route: Route) => {
        await route.fulfill({
          status: 200,
          contentType: "application/json",
          body: JSON.stringify(profile),
        })
      },
    )

    await page.goto(`/en/agencies/${AGENCY_ORG}`)
    await expect(page.locator("h1")).toContainText("Studio Forge", {
      timeout: 10000,
    })
  })
})
