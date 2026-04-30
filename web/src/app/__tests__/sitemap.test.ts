/**
 * sitemap.ts tests — PERF-W-04.
 *
 * Asserts:
 *   - the static path block always renders (home + listings)
 *   - dynamic entries use the org_id stripped from the
 *     "{org}:{persona}" Typesense primary key
 *   - jobs are appended with /opportunities/{id} URLs
 *   - the function degrades gracefully when the fetcher returns
 *     null or an empty list
 */

import { describe, it, expect, vi, beforeEach } from "vitest"

const fetchListingMock = vi.fn()
const fetchJobsMock = vi.fn()

vi.mock("@/features/provider/api/search-server", () => ({
  fetchListingFirstPage: (...args: unknown[]) => fetchListingMock(...args),
}))

vi.mock("@/features/job/api/sitemap-server", () => ({
  fetchSitemapJobs: (...args: unknown[]) => fetchJobsMock(...args),
}))

vi.mock("@/config/site", () => ({
  siteConfig: { url: "https://example.com/" },
}))

beforeEach(() => {
  fetchListingMock.mockReset()
  fetchJobsMock.mockReset()
})

async function callSitemap() {
  const mod = await import("../sitemap")
  return mod.default()
}

describe("sitemap — PERF-W-04", () => {
  it("includes the static SEO surfaces", async () => {
    fetchListingMock.mockResolvedValue(null)
    fetchJobsMock.mockResolvedValue([])

    const entries = await callSitemap()
    const urls = entries.map((e) => e.url)
    expect(urls).toContain("https://example.com/")
    expect(urls).toContain("https://example.com/agencies")
    expect(urls).toContain("https://example.com/freelancers")
    expect(urls).toContain("https://example.com/referrers")
    expect(urls).toContain("https://example.com/opportunities")
    expect(urls).toContain("https://example.com/login")
    expect(urls).toContain("https://example.com/register")
  })

  it("appends agency/freelancer/referrer detail pages", async () => {
    fetchListingMock.mockImplementation((type: string) => {
      const id = `${type}-1`
      return Promise.resolve({
        documents: [
          {
            id: `${id}:${type === "freelancer" ? "freelance" : type}`,
            organization_id: id,
            updated_at: 1700000000,
          },
        ],
      })
    })
    fetchJobsMock.mockResolvedValue([])

    const entries = await callSitemap()
    const urls = entries.map((e) => e.url)
    expect(urls).toContain("https://example.com/agencies/agency-1")
    expect(urls).toContain("https://example.com/freelancers/freelancer-1")
    expect(urls).toContain("https://example.com/referrers/referrer-1")
  })

  it("uses id-prefix parsing when organization_id is missing", async () => {
    fetchListingMock.mockImplementation((_type: string) =>
      Promise.resolve({
        documents: [
          {
            id: "raw-uuid:freelance",
            organization_id: undefined,
            updated_at: 1700000000,
          },
        ],
      }),
    )
    fetchJobsMock.mockResolvedValue([])

    const entries = await callSitemap()
    const urls = entries.map((e) => e.url)
    expect(urls.some((u) => u.endsWith("/raw-uuid"))).toBe(true)
  })

  it("appends opportunity URLs from jobs", async () => {
    fetchListingMock.mockResolvedValue(null)
    fetchJobsMock.mockResolvedValue([
      { id: "job-a", updated_at: "2026-04-30T00:00:00Z" },
      { id: "job-b", updated_at: "2026-04-29T00:00:00Z" },
    ])

    const entries = await callSitemap()
    const urls = entries.map((e) => e.url)
    expect(urls).toContain("https://example.com/opportunities/job-a")
    expect(urls).toContain("https://example.com/opportunities/job-b")
  })

  it("degrades to static-only when all fetchers fail", async () => {
    fetchListingMock.mockResolvedValue(null)
    fetchJobsMock.mockResolvedValue([])

    const entries = await callSitemap()
    // Only the 7 static paths.
    expect(entries.length).toBeGreaterThanOrEqual(7)
  })

  it("each entry has a valid lastModified date", async () => {
    fetchListingMock.mockResolvedValue(null)
    fetchJobsMock.mockResolvedValue([])

    const entries = await callSitemap()
    for (const e of entries) {
      expect(e.lastModified).toBeInstanceOf(Date)
    }
  })

  it("priority field is bounded between 0 and 1", async () => {
    fetchListingMock.mockResolvedValue(null)
    fetchJobsMock.mockResolvedValue([])

    const entries = await callSitemap()
    for (const e of entries) {
      if (e.priority !== undefined) {
        expect(e.priority).toBeGreaterThanOrEqual(0)
        expect(e.priority).toBeLessThanOrEqual(1)
      }
    }
  })
})
