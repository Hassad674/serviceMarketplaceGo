import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  deleteFreelanceSocialLink,
  getMyFreelanceSocialLinks,
  getPublicFreelanceSocialLinks,
  upsertFreelanceSocialLink,
} from "../freelance-social-links-api"

// Verifies the freelance-social-links API hits the dedicated
// per-persona endpoints. Mocks `fetch` to stay free of network IO.

interface FetchCall {
  url: string
  init?: RequestInit
}

const calls: FetchCall[] = []
const originalFetch = globalThis.fetch

beforeEach(() => {
  calls.length = 0
  globalThis.fetch = vi.fn(async (url: RequestInfo | URL, init?: RequestInit) => {
    calls.push({ url: String(url), init })
    const body = init?.method === "DELETE" || init?.method === "PUT" ? null : "[]"
    const status = init?.method === "DELETE" || init?.method === "PUT" ? 204 : 200
    return new Response(body, {
      status,
      headers: { "Content-Type": "application/json" },
    })
  }) as typeof fetch
})

afterEach(() => {
  globalThis.fetch = originalFetch
})

describe("freelance-social-links-api", () => {
  it("getMyFreelanceSocialLinks GETs the scoped path", async () => {
    await getMyFreelanceSocialLinks()
    expect(calls).toHaveLength(1)
    expect(calls[0].url).toContain("/api/v1/freelance-profile/social-links")
    expect(calls[0].init?.method ?? "GET").toBe("GET")
  })

  it("getPublicFreelanceSocialLinks GETs the public path with the org id", async () => {
    await getPublicFreelanceSocialLinks("org-123")
    expect(calls[0].url).toContain("/api/v1/freelance-profiles/org-123/social-links")
  })

  it("upsertFreelanceSocialLink PUTs a body payload", async () => {
    await upsertFreelanceSocialLink("github", "https://github.com/u")
    expect(calls[0].url).toContain("/api/v1/freelance-profile/social-links")
    expect(calls[0].init?.method).toBe("PUT")
    expect(calls[0].init?.body).toBe(
      JSON.stringify({ platform: "github", url: "https://github.com/u" }),
    )
  })

  it("deleteFreelanceSocialLink DELETEs the platform path segment", async () => {
    await deleteFreelanceSocialLink("linkedin")
    expect(calls[0].url).toContain(
      "/api/v1/freelance-profile/social-links/linkedin",
    )
    expect(calls[0].init?.method).toBe("DELETE")
  })
})
