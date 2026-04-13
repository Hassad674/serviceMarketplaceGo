import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useUpdateProfileSkills } from "../use-update-profile-skills"
import { SKILLS_QUERY_KEY } from "../../constants"
import type { ProfileSkillResponse } from "../../types"

const mockUpdateProfileSkills = vi.fn()

vi.mock("../../api/skill-api", () => ({
  updateProfileSkills: (...args: unknown[]) =>
    mockUpdateProfileSkills(...args),
}))

function createClient() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 5 * 60 * 1000 },
      mutations: { retry: false },
    },
  })
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  return { queryClient, wrapper }
}

const baseSkills: ProfileSkillResponse[] = [
  { skill_text: "react", display_text: "React", position: 0 },
]

beforeEach(() => {
  vi.clearAllMocks()
})

describe("useUpdateProfileSkills", () => {
  it("optimistically updates the cache before the request resolves", async () => {
    const { queryClient, wrapper } = createClient()
    queryClient.setQueryData(SKILLS_QUERY_KEY.profile, baseSkills)

    let resolveMutation: () => void = () => {}
    mockUpdateProfileSkills.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          resolveMutation = resolve
        }),
    )

    const { result } = renderHook(() => useUpdateProfileSkills(), {
      wrapper,
    })

    act(() => {
      result.current.mutate(["react", "vue"])
    })

    await waitFor(() => {
      const cached = queryClient.getQueryData<ProfileSkillResponse[]>(
        SKILLS_QUERY_KEY.profile,
      )
      expect(cached?.map((s) => s.skill_text)).toEqual(["react", "vue"])
    })

    const cached = queryClient.getQueryData<ProfileSkillResponse[]>(
      SKILLS_QUERY_KEY.profile,
    )
    // Reused display_text from the previous snapshot for known items.
    expect(cached?.[0].display_text).toBe("React")
    // Falls back to skill_text for unknown items until refetch.
    expect(cached?.[1].display_text).toBe("vue")

    resolveMutation()
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
  })

  it("rolls back the cache to the previous value when the API fails", async () => {
    const { queryClient, wrapper } = createClient()
    queryClient.setQueryData(SKILLS_QUERY_KEY.profile, baseSkills)

    mockUpdateProfileSkills.mockRejectedValue(new Error("boom"))

    const { result } = renderHook(() => useUpdateProfileSkills(), {
      wrapper,
    })

    await act(async () => {
      result.current.mutate(["react", "vue"])
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    const cached = queryClient.getQueryData<ProfileSkillResponse[]>(
      SKILLS_QUERY_KEY.profile,
    )
    expect(cached?.map((s) => s.skill_text)).toEqual(["react"])
  })

  it("invalidates the profile skills query on settle", async () => {
    const { queryClient, wrapper } = createClient()
    queryClient.setQueryData(SKILLS_QUERY_KEY.profile, baseSkills)

    mockUpdateProfileSkills.mockResolvedValue(undefined)
    const spy = vi.spyOn(queryClient, "invalidateQueries")

    const { result } = renderHook(() => useUpdateProfileSkills(), {
      wrapper,
    })

    await act(async () => {
      result.current.mutate(["react", "vue"])
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(spy).toHaveBeenCalledWith({
      queryKey: SKILLS_QUERY_KEY.profile,
    })
  })
})
