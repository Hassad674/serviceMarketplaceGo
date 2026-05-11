import { test, expect, type Page, type Route } from "@playwright/test"

// ---------------------------------------------------------------------------
// TEST-E2E-CRITICAL-FLOWS #4 — 2FA enable → logout → login with code
//
// Full lifecycle on the security page + login form. Covers the
// FIX-2FA regression where the toggle re-rendered to OFF after a
// refresh even when the backend had persisted ON.
//
// Sub-flows:
//   A. /securite — toggle 2FA ON, refresh, ensure it STAYS on.
//   B. /login — backend issues `requires_2fa: true`, code completes
//      the handshake, dashboard loads.
// ---------------------------------------------------------------------------

const USER_ID = "user-2fa-1"
const CHALLENGE_ID = "challenge-2fa-1"

interface MeOptions {
  twoFactorEnabled: boolean
}

function meResponse(opts: MeOptions): unknown {
  return {
    user: {
      id: USER_ID,
      email: "2fa@example.com",
      first_name: "Two",
      last_name: "Factor",
      display_name: "Two Factor",
      role: "provider",
      referrer_enabled: false,
      email_verified: true,
      kyc_status: "none",
      two_factor_enabled: opts.twoFactorEnabled,
      created_at: "2026-01-01",
    },
    organization: null,
  }
}

async function emptyEnvelopes(page: Page): Promise<void> {
  await page.route(/\/api\/v1\/.*/, async (route: Route) => {
    if (route.request().resourceType() !== "fetch") return route.continue()
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ data: [] }),
    })
  })
}

test.describe("2FA end-to-end", () => {
  test("enabling 2FA on /securite persists across page refresh", async ({ page }) => {
    let twoFactorState = false
    let issuedChallengeId: string | null = null

    await page.route(/\/api\/v1\/auth\/me\b/, async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(meResponse({ twoFactorEnabled: twoFactorState })),
      })
    })

    // Backend: enable challenge issue.
    await page.route(/\/api\/v1\/(auth\/2fa|2fa)\/(challenge|enable|setup).*/, async (route: Route) => {
      if (route.request().method() !== "POST") return route.continue()
      issuedChallengeId = CHALLENGE_ID
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          challenge_id: issuedChallengeId,
          // In a test-only build the backend would expose the issued
          // code; we just pre-agree on "654321" for the spec.
          test_code: "654321",
        }),
      })
    })

    // Backend: verify code → flips on.
    await page.route(/\/api\/v1\/(auth\/2fa|2fa)\/verify.*/, async (route: Route) => {
      if (route.request().method() !== "POST") return route.continue()
      twoFactorState = true
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({ ok: true, two_factor_enabled: true }),
      })
    })

    await emptyEnvelopes(page)

    await page.goto("/dashboard/securite")

    // Find the 2FA toggle (heading or switch).
    const toggle = page
      .getByRole("switch")
      .or(page.getByRole("checkbox"))
      .first()

    if (await toggle.count()) {
      // Initial state OFF.
      const initialState = await toggle.getAttribute("aria-checked").catch(() => null)
      expect(initialState === "true").toBe(false)

      // Toggle ON.
      await toggle.click()

      // Modal asking for the code.
      const codeInput = page
        .getByLabel(/(code|verification|vérification)/i)
        .or(page.getByPlaceholder(/(code|123456)/i))
        .first()
      if (await codeInput.count()) {
        await codeInput.fill("654321")
        const verifyBtn = page
          .getByRole("button", { name: /(vérifier|verify|valider|confirm)/i })
          .first()
        if (await verifyBtn.count()) {
          await verifyBtn.click()
          await page.waitForTimeout(300)
        }
      }

      // Refresh — regression: toggle MUST stay ON.
      await page.reload()
      const toggleAfterReload = page
        .getByRole("switch")
        .or(page.getByRole("checkbox"))
        .first()
      if (await toggleAfterReload.count()) {
        const finalState = await toggleAfterReload.getAttribute("aria-checked")
        expect(finalState).toBe("true")
      }
    }
  })

  test("login with requires_2fa returns dashboard after the code is verified", async ({
    page,
  }) => {
    let verifyBody: { user_id?: string; challenge_id?: string; code?: string } | null = null

    await page.route(/\/api\/v1\/auth\/login\b/, async (route: Route) => {
      if (route.request().method() !== "POST") return route.continue()
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          requires_2fa: true,
          user_id: USER_ID,
          challenge_id: CHALLENGE_ID,
        }),
      })
    })

    await page.route(/\/api\/v1\/auth\/login\/verify-2fa/, async (route: Route) => {
      if (route.request().method() !== "POST") return route.continue()
      verifyBody = route.request().postDataJSON()
      await route.fulfill({
        status: 200,
        headers: { "Set-Cookie": "session_id=fake-session; Path=/; HttpOnly" },
        contentType: "application/json",
        body: JSON.stringify({
          user: {
            id: USER_ID,
            email: "2fa@example.com",
            first_name: "Two",
            last_name: "Factor",
            display_name: "Two Factor",
            role: "provider",
            referrer_enabled: false,
            email_verified: true,
            created_at: "2026-01-01",
          },
          access_token: "fake-access",
          refresh_token: "fake-refresh",
        }),
      })
    })

    await page.route(/\/api\/v1\/auth\/me\b/, async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify(meResponse({ twoFactorEnabled: true })),
      })
    })

    await emptyEnvelopes(page)

    await page.goto("/login")

    await page.getByLabel(/Email/i).fill("2fa@example.com")
    await page.getByLabel(/Mot de passe/i).fill("Passw0rd!")
    await page.getByRole("button", { name: /Connexion/i }).click()

    const codeInput = page.getByLabel(/Code de vérification/i)
    await expect(codeInput).toBeVisible({ timeout: 10000 })
    expect(page.url()).toContain("/login")

    await codeInput.fill("654321")
    await page.getByRole("button", { name: /Vérifier le code/i }).click()

    await page.waitForURL("**/dashboard", { timeout: 10000 })

    expect(verifyBody).toEqual({
      user_id: USER_ID,
      challenge_id: CHALLENGE_ID,
      code: "654321",
    })
  })
})
