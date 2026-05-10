import { test, expect } from "@playwright/test"
import { clearAuth } from "./helpers/auth"

// Bug A regression: an unauthenticated visitor reaching the
// invitation acceptance page from the email link must NOT be
// hard-redirected to /login. The PostHogProvider mounted in the
// locale layout fires `useSession()` on every page; without the
// `/invitation` whitelist in `AUTH_PUBLIC_PATHS` the resulting 401
// from /auth/me triggered `window.location.href = "/login"`.
//
// We don't seed a real invitation token here — the page renders an
// "expired/invalid invitation" panel for any token, which is enough
// to assert the critical invariant: the URL stays on /invitation
// and never bounces to /login.
test.describe("Invitation acceptance — public access", () => {
  test.beforeEach(async ({ page }) => {
    await page.goto("/")
    await clearAuth(page)
  })

  test("an unauthenticated visitor stays on /fr/invitation/:token", async ({ page }) => {
    const token = "fake-token-for-public-access-check"

    await page.goto(`/fr/invitation/${token}`)

    // Give the React tree time to mount + run any session bootstrap
    // that might trigger a redirect. The bug surface is precisely a
    // window.location.href = "/login" assignment fired async after
    // the /auth/me fetch resolves with 401.
    await page.waitForTimeout(1500)

    expect(page.url()).toContain(`/invitation/${token}`)
    expect(page.url()).not.toContain("/login")
  })

  test("an unauthenticated visitor stays on /en/invitation/:token", async ({ page }) => {
    const token = "fake-token-for-public-access-check"

    await page.goto(`/en/invitation/${token}`)
    await page.waitForTimeout(1500)

    expect(page.url()).toContain(`/invitation/${token}`)
    expect(page.url()).not.toContain("/login")
  })
})
