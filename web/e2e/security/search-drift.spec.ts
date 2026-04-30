import { test, expect } from "@playwright/test"

/**
 * BUG-05 — search index outbox.
 *
 * After a profile mutation commits, the matching `search.reindex`
 * pending event is committed in the same transaction. The outbox
 * worker then drains the event and pushes the document to Typesense
 * with at-least-once semantics. A Typesense outage cannot lose the
 * event because it lives in Postgres until processed.
 *
 * We can't easily simulate a Typesense outage from the e2e harness
 * without privileged docker access, so the test focuses on the
 * eventual-consistency contract: a profile mutation MUST eventually
 * be reflected in the search index. The 5-second budget matches the
 * typical worker latency under load.
 *
 * Skip when:
 *  - The backend has no auth (login fails) — we don't have a fixture
 *    user we can rely on in every environment.
 *  - The Typesense host isn't configured.
 */

const STRONG_PASSWORD = "SearchDriftPass1!"

test.describe("BUG-05 search index outbox", () => {
  test("profile update is eventually reflected in search results", async ({
    request,
  }) => {
    // Register a fresh provider so the test can mutate their profile
    // freely without cleanup concerns. Skip if registration is gated.
    const email = `drift-${Date.now()}@playwright.com`
    const reg = await request.post("/api/v1/auth/register", {
      data: {
        email,
        password: STRONG_PASSWORD,
        first_name: "Drift",
        last_name: "Tester",
        display_name: "Drift Tester",
        role: "provider",
      },
      headers: { "Content-Type": "application/json", "X-Auth-Mode": "token" },
      failOnStatusCode: false,
    })
    if (reg.status() !== 201 && reg.status() !== 200) {
      test.skip(
        true,
        `register returned ${reg.status()} — search drift e2e needs an open registration endpoint`,
      )
    }
    const { access_token: token } = await reg.json()
    expect(token).toBeTruthy()

    // Mutate the freelance profile with a unique title we can search
    // for. The title flows into Typesense via the outbox path.
    const uniqueTitle = `outbox-${Date.now()}-drift`
    const update = await request.put("/api/v1/freelance-profile/core", {
      data: {
        title: uniqueTitle,
        about: "Drift e2e about",
        video_url: "",
      },
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json",
      },
      failOnStatusCode: false,
    })
    expect(update.status()).toBeLessThan(500)
    if (update.status() >= 400) {
      test.skip(
        true,
        `freelance profile update returned ${update.status()} — outbox e2e needs a writable profile endpoint`,
      )
    }

    // Poll the public search endpoint for up to 5 seconds. The outbox
    // worker should drain the pending event and push the doc within
    // that window under normal load.
    const deadline = Date.now() + 5_000
    let foundCount = 0
    while (Date.now() < deadline) {
      const search = await request.get(
        `/api/v1/search/freelancers?q=${encodeURIComponent(uniqueTitle)}`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
          failOnStatusCode: false,
        },
      )
      if (search.status() === 200) {
        const body = await search.json()
        const items = body?.data ?? body?.results ?? []
        if (Array.isArray(items)) {
          foundCount = items.length
          if (foundCount > 0) break
        }
      }
      await new Promise((resolve) => setTimeout(resolve, 500))
    }

    expect(foundCount).toBeGreaterThan(0)
  })
})
