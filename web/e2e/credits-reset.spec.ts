import { test, expect, type BrowserContext } from "@playwright/test"
import {
  registerProvider,
  registerEnterprise,
  clearAuth,
  login,
} from "./helpers/auth"

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const API_URL = "http://localhost:8083"
const INITIAL_CREDITS = 10
const ADMIN_EMAIL = "admin@marketplace.local"
const ADMIN_PASSWORD = "Admin123!"

// ---------------------------------------------------------------------------
// API helpers
// ---------------------------------------------------------------------------

async function getSessionCookie(context: BrowserContext): Promise<string> {
  const cookies = await context.cookies()
  const session = cookies.find((c) => c.name === "session_id")
  return session ? `session_id=${session.value}` : ""
}

async function createJobViaAPI(
  cookie: string,
  index: number,
): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/jobs`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Cookie: cookie,
    },
    body: JSON.stringify({
      title: `E2E Reset Test Job ${index}`,
      description: `Automated test job #${index} for credits reset E2E test`,
      skills: ["testing"],
      applicant_type: "all",
      budget_type: "one_shot",
      min_budget: 1000,
      max_budget: 5000,
      is_indefinite: false,
      description_type: "text",
    }),
  })
  expect(res.ok, `Failed to create job ${index}: ${res.status}`).toBe(true)
  const data = await res.json()
  return data.id
}

async function fetchCreditsViaAPI(cookie: string): Promise<number> {
  const res = await fetch(`${API_URL}/api/v1/jobs/credits`, {
    headers: { Cookie: cookie },
  })
  expect(res.ok).toBe(true)
  const data = await res.json()
  return data.credits
}

async function applyToJobViaAPI(
  cookie: string,
  jobId: string,
  index: number,
): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/jobs/${jobId}/apply`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Cookie: cookie,
    },
    body: JSON.stringify({
      message: `Reset test application #${index}`,
    }),
  })
  expect(res.ok, `Failed to apply to job ${index}: ${res.status}`).toBe(true)
}

async function resetCreditsAsAdmin(cookie: string): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/admin/credits/reset`, {
    method: "POST",
    headers: { Cookie: cookie },
  })
  expect(
    res.ok,
    `Admin credits reset failed: ${res.status}`,
  ).toBe(true)
  const data = await res.json()
  expect(data.status).toBe("ok")
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe("Admin credits reset", () => {
  test.beforeEach(async ({ page }) => {
    await clearAuth(page)
  })

  test("credits below 10 are reset back to 10 by admin", async ({
    page,
    context,
  }) => {
    test.setTimeout(120_000)

    // Step 1: Register an enterprise and create 5 jobs
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)

    const jobIds: string[] = []
    for (let i = 1; i <= 5; i++) {
      const id = await createJobViaAPI(enterpriseCookie, i)
      jobIds.push(id)
    }

    // Step 2: Register a provider and apply to 5 jobs (credits 10 -> 5)
    await clearAuth(page)
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)
    const providerUser = await page.evaluate(() => {
      const raw = localStorage.getItem("marketplace-auth")
      if (!raw) return null
      return JSON.parse(raw)
    })

    const initialCredits = await fetchCreditsViaAPI(providerCookie)
    expect(initialCredits).toBe(INITIAL_CREDITS)

    for (let i = 0; i < 5; i++) {
      await applyToJobViaAPI(providerCookie, jobIds[i], i + 1)
    }

    const creditsAfterApply = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterApply).toBe(5)

    // Step 3: Login as admin and trigger the weekly reset
    await clearAuth(page)
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD)
    const adminCookie = await getSessionCookie(context)
    await resetCreditsAsAdmin(adminCookie)

    // Step 4: Login back as provider and verify credits are restored to 10
    await clearAuth(page)
    await login(
      page,
      providerUser?.state?.user?.email ?? "",
      "TestPass1234!",
    )
    const providerCookieAfter = await getSessionCookie(context)
    const creditsAfterReset = await fetchCreditsViaAPI(providerCookieAfter)
    expect(creditsAfterReset).toBe(INITIAL_CREDITS)
  })

  test("credits at 10 are unchanged after admin reset", async ({
    page,
    context,
  }) => {
    test.setTimeout(60_000)

    // Step 1: Register a provider (starts with 10 credits)
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)
    const providerUser = await page.evaluate(() => {
      const raw = localStorage.getItem("marketplace-auth")
      if (!raw) return null
      return JSON.parse(raw)
    })

    const initialCredits = await fetchCreditsViaAPI(providerCookie)
    expect(initialCredits).toBe(INITIAL_CREDITS)

    // Step 2: Login as admin and trigger the weekly reset
    await clearAuth(page)
    await login(page, ADMIN_EMAIL, ADMIN_PASSWORD)
    const adminCookie = await getSessionCookie(context)
    await resetCreditsAsAdmin(adminCookie)

    // Step 3: Login back as provider and verify credits still = 10
    await clearAuth(page)
    await login(
      page,
      providerUser?.state?.user?.email ?? "",
      "TestPass1234!",
    )
    const providerCookieAfter = await getSessionCookie(context)
    const creditsAfterReset = await fetchCreditsViaAPI(providerCookieAfter)
    expect(creditsAfterReset).toBe(INITIAL_CREDITS)
  })

  test("non-admin user cannot trigger credits reset", async ({
    page,
    context,
  }) => {
    // Register a regular provider and try to call the admin endpoint
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)

    const res = await fetch(`${API_URL}/api/v1/admin/credits/reset`, {
      method: "POST",
      headers: { Cookie: providerCookie },
    })

    // Should receive 403 Forbidden
    expect(res.status).toBe(403)
  })
})
