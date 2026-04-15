/**
 * typesense-client.test.ts pins the wire format the browser sends
 * to Typesense via the scoped key. We mock global fetch so the
 * test stays hermetic and runs in the standard vitest sandbox.
 */

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  TypesenseSearchClient,
  type TypesenseSearchResponse,
} from "../typesense-client"

const SAMPLE_RESPONSE: TypesenseSearchResponse = {
  found: 1,
  out_of: 100,
  page: 1,
  per_page: 20,
  search_time_ms: 4,
  hits: [],
  facet_counts: [],
  request_params: { collection_name: "marketplace_actors", q: "*", per_page: 20 },
}

describe("TypesenseSearchClient", () => {
  let fetchMock: ReturnType<typeof vi.fn>

  beforeEach(() => {
    fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      json: async () => SAMPLE_RESPONSE,
    })
    vi.stubGlobal("fetch", fetchMock)
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it("rejects empty host or key in constructor", () => {
    expect(() => new TypesenseSearchClient("", "key")).toThrow(/host is required/)
    expect(() => new TypesenseSearchClient("http://localhost:8108", "")).toThrow(
      /scoped key is required/,
    )
  })

  it("strips trailing slash from host", async () => {
    const client = new TypesenseSearchClient("http://localhost:8108/", "test-key")
    await client.search("marketplace_actors", { q: "*", query_by: "display_name" })
    const calledUrl = fetchMock.mock.calls[0]?.[0] as string
    expect(calledUrl.startsWith("http://localhost:8108/collections/")).toBe(true)
    expect(calledUrl.startsWith("http://localhost:8108//")).toBe(false)
  })

  it("sends X-TYPESENSE-API-KEY header", async () => {
    const client = new TypesenseSearchClient("http://localhost:8108", "scoped-xyz")
    await client.search("marketplace_actors", { q: "*", query_by: "display_name" })
    const init = fetchMock.mock.calls[0]?.[1] as RequestInit
    expect((init.headers as Record<string, string>)["X-TYPESENSE-API-KEY"]).toBe(
      "scoped-xyz",
    )
  })

  it("encodes optional parameters when present", async () => {
    const client = new TypesenseSearchClient("http://localhost:8108", "key")
    await client.search("marketplace_actors", {
      q: "alice",
      query_by: "display_name,title",
      filter_by: "skills:[react]",
      facet_by: "skills",
      sort_by: "_text_match:desc",
      page: 2,
      per_page: 30,
      exclude_fields: "embedding",
      highlight_fields: "display_name",
      highlight_full_fields: "display_name",
      num_typos: "2,1",
      max_facet_values: 50,
    })
    const url = fetchMock.mock.calls[0]?.[0] as string
    expect(url).toContain("q=alice")
    expect(url).toContain("query_by=display_name%2Ctitle")
    expect(url).toContain("filter_by=skills%3A%5Breact%5D")
    expect(url).toContain("facet_by=skills")
    expect(url).toContain("sort_by=_text_match%3Adesc")
    expect(url).toContain("page=2")
    expect(url).toContain("per_page=30")
    expect(url).toContain("exclude_fields=embedding")
    expect(url).toContain("highlight_fields=display_name")
    expect(url).toContain("num_typos=2%2C1")
    expect(url).toContain("max_facet_values=50")
  })

  it("omits optional parameters when not set", async () => {
    const client = new TypesenseSearchClient("http://localhost:8108", "key")
    await client.search("marketplace_actors", { q: "*", query_by: "display_name" })
    const url = fetchMock.mock.calls[0]?.[0] as string
    expect(url).not.toContain("filter_by")
    expect(url).not.toContain("facet_by")
    expect(url).not.toContain("sort_by")
  })

  it("throws on non-2xx response", async () => {
    fetchMock.mockResolvedValueOnce({
      ok: false,
      status: 400,
      text: async () => '{"message":"bad filter"}',
    })
    const client = new TypesenseSearchClient("http://localhost:8108", "key")
    await expect(
      client.search("marketplace_actors", { q: "*", query_by: "display_name" }),
    ).rejects.toThrow(/typesense search failed: 400/)
  })

  it("returns parsed JSON on success", async () => {
    const client = new TypesenseSearchClient("http://localhost:8108", "key")
    const got = await client.search("marketplace_actors", {
      q: "*",
      query_by: "display_name",
    })
    expect(got).toEqual(SAMPLE_RESPONSE)
  })
})
