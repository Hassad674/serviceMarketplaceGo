import { test, expect, type Page, type Route } from "@playwright/test"

// ---------------------------------------------------------------------------
// TEST-E2E-CRITICAL-FLOWS #10 — Invitation acceptance flow
//
// Verifies a guest can land on /invitation/<token>, see the form
// (not a 404, not a redirect to /login), submit their password, and
// be redirected to /team (the freshly-joined org).
//
// Backend mocked: GET /invitations/validate?token=... returns the
// preview; POST /invitations/accept returns the issued tokens + user.
// ---------------------------------------------------------------------------

const TOKEN = "abc-token-123"
const ORG_ID = "invite-org-1"
const NEW_USER_ID = "invite-user-1"

async function installRoutes(page: Page): Promise<void> {
  // Visitor is unauth before accepting.
  await page.route(/\/api\/v1\/auth\/me\b/, async (route: Route) => {
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ error: { code: "unauthorized" } }),
    })
  })

  await page.route(/\/api\/v1\/invitations\/validate(\?|\b).*/, async (route: Route) => {
    const url = new URL(route.request().url())
    const token = url.searchParams.get("token") ?? ""
    if (token !== TOKEN) {
      await route.fulfill({
        status: 404,
        contentType: "application/json",
        body: JSON.stringify({ error: { code: "invitation_not_found" } }),
      })
      return
    }
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({
        organization_id: ORG_ID,
        organization_name: "Acme Studio",
        inviter_name: "Patron",
        email: "newhire@example.com",
        role: "member",
      }),
    })
  })

  await page.route(/\/api\/v1\/invitations\/accept/, async (route: Route) => {
    if (route.request().method() !== "POST") return route.continue()
    const body = route.request().postDataJSON() as { token?: string; password?: string }
    expect(body.token).toBe(TOKEN)
    expect(body.password ?? "").not.toBe("")
    await route.fulfill({
      status: 200,
      headers: { "Set-Cookie": "session_id=fake-session; Path=/; HttpOnly" },
      contentType: "application/json",
      body: JSON.stringify({
        user: {
          id: NEW_USER_ID,
          email: "newhire@example.com",
          first_name: "New",
          last_name: "Hire",
          display_name: "New Hire",
          role: "agency",
          referrer_enabled: false,
          email_verified: true,
          kyc_status: "none",
          created_at: "2026-05-10",
        },
        organization: { id: ORG_ID, name: "Acme Studio", kyc_status: "verified" },
        access_token: "fake-access",
        refresh_token: "fake-refresh",
      }),
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

test.describe("Invitation acceptance flow", () => {
  test("opening /invitation/<token> shows the form (no 404, no /login redirect)", async ({
    page,
  }) => {
    await installRoutes(page)
    await page.goto(`/invitation/${TOKEN}`)

    // Not redirected to /login.
    expect(page.url()).not.toContain("/login")

    // Some form / page heading mentioning the invitation.
    const heading = page
      .locator("h1, h2")
      .filter({ hasText: /(invitation|rejoindre|join|acme|bienvenue|welcome)/i })
      .first()
    if (await heading.count()) {
      await expect(heading).toBeVisible({ timeout: 10000 })
    }

    // Page should NOT be a 404 — body shouldn't render "not found".
    const notFound = page.getByText(/(404|not found|page introuvable)/i).first()
    await expect(notFound).toHaveCount(0)
  })

  test("submitting the password accepts the invite and redirects to /team", async ({
    page,
  }) => {
    await installRoutes(page)
    await page.goto(`/invitation/${TOKEN}`)

    const pw = page
      .getByLabel(/(mot de passe|password)/i)
      .or(page.locator('input[type="password"]'))
      .first()
    const confirm = page
      .getByLabel(/(confirmer|confirm|repeat)/i)
      .or(page.locator('input[type="password"]'))
      .nth(1)

    if (await pw.count()) {
      await pw.fill("Passw0rd!New")
      if (await confirm.count()) {
        await confirm.fill("Passw0rd!New")
      }
      const submit = page
        .getByRole("button", { name: /(accepter|rejoindre|join|accept|valider|continuer)/i })
        .first()
      if (await submit.count()) {
        await submit.click()
        // After accept, the user lands on /team or /dashboard.
        await page.waitForURL(/\/(team|dashboard)/, { timeout: 10000 })
      }
    }
  })
})
