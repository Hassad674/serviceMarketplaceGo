import { test, expect, type BrowserContext } from "@playwright/test"
import { registerProvider, registerEnterprise, clearAuth } from "./helpers/auth"

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const API_URL = "http://localhost:8083"
const TOTAL_JOBS = 11 // 10 to exhaust credits + 1 to test rejection
const INITIAL_CREDITS = 10

// ---------------------------------------------------------------------------
// API helpers — create jobs directly via backend
// ---------------------------------------------------------------------------

/**
 * Extract the session cookie from the browser context so we can make
 * authenticated API calls with fetch (outside the browser).
 */
async function getSessionCookie(context: BrowserContext): Promise<string> {
  const cookies = await context.cookies()
  const session = cookies.find((c) => c.name === "session_id")
  return session ? `session_id=${session.value}` : ""
}

/**
 * Create a single open job via the backend API.
 * Returns the created job's ID.
 */
async function createJobViaAPI(cookie: string, index: number): Promise<string> {
  const res = await fetch(`${API_URL}/api/v1/jobs`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Cookie: cookie,
    },
    body: JSON.stringify({
      title: `E2E Credit Test Job ${index}`,
      description: `Automated test job #${index} for application credits E2E test`,
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

/**
 * Fetch current credits for the authenticated user via the backend API.
 */
async function fetchCreditsViaAPI(cookie: string): Promise<number> {
  const res = await fetch(`${API_URL}/api/v1/jobs/credits`, {
    headers: { Cookie: cookie },
  })
  expect(res.ok).toBe(true)
  const data = await res.json()
  return data.credits
}

// ---------------------------------------------------------------------------
// Application credits E2E tests
// ---------------------------------------------------------------------------

test.describe("Application credits system", () => {
  test.beforeEach(async ({ page, context }) => {
    await context.clearCookies()
    await page.goto("/")
  })

  test("new provider starts with 10 credits", async ({ page }) => {
    await registerProvider(page)

    // Navigate to opportunities page where credits are displayed
    await page.goto("/opportunities")
    await page.waitForLoadState("networkidle")

    // The credits badge should show 10
    await expect(
      page.getByText(`${INITIAL_CREDITS} credits remaining`),
    ).toBeVisible({ timeout: 15000 })
  })

  test("credits decrement after each application", async ({ page, context }) => {
    // Step 1: Register an enterprise and create 3 jobs via API
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)

    const jobIds: string[] = []
    for (let i = 1; i <= 3; i++) {
      const id = await createJobViaAPI(enterpriseCookie, i)
      jobIds.push(id)
    }

    // Step 2: Register a provider (clears enterprise session)
    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)

    // Verify initial credits via API
    const initialCredits = await fetchCreditsViaAPI(providerCookie)
    expect(initialCredits).toBe(INITIAL_CREDITS)

    // Step 3: Apply to jobs and verify credits go down
    for (let i = 0; i < 3; i++) {
      await page.goto(`/opportunities/${jobIds[i]}`)
      await page.waitForLoadState("networkidle")

      // Wait for the Apply button to be visible and enabled
      const applyButton = page.getByRole("button", { name: /Apply/i })
      await expect(applyButton).toBeVisible({ timeout: 10000 })
      await expect(applyButton).toBeEnabled()

      // Click Apply to open the modal
      await applyButton.click()

      // Fill in the application message
      const messageInput = page.getByPlaceholder(/Why are you the right fit/i)
      await expect(messageInput).toBeVisible({ timeout: 5000 })
      await messageInput.fill(`Test application message for job ${i + 1}`)

      // Submit the application via the modal's Apply button
      const submitButton = page
        .getByRole("dialog")
        .or(page.locator(".fixed"))
        .getByRole("button", { name: /Apply/i })
      await submitButton.click()

      // Wait for the application to complete (modal closes)
      await expect(messageInput).not.toBeVisible({ timeout: 10000 })

      // Verify credits decremented via API
      const updatedCookie = await getSessionCookie(context)
      const creditsAfter = await fetchCreditsViaAPI(updatedCookie)
      expect(creditsAfter).toBe(INITIAL_CREDITS - (i + 1))
    }
  })

  test("exhausting all 10 credits blocks further applications", async ({
    page,
    context,
  }) => {
    // Increase timeout — this test creates 11 jobs and applies to 10
    test.setTimeout(180_000)

    // Step 1: Register an enterprise and create 11 jobs via API
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)

    const jobIds: string[] = []
    for (let i = 1; i <= TOTAL_JOBS; i++) {
      const id = await createJobViaAPI(enterpriseCookie, i)
      jobIds.push(id)
    }
    expect(jobIds).toHaveLength(TOTAL_JOBS)

    // Step 2: Register a provider
    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)

    // Verify initial credits
    const initialCredits = await fetchCreditsViaAPI(providerCookie)
    expect(initialCredits).toBe(INITIAL_CREDITS)

    // Step 3: Apply to 10 jobs (exhausting all credits)
    for (let i = 0; i < INITIAL_CREDITS; i++) {
      await page.goto(`/opportunities/${jobIds[i]}`)
      await page.waitForLoadState("networkidle")

      // Wait for Apply button
      const applyButton = page.getByRole("button", { name: /Apply/i })
      await expect(applyButton).toBeVisible({ timeout: 10000 })
      await expect(applyButton).toBeEnabled()

      // Open apply modal
      await applyButton.click()

      // Fill message
      const messageInput = page.getByPlaceholder(/Why are you the right fit/i)
      await expect(messageInput).toBeVisible({ timeout: 5000 })
      await messageInput.fill(`Credit test application #${i + 1}`)

      // Submit
      const submitButton = page
        .locator(".fixed")
        .getByRole("button", { name: /Apply/i })
      await submitButton.click()

      // Wait for modal to close (application submitted)
      await expect(messageInput).not.toBeVisible({ timeout: 10000 })
    }

    // Step 4: Verify credits are now 0 via API
    const updatedCookie = await getSessionCookie(context)
    const creditsAfterExhaustion = await fetchCreditsViaAPI(updatedCookie)
    expect(creditsAfterExhaustion).toBe(0)

    // Step 5: Navigate to the 11th job (the one we haven't applied to)
    await page.goto(`/opportunities/${jobIds[INITIAL_CREDITS]}`)
    await page.waitForLoadState("networkidle")

    // Step 6: Verify the Apply button is disabled
    const applyButton = page.getByRole("button", { name: /Apply/i })
    await expect(applyButton).toBeVisible({ timeout: 10000 })
    await expect(applyButton).toBeDisabled()

    // Step 7: Verify the "no credits left" warning appears
    await expect(
      page.getByText(/no application credits left/i),
    ).toBeVisible({ timeout: 10000 })
  })

  test("no credits warning banner appears on opportunities list", async ({
    page,
    context,
  }) => {
    // Increase timeout for job creation + applications
    test.setTimeout(180_000)

    // Create 10 jobs as enterprise
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)

    const jobIds: string[] = []
    for (let i = 1; i <= INITIAL_CREDITS; i++) {
      const id = await createJobViaAPI(enterpriseCookie, i)
      jobIds.push(id)
    }

    // Register a provider and exhaust all 10 credits via API
    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)

    // Apply to all 10 jobs via the backend API directly (faster than UI)
    for (let i = 0; i < INITIAL_CREDITS; i++) {
      const res = await fetch(`${API_URL}/api/v1/jobs/${jobIds[i]}/apply`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Cookie: providerCookie,
        },
        body: JSON.stringify({
          message: `Quick apply for credit exhaustion test #${i + 1}`,
        }),
      })
      expect(
        res.ok,
        `Failed to apply to job ${i + 1}: ${res.status}`,
      ).toBe(true)
    }

    // Verify credits are 0
    const credits = await fetchCreditsViaAPI(providerCookie)
    expect(credits).toBe(0)

    // Navigate to the opportunities list page
    await page.goto("/opportunities")
    await page.waitForLoadState("networkidle")

    // The "no credits left" warning banner should be visible
    await expect(
      page.getByText(/no application credits left/i),
    ).toBeVisible({ timeout: 15000 })

    // The credits badge should show 0
    await expect(
      page.getByText("0 credits remaining"),
    ).toBeVisible({ timeout: 10000 })
  })

  test("already-applied jobs show disabled button without credit cost", async ({
    page,
    context,
  }) => {
    // Create a job as enterprise
    await registerEnterprise(page)
    const enterpriseCookie = await getSessionCookie(context)
    const jobId = await createJobViaAPI(enterpriseCookie, 1)

    // Register a provider
    await context.clearCookies()
    await page.goto("/")
    await registerProvider(page)
    const providerCookie = await getSessionCookie(context)

    // Apply via API
    const res = await fetch(`${API_URL}/api/v1/jobs/${jobId}/apply`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Cookie: providerCookie,
      },
      body: JSON.stringify({ message: "Already applied test" }),
    })
    expect(res.ok).toBe(true)

    // Navigate to the job detail page
    await page.goto(`/opportunities/${jobId}`)
    await page.waitForLoadState("networkidle")

    // The button should show "Already Applied" and be disabled
    const applyButton = page.getByRole("button", { name: /Already Applied/i })
    await expect(applyButton).toBeVisible({ timeout: 10000 })
    await expect(applyButton).toBeDisabled()

    // Verify credits are 9 (only 1 used) — not 0
    const credits = await fetchCreditsViaAPI(providerCookie)
    expect(credits).toBe(INITIAL_CREDITS - 1)
  })
})
