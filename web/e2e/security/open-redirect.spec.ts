import { test, expect } from "@playwright/test"

// ---------------------------------------------------------------------------
// gosec G710 — Open redirect on /api/v1/me/invoices/{id}/pdf and
// /api/v1/admin/invoices/{id}/pdf.
//
// The backend now passes the URL returned by the storage adapter
// through validateStorageRedirect before issuing a 302. The unit
// suite in internal/handler/open_redirect_test.go drives the handler
// with a fake storage adapter that returns hostile URLs and verifies
// every variant is rejected with 502.
//
// This spec is a smoke-test: it confirms the wired endpoint behaves
// correctly when poked from the wire. Without admin/staff
// credentials we cannot trigger the success path, so we focus on:
//   - hitting the endpoint with bad / non-existent IDs returns 4xx
//     (no leak of internal URLs);
//   - the response NEVER carries a Location: header with an
//     attacker-controlled host (defense-in-depth — even a future
//     misroute should not turn into an open redirect).
// ---------------------------------------------------------------------------

test.describe("gosec G710 — invoice PDF redirect hardening", () => {
  test("GET /api/v1/me/invoices/{id}/pdf with an unknown ID returns 4xx", async ({ request }) => {
    const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8083"
    const fakeID = "00000000-0000-0000-0000-000000000000"

    // Without auth, the endpoint rejects at the auth middleware.
    // The crucial assertion is the response is NOT a redirect.
    const resp = await request.get(`${apiBase}/api/v1/me/invoices/${fakeID}/pdf`, {
      maxRedirects: 0,
    })
    expect(resp.status()).toBeGreaterThanOrEqual(400)
    expect(resp.status()).toBeLessThan(500)
    // No Location header on a rejected request — even if the
    // backend were to misroute, the validateStorageRedirect gate
    // would convert any non-allowlisted URL into a 502.
    expect(resp.headers()["location"]).toBeFalsy()
  })

  test("GET /api/v1/admin/invoices/{id}/pdf with an unknown ID returns 4xx", async ({
    request,
  }) => {
    const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8083"
    const fakeID = "00000000-0000-0000-0000-000000000000"

    const resp = await request.get(
      `${apiBase}/api/v1/admin/invoices/${fakeID}/pdf?type=invoice`,
      { maxRedirects: 0 },
    )
    expect(resp.status()).toBeGreaterThanOrEqual(400)
    expect(resp.status()).toBeLessThan(500)
    expect(resp.headers()["location"]).toBeFalsy()
  })

  test("GET /api/v1/me/invoices/{id}/pdf with a malformed UUID returns 400", async ({
    request,
  }) => {
    const apiBase = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8083"

    // Each entry below is a "look like an open-redirect payload"
    // string that an attacker might try to smuggle through the URL
    // path. The handler must reject all of them BEFORE reaching
    // the redirect step — they don't even parse as UUIDs.
    const malformedIDs = [
      "javascript:alert(1)",
      "..%2F..%2Fetc%2Fpasswd",
      "/etc/passwd",
      "https://evil.com/",
    ]
    for (const id of malformedIDs) {
      const resp = await request.get(`${apiBase}/api/v1/me/invoices/${id}/pdf`, {
        maxRedirects: 0,
      })
      expect(resp.status(), `malformed=${id}`).toBeGreaterThanOrEqual(400)
      expect(resp.status(), `malformed=${id}`).toBeLessThan(500)
      expect(resp.headers()["location"], `malformed=${id}`).toBeFalsy()
    }
  })
})
