import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useProfileSkills } from "../use-profile-skills"

const mockFetchProfileSkills = vi.fn()

vi.mock("../../api/skill-api", () => ({
  fetchProfileSkills: () => mockFetchProfileSkills(),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  return { queryClient, wrapper }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("useProfileSkills", () => {
  it("returns the skills fetched from the API", async () => {
    const data = [
      { skill_text: "react", display_text: "React", position: 0 },
      { skill_text: "vue", display_text: "Vue.js", position: 1 },
    ]
    mockFetchProfileSkills.mockResolvedValue(data)

    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useProfileSkills(), { wrapper })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(data)
    expect(mockFetchProfileSkills).toHaveBeenCalledTimes(1)
  })
})
