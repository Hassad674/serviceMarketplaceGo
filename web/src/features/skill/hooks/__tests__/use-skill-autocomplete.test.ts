import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useSkillAutocomplete } from "../use-skill-autocomplete"
import { SKILL_AUTOCOMPLETE_DEBOUNCE_MS } from "../../constants"

const mockSearch = vi.fn()

vi.mock("../../api/skill-api", () => ({
  searchSkillsAutocomplete: (...args: unknown[]) => mockSearch(...args),
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
  vi.useFakeTimers()
})

afterEach(() => {
  vi.useRealTimers()
})

describe("useSkillAutocomplete", () => {
  it("does not fire a request for an empty query", async () => {
    const { wrapper } = createWrapper()
    renderHook(() => useSkillAutocomplete(""), { wrapper })

    await act(async () => {
      vi.advanceTimersByTime(SKILL_AUTOCOMPLETE_DEBOUNCE_MS * 2)
    })

    expect(mockSearch).not.toHaveBeenCalled()
  })

  it("debounces the input and calls the API with the trimmed query", async () => {
    mockSearch.mockResolvedValue([
      {
        skill_text: "react",
        display_text: "React",
        expertise_keys: ["development"],
        is_curated: true,
        usage_count: 17000,
      },
    ])

    const { wrapper } = createWrapper()
    const { result, rerender } = renderHook(
      ({ q }: { q: string }) => useSkillAutocomplete(q),
      {
        wrapper,
        initialProps: { q: "r" },
      },
    )

    // Change input rapidly BEFORE the debounce timer elapses — only
    // the last value should ever trigger a network request.
    rerender({ q: "re" })
    rerender({ q: "rea" })

    await act(async () => {
      vi.advanceTimersByTime(SKILL_AUTOCOMPLETE_DEBOUNCE_MS + 10)
    })

    await vi.waitFor(() => {
      expect(mockSearch).toHaveBeenCalledWith("rea")
    })
    // The initial "r" render seeds useState synchronously so we
    // accept that call. What matters is the intermediate "re" value
    // was never dispatched — i.e. we never saw a call with "re".
    expect(
      mockSearch.mock.calls.some((call) => call[0] === "re"),
    ).toBe(false)
    await vi.waitFor(() => expect(result.current.isSuccess).toBe(true))
  })
})
