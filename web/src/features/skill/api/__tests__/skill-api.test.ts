import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"
import {
  createUserSkill,
  fetchCatalog,
  fetchProfileSkills,
  searchSkillsAutocomplete,
  updateProfileSkills,
} from "../skill-api"
import { ApiError } from "@/shared/lib/api-client"

// Mock global fetch instead of mocking apiClient, so we exercise the
// real error mapping path and the real URL composition of the api.
const fetchMock = vi.fn()

beforeEach(() => {
  vi.stubGlobal("fetch", fetchMock)
})

afterEach(() => {
  vi.unstubAllGlobals()
  fetchMock.mockReset()
})

function okJson<T>(payload: T): Response {
  return {
    ok: true,
    status: 200,
    json: async () => payload,
  } as unknown as Response
}

function errJson(status: number, code: string, message: string): Response {
  return {
    ok: false,
    status,
    json: async () => ({ error: code, message }),
  } as unknown as Response
}

describe("skill-api", () => {
  it("fetchProfileSkills calls GET /api/v1/profile/skills", async () => {
    const data = [
      { skill_text: "react", display_text: "React", position: 0 },
    ]
    fetchMock.mockResolvedValueOnce(okJson(data))

    const result = await fetchProfileSkills()

    expect(result).toEqual(data)
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/api/v1/profile/skills"),
      expect.objectContaining({ method: "GET" }),
    )
  })

  it("updateProfileSkills PUTs the skill_texts payload", async () => {
    fetchMock.mockResolvedValueOnce(okJson({ status: "ok" }))

    await updateProfileSkills(["react", "vue"])

    const call = fetchMock.mock.calls[0]
    expect(call[0]).toContain("/api/v1/profile/skills")
    expect(call[1].method).toBe("PUT")
    expect(JSON.parse(call[1].body as string)).toEqual({
      skill_texts: ["react", "vue"],
    })
  })

  it("fetchCatalog builds the query string with expertise and limit", async () => {
    fetchMock.mockResolvedValueOnce(
      okJson({ skills: [], total: 0 }),
    )

    await fetchCatalog("development", 25)

    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toContain("expertise=development")
    expect(url).toContain("limit=25")
  })

  it("searchSkillsAutocomplete builds the query string with q", async () => {
    fetchMock.mockResolvedValueOnce(okJson([]))

    await searchSkillsAutocomplete("rea", 10)

    const url = fetchMock.mock.calls[0][0] as string
    expect(url).toContain("q=rea")
    expect(url).toContain("limit=10")
  })

  it("createUserSkill POSTs display_text", async () => {
    fetchMock.mockResolvedValueOnce(
      okJson({
        skill_text: "my-skill",
        display_text: "My Skill",
        expertise_keys: [],
        is_curated: false,
        usage_count: 1,
      }),
    )

    const result = await createUserSkill("My Skill")

    expect(result.skill_text).toBe("my-skill")
    const call = fetchMock.mock.calls[0]
    expect(call[1].method).toBe("POST")
    expect(JSON.parse(call[1].body as string)).toEqual({
      display_text: "My Skill",
    })
  })

  it("maps backend error envelope to an ApiError", async () => {
    fetchMock.mockResolvedValueOnce(
      errJson(400, "too_many_skills", "Too many skills"),
    )

    await expect(updateProfileSkills(["x"])).rejects.toBeInstanceOf(ApiError)
  })
})
