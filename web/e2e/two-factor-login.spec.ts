import { test, expect } from "@playwright/test"

// ---------------------------------------------------------------------------
// B.6 — Web 2FA login happy path with a fully mocked backend.
//
// Goal: verify that when /auth/login responds with the
// `requires_2fa: true` envelope, the LoginForm swaps to the 6-digit
// challenge UI and a successful /auth/login/verify-2fa lands the user
// on /dashboard (no full-page redirect, no extra route hop).
//
// Backend interception keeps this test deterministic — the live API
// requires a real opted-in user + a real Resend email round trip,
// neither of which is acceptable in CI.
// ---------------------------------------------------------------------------

test.describe("two-factor login flow", () => {
  test("requires_2fa response gates the login on a 6-digit code", async ({
    page,
  }) => {
    let verifyCalledWith: { user_id?: string; challenge_id?: string; code?: string } | null = null

    // /auth/login → returns the 2FA challenge envelope.
    await page.route("**/api/v1/auth/login", async (route) => {
      const req = route.request()
      if (req.method() !== "POST") {
        return route.continue()
      }
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          requires_2fa: true,
          user_id: "user-123",
          challenge_id: "challenge-456",
        }),
      })
    })

    // /auth/login/verify-2fa → records the body, returns 200 + a fake
    // session cookie so any downstream /auth/me can reasonably 401
    // without breaking the redirect assertion.
    await page.route("**/api/v1/auth/login/verify-2fa", async (route) => {
      const req = route.request()
      if (req.method() !== "POST") {
        return route.continue()
      }
      const body = req.postDataJSON() as {
        user_id: string
        challenge_id: string
        code: string
      }
      verifyCalledWith = body
      await route.fulfill({
        status: 200,
        headers: {
          "Set-Cookie": "session_id=fake-session; Path=/; HttpOnly",
        },
        contentType: "application/json",
        body: JSON.stringify({
          user: {
            id: "user-123",
            email: "test@example.com",
            first_name: "Test",
            last_name: "User",
            display_name: "Test User",
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

    // /auth/me → after the verify call, the dashboard refetches the
    // session. Give it a valid response so the redirect lands.
    await page.route("**/api/v1/auth/me", async (route) => {
      await route.fulfill({
        status: 200,
        contentType: "application/json",
        body: JSON.stringify({
          user: {
            id: "user-123",
            email: "test@example.com",
            first_name: "Test",
            last_name: "User",
            display_name: "Test User",
            role: "provider",
            referrer_enabled: false,
            email_verified: true,
            kyc_status: "none",
            created_at: "2026-01-01",
          },
          organization: null,
        }),
      })
    })

    await page.goto("/login")

    await page.getByLabel(/Email/i).fill("test@example.com")
    await page.getByLabel(/Mot de passe/i).fill("Passw0rd!")
    await page.getByRole("button", { name: /Connexion/i }).click()

    // The verification UI shows up — we never left /login.
    const codeInput = page.getByLabel(/Code de vérification/i)
    await expect(codeInput).toBeVisible({ timeout: 10000 })
    expect(page.url()).toContain("/login")

    await codeInput.fill("654321")
    await page.getByRole("button", { name: /Vérifier le code/i }).click()

    // Successful verify → dashboard redirect.
    await page.waitForURL("**/dashboard", { timeout: 10000 })

    expect(verifyCalledWith).toEqual({
      user_id: "user-123",
      challenge_id: "challenge-456",
      code: "654321",
    })
  })
})
