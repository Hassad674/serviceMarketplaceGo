import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  deleteReferrerSocialLink,
  getMyReferrerSocialLinks,
  getPublicReferrerSocialLinks,
  upsertReferrerSocialLink,
} from "../referrer-social-links-api"

// Mirrors the freelance-social-links API test, targeting the
// referrer persona endpoints.

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

describe("referrer-social-links-api", () => {
  it("getMyReferrerSocialLinks GETs the scoped path", async () => {
    await getMyReferrerSocialLinks()
    expect(calls[0].url).toContain("/api/v1/referrer-profile/social-links")
    expect(calls[0].init?.method ?? "GET").toBe("GET")
  })

  it("getPublicReferrerSocialLinks GETs the public path with the org id", async () => {
    await getPublicReferrerSocialLinks("org-999")
    expect(calls[0].url).toContain("/api/v1/referrer-profiles/org-999/social-links")
  })

  it("upsertReferrerSocialLink PUTs a body payload", async () => {
    await upsertReferrerSocialLink("linkedin", "https://linkedin.com/in/u")
    expect(calls[0].url).toContain("/api/v1/referrer-profile/social-links")
    expect(calls[0].init?.method).toBe("PUT")
    expect(calls[0].init?.body).toBe(
      JSON.stringify({ platform: "linkedin", url: "https://linkedin.com/in/u" }),
    )
  })

  it("deleteReferrerSocialLink DELETEs the platform path segment", async () => {
    await deleteReferrerSocialLink("website")
    expect(calls[0].url).toContain(
      "/api/v1/referrer-profile/social-links/website",
    )
    expect(calls[0].init?.method).toBe("DELETE")
  })
})
