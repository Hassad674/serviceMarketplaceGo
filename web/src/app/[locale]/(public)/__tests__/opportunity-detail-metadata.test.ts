/**
 * opportunities/[id]/page generateMetadata tests — PERF-W-06.
 * Plus implicit coverage of the JSON-LD JobPosting export branch via
 * the page module's dynamic rendering path.
 *
 * Tests focus on the metadata export because that's the SEO primitive
 * Google reads first; JSON-LD is asserted via a separate snapshot at
 * the e2e layer.
 */

import { describe, it, expect, vi, beforeEach } from "vitest"

const fetchMock = vi.fn()
vi.mock("@/features/job/api/job-server", () => ({
  fetchJobForMetadata: (...args: unknown[]) => fetchMock(...args),
}))

vi.mock("@/features/job/components/opportunity-detail", () => ({
  OpportunityDetail: () => null,
}))

beforeEach(() => {
  fetchMock.mockReset()
})

async function callMetadata(id: string): Promise<Record<string, unknown>> {
  const mod = await import("../opportunities/[id]/page")
  return (await mod.generateMetadata({
    params: Promise.resolve({ id, locale: "en" }),
  })) as Record<string, unknown>
}

describe("opportunities/[id] generateMetadata — PERF-W-06", () => {
  it("uses job title for title + 160-char description", async () => {
    fetchMock.mockResolvedValue({
      id: "job-1",
      creator_id: "user-1",
      title: "Build a marketplace",
      description: "We need a backend Go developer for 6 weeks of work.",
      skills: ["go", "postgres"],
      applicant_type: "all",
      budget_type: "long_term",
      min_budget: 50,
      max_budget: 100,
      status: "open",
      created_at: "2026-04-01T00:00:00Z",
      updated_at: "2026-04-01T00:00:00Z",
      is_indefinite: false,
      description_type: "text",
    })

    const md = await callMetadata("job-1")
    expect(md.title as string).toBe("Build a marketplace | Marketplace Service")
    expect(md.description as string).toContain("backend Go developer")
    expect((md.alternates as { canonical: string }).canonical).toBe(
      "/opportunities/job-1",
    )
    expect((md.openGraph as { title: string }).title).toBe(
      "Build a marketplace | Marketplace Service",
    )
    expect((md.twitter as { card: string }).card).toBe("summary")
  })

  it("falls back to generic title when job not found", async () => {
    fetchMock.mockResolvedValue(null)
    const md = await callMetadata("missing")
    expect(md.title as string).toContain("Opportunity")
    expect((md.alternates as { canonical: string }).canonical).toBe(
      "/opportunities/missing",
    )
  })

  it("truncates description to 160 chars", async () => {
    const longDesc = "z".repeat(500)
    fetchMock.mockResolvedValue({
      id: "job-1",
      creator_id: "u",
      title: "T",
      description: longDesc,
      skills: [],
      applicant_type: "all",
      budget_type: "one_shot",
      min_budget: 0,
      max_budget: 0,
      status: "open",
      created_at: "",
      updated_at: "",
      is_indefinite: false,
      description_type: "text",
    })

    const md = await callMetadata("job-1")
    expect((md.description as string).length).toBe(160)
  })
})
