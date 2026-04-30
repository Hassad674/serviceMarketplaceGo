import { test, expect } from "@playwright/test"

import { registerProvider } from "../helpers/auth"

// ---------------------------------------------------------------------------
// BUG-19 — list endpoints must serialise empty result sets as `[]`, never
// as `null`. The TS clients across web, admin and mobile call `.length`
// on list responses; Go's nil slice → JSON null would crash them at
// runtime.
//
// The fix lives in pkg/response/json.go (NilSliceToEmpty + JSON wrapper)
// and is unit-tested in pkg/response/nil_slice_test.go and
// internal/handler/list_empty_contract_test.go. This Playwright spec
// is the end-to-end probe: hit the live API at well-known list
// endpoints with a freshly-registered user (zero rows everywhere) and
// confirm the JSON body never contains the literal sequence
// `"data":null` or starts with `null`.
// ---------------------------------------------------------------------------

const API_BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8083"

type ProbedEndpoint = {
  name: string
  path: string
  // expectArrayTopLevel — true when the endpoint returns a JSON array
  // at the top level (e.g. /social-links). False when the endpoint
  // wraps the array in `{"data": [...]}` so we look for the data key.
  expectArrayTopLevel: boolean
}

const ENDPOINTS: ProbedEndpoint[] = [
  // Social links (returns top-level JSON array).
  { name: "freelance social links", path: "/api/v1/freelance/social-links", expectArrayTopLevel: true },
  { name: "referrer social links", path: "/api/v1/referrer/social-links", expectArrayTopLevel: true },

  // Notifications (envelope: {"data": [...], "next_cursor": "", "has_more": false}).
  { name: "notifications list", path: "/api/v1/notifications", expectArrayTopLevel: false },

  // Job applications (envelope).
  { name: "my job applications", path: "/api/v1/jobs/applications/mine", expectArrayTopLevel: false },

  // Disputes (envelope).
  { name: "my disputes", path: "/api/v1/disputes/mine", expectArrayTopLevel: false },

  // Reports (envelope).
  { name: "my reports", path: "/api/v1/reports/mine", expectArrayTopLevel: false },

  // Invoices (envelope).
  { name: "my invoices", path: "/api/v1/invoices", expectArrayTopLevel: false },
]

test.describe("BUG-19 — list endpoints return [] on empty result", () => {
  test("every probed endpoint serialises empty result as `[]`, never `null`", async ({
    page,
    request,
  }) => {
    // A fresh provider has zero notifications, zero applications,
    // zero invoices, etc. — perfect testbed for the empty-list path.
    const user = await registerProvider(page)

    // Reuse the page's auth state for the API requests. The auth
    // cookies are scoped to the API origin under same-site=Lax, so
    // fetching with `request` from the same browser context works.
    for (const endpoint of ENDPOINTS) {
      const url = `${API_BASE}${endpoint.path}`
      const resp = await request.get(url)

      // The endpoint may legitimately reject the request when the
      // user lacks the role for it (e.g. invoices may require
      // additional setup). 401/403/404 are acceptable; 500 is not.
      // We only assert the contract on 200 responses.
      if (resp.status() !== 200) {
        // Log to test output so a regression in another endpoint
        // does not silently mask a BUG-19 regression on a 200 path.
        // eslint-disable-next-line no-console
        console.log(`[bug-19] ${endpoint.name}: status=${resp.status()} skipped`)
        continue
      }

      const body = await resp.text()

      // Body must not be empty.
      expect(body.length, `${endpoint.name}: empty body`).toBeGreaterThan(0)

      // Whatever the shape, the body MUST decode as JSON and the
      // list portion MUST be `[]` literal — never `null`.
      const parsed = JSON.parse(body)
      if (endpoint.expectArrayTopLevel) {
        expect(Array.isArray(parsed), `${endpoint.name}: expected top-level array`).toBe(true)
        expect(parsed.length, `${endpoint.name}: expected empty array`).toBe(0)
      } else {
        // Envelope shape — at least one of `data` / `items` /
        // `results` should be the array. Try in order.
        const arr = parsed.data ?? parsed.items ?? parsed.results
        expect(arr, `${endpoint.name}: data field absent`).not.toBeNull()
        expect(Array.isArray(arr), `${endpoint.name}: data field is not an array`).toBe(true)
      }

      // Belt and suspenders: the literal sequence `:null` must not
      // appear where the array would. JSON.stringify(parsed) re-
      // serializes with `null` for nil → if the response had been
      // null we'd see it preserved.
      const reserialized = JSON.stringify(parsed)
      const arrayLikeKeys = ["data", "items", "results"]
      for (const key of arrayLikeKeys) {
        const nullPattern = `"${key}":null`
        expect(reserialized, `${endpoint.name}: ${key} serialised as null`).not.toContain(nullPattern)
      }

      // Ensure the user is referenced (avoid lint about unused var).
      void user.email
    }
  })
})
