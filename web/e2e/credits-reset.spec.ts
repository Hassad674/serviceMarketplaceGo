import { test, expect, type BrowserContext } from "@playwright/test"
import { registerProvider, registerEnterprise } from "./helpers/auth"

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

async function loginViaAPI(email: string, password: string): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
    redirect: "manual",
  })
  const setCookieHeaders = res.headers.getSetCookie?.() ?? []
  const allCookies = setCookieHeaders.length > 0
    ? setCookieHeaders
    : (res.headers.get("set-cookie") ?? "").split(", ")
  const cookies: string[] = []
  for (const c of allCookies) {
    const match = c.match(/^([^=]+=[^;]+)/)
    if (match) cookies.push(match[1])
  }
  if (cookies.length > 0) return cookies.join("; ")
  throw new Error(`Login failed for ${email}: ${res.status} - no cookies`)
}

async function resetCreditsForUser(adminCookie: string, userId: string): Promise<void> {
  const res = await fetch(`${API_URL}/api/v1/admin/credits/reset/${userId}`, {
    method: "POST",
    headers: { Cookie: adminCookie },
  })
  expect(
    res.ok,
    `Admin credits reset for user failed: ${res.status}`,
  ).toBe(true)
  const data = await res.json()
  expect(data.status).toBe("ok")
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

test.describe("Admin credits reset", () => {
  test.beforeEach(async ({ page, context }) => {
    await context.clearCookies()
    await page.goto("/")
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
    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)

    // Get provider user ID via API
    const meRes = await fetch(`${API_URL}/api/v1/auth/me`, {
      headers: { Cookie: providerCookie },
    })
    const meData = await meRes.json()
    const providerUserId = meData.id ?? meData.data?.id
    expect(providerUserId).toBeTruthy()

    const initialCredits = await fetchCreditsViaAPI(providerCookie)
    expect(initialCredits).toBe(INITIAL_CREDITS)

    for (let i = 0; i < 5; i++) {
      await applyToJobViaAPI(providerCookie, jobIds[i], i + 1)
    }

    const creditsAfterApply = await fetchCreditsViaAPI(providerCookie)
    expect(creditsAfterApply).toBe(5)

    // Step 3: Login as admin via API and trigger reset for this user only
    const adminCookie = await loginViaAPI(ADMIN_EMAIL, ADMIN_PASSWORD)
    await resetCreditsForUser(adminCookie, providerUserId)

    // Step 4: Verify credits are restored to 10 (using existing provider cookie)
    const providerCookieAfter = providerCookie
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

    const meRes = await fetch(`${API_URL}/api/v1/auth/me`, {
      headers: { Cookie: providerCookie },
    })
    const meData = await meRes.json()
    const providerUserId = meData.id ?? meData.data?.id
    expect(providerUserId).toBeTruthy()

    const initialCredits = await fetchCreditsViaAPI(providerCookie)
    expect(initialCredits).toBe(INITIAL_CREDITS)

    // Step 2: Login as admin via API and trigger reset for this user only
    const adminCookie = await loginViaAPI(ADMIN_EMAIL, ADMIN_PASSWORD)
    await resetCreditsForUser(adminCookie, providerUserId)

    // Step 3: Verify credits still = 10
    const creditsAfterReset = await fetchCreditsViaAPI(providerCookie)
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
