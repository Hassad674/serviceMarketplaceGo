import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  getPublicProfile,
  searchProfiles,
  type PublicProfileSummary,
  type SearchResponse,
  type SearchType,
} from "../search-api"

// search-api thin-wraps `apiClient` so we mock the typed client to
// keep the test focused on URL construction, query-string handling
// and pass-through of the parsed response.
vi.mock("@/shared/lib/api-client", () => ({
  apiClient: vi.fn(),
}))

import { apiClient } from "@/shared/lib/api-client"

const mockedApiClient = vi.mocked(apiClient)

const SAMPLE_PROFILE: PublicProfileSummary = {
  organization_id: "org-1",
  owner_user_id: "user-1",
  name: "Acme",
  org_type: "agency",
  title: "Web Studio",
  photo_url: "https://cdn/acme.jpg",
  referrer_enabled: false,
  average_rating: 4.6,
  review_count: 12,
  skills: [{ skill_text: "react", display_text: "React" }],
  city: "Paris",
  country_code: "FR",
  work_mode: ["remote", "hybrid"],
  languages_professional: ["en", "fr"],
  availability_status: "available_now",
  pricing: [
    {
      kind: "direct",
      type: "daily",
      min_amount: 50000,
      max_amount: null,
      currency: "EUR",
      note: "",
      negotiable: true,
    },
  ],
}

describe("shared/lib/search/search-api", () => {
  beforeEach(() => {
    mockedApiClient.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe("searchProfiles", () => {
    it.each<SearchType>(["freelancer", "agency", "enterprise", "referrer"])(
      "calls /api/v1/profiles/search with the type=%s query parameter",
      async (type) => {
        const fakeResponse: SearchResponse = {
          data: [],
          next_cursor: "",
          has_more: false,
        }
        mockedApiClient.mockResolvedValue(fakeResponse)

        await searchProfiles(type)

        expect(mockedApiClient).toHaveBeenCalledOnce()
        const url = mockedApiClient.mock.calls[0][0] as string
        expect(url).toContain("/api/v1/profiles/search?")
        expect(url).toContain(`type=${type}`)
        // No cursor parameter when not provided.
        expect(url).not.toContain("cursor=")
      },
    )

    it("appends the cursor query param when provided", async () => {
      mockedApiClient.mockResolvedValue({
        data: [],
        next_cursor: "",
        has_more: false,
      })

      await searchProfiles("freelancer", "abc-cursor")

      const url = mockedApiClient.mock.calls[0][0] as string
      expect(url).toContain("type=freelancer")
      expect(url).toContain("cursor=abc-cursor")
    })

    it("returns the parsed SearchResponse with the full set of fields", async () => {
      const expected: SearchResponse = {
        data: [SAMPLE_PROFILE],
        next_cursor: "next-page-token",
        has_more: true,
      }
      mockedApiClient.mockResolvedValue(expected)

      const result = await searchProfiles("agency")

      expect(result).toEqual(expected)
      expect(result.data[0]).toEqual(SAMPLE_PROFILE)
      expect(result.has_more).toBe(true)
    })

    it("propagates errors thrown by apiClient", async () => {
      mockedApiClient.mockRejectedValueOnce(new Error("network down"))

      await expect(searchProfiles("freelancer")).rejects.toThrow(
        "network down",
      )
    })

    it("URL-encodes a cursor that contains reserved characters", async () => {
      mockedApiClient.mockResolvedValue({
        data: [],
        next_cursor: "",
        has_more: false,
      })

      await searchProfiles("agency", "cursor with spaces&extra=stuff")

      const url = mockedApiClient.mock.calls[0][0] as string
      // URLSearchParams encodes spaces as `+` and `&` as `%26`.
      expect(url).toContain("cursor=cursor+with+spaces%26extra%3Dstuff")
    })
  })

  describe("getPublicProfile", () => {
    it("calls /api/v1/profiles/:orgId and returns the parsed PublicProfileSummary", async () => {
      mockedApiClient.mockResolvedValue(SAMPLE_PROFILE)

      const result = await getPublicProfile("org-42")

      expect(mockedApiClient).toHaveBeenCalledWith("/api/v1/profiles/org-42")
      expect(result).toEqual(SAMPLE_PROFILE)
    })

    it("propagates errors when the org id cannot be fetched", async () => {
      mockedApiClient.mockRejectedValueOnce(new Error("not found"))

      await expect(getPublicProfile("bad-id")).rejects.toThrow("not found")
    })
  })

  describe("type contract — basic shape sanity", () => {
    it("exports the SearchType union with the four marketplace personas", () => {
      // Compile-time check via assignment — if any persona disappears
      // this test stops compiling.
      const types: SearchType[] = [
        "freelancer",
        "agency",
        "enterprise",
        "referrer",
      ]
      expect(types).toHaveLength(4)
    })
  })
})
