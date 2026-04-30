import { test, expect } from "@playwright/test"

/**
 * SEC-06: refresh-token rotation.
 *
 * Every successful /auth/refresh blacklists the JTI of the consumed
 * refresh token. A replay returns 401 because the JTI is already on
 * the blacklist — this is the canonical signal that the token was
 * stolen + already used by either the legitimate user OR the
 * attacker.
 *
 * Test happy-path: register → refresh → save new pair. Test replay:
 * reuse the OLD refresh token → 401.
 */

const STRONG_PASSWORD = "TestPass1234!"

test.describe("SEC-06 refresh-token rotation", () => {
  test("rotation blacklists the old refresh token", async ({ request }) => {
    const email = `rot-${Date.now()}@playwright.com`

    const register = await request.post("/api/v1/auth/register", {
      data: {
        email,
        password: STRONG_PASSWORD,
        first_name: "Rotation",
        last_name: "Tester",
        display_name: "Rotation Tester",
        role: "provider",
      },
      headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
      failOnStatusCode: false,
    })
    if (register.status() !== 201 && register.status() !== 200) {
      // Skip when the deployment does not allow open registrations or
      // the moderation pipeline rejects the synthetic display_name.
      test.skip(true, `register returned ${register.status()}`)
    }

    const initial = await register.json()
    const oldRefresh = initial.refresh_token as string
    expect(oldRefresh).toBeTruthy()

    const refreshOnce = await request.post("/api/v1/auth/refresh", {
      data: { refresh_token: oldRefresh },
      headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
    })
    expect(refreshOnce.status()).toBe(200)
    const rotated = await refreshOnce.json()
    expect(rotated.refresh_token).toBeTruthy()
    expect(rotated.refresh_token).not.toBe(oldRefresh)

    // Replay the OLD refresh token — must be 401.
    const replay = await request.post("/api/v1/auth/refresh", {
      data: { refresh_token: oldRefresh },
      headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
      failOnStatusCode: false,
    })
    expect(replay.status()).toBe(401)
  })

  test("logout blacklists the refresh token", async ({ request }) => {
    const email = `lo-${Date.now()}@playwright.com`

    const register = await request.post("/api/v1/auth/register", {
      data: {
        email,
        password: STRONG_PASSWORD,
        first_name: "Logout",
        last_name: "Tester",
        display_name: "Logout Tester",
        role: "provider",
      },
      headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
      failOnStatusCode: false,
    })
    if (register.status() !== 201 && register.status() !== 200) {
      test.skip(true, `register returned ${register.status()}`)
    }
    const data = await register.json()
    const accessToken = data.access_token as string
    const refreshToken = data.refresh_token as string

    const logout = await request.post("/api/v1/auth/logout", {
      data: { refresh_token: refreshToken },
      headers: {
        "Content-Type": "application/json",
        "X-Auth-Mode": "token",
        Authorization: `Bearer ${accessToken}`,
      },
      failOnStatusCode: false,
    })
    // Logout always returns 200 from the user's POV.
    expect(logout.status()).toBe(200)

    // Now the refresh token must be unusable.
    const replay = await request.post("/api/v1/auth/refresh", {
      data: { refresh_token: refreshToken },
      headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
      failOnStatusCode: false,
    })
    expect(replay.status()).toBe(401)
  })
})
