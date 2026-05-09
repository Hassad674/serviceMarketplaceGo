import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import { getMyProfileCompletion } from "../api/profile-completion-api"

// Mocks fetch so the suite is free of network IO. Mirrors the pattern
// used by the rest of the feature api tests.
const calls: Array<{ url: string; init?: RequestInit }> = []
const originalFetch = globalThis.fetch

beforeEach(() => {
  calls.length = 0
  globalThis.fetch = vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
    calls.push({ url: String(url), init })
    return new Response(
      JSON.stringify({
        role: "provider",
        persona: "freelance",
        percent: 50,
        total_sections: 10,
        filled_sections: 5,
        sections: [],
      }),
      { status: 200, headers: { "Content-Type": "application/json" } },
    )
  }) as typeof fetch
})

afterEach(() => {
  globalThis.fetch = originalFetch
})

describe("profile-completion-api", () => {
  it("getMyProfileCompletion GETs /api/v1/me/profile/completion", async () => {
    const report = await getMyProfileCompletion()

    expect(calls).toHaveLength(1)
    expect(calls[0].url).toContain("/api/v1/me/profile/completion")
    expect(calls[0].url).not.toContain("persona=")
    expect(report.persona).toBe("freelance")
    expect(report.percent).toBe(50)
  })

  it("forwards the persona override on the query string", async () => {
    await getMyProfileCompletion("referrer")

    expect(calls).toHaveLength(1)
    expect(calls[0].url).toContain(
      "/api/v1/me/profile/completion?persona=referrer",
    )
  })

  it("omits the query string when persona is undefined", async () => {
    await getMyProfileCompletion(undefined)

    expect(calls).toHaveLength(1)
    expect(calls[0].url).not.toContain("persona=")
  })
})
