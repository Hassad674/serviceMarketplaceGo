import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useSearchProfiles } from "../use-search"

const mockSearchProfiles = vi.fn()
const mockGetPublicProfile = vi.fn()

vi.mock("../../api/search-api", () => ({
  searchProfiles: (...args: unknown[]) => mockSearchProfiles(...args),
  getPublicProfile: (...args: unknown[]) => mockGetPublicProfile(...args),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
    },
  })
  return ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
}

describe("useSearchProfiles", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls searchProfiles API with freelancer type", async () => {
    mockSearchProfiles.mockResolvedValue({
      data: [],
      next_cursor: "",
      has_more: false,
    })

    const { result } = renderHook(() => useSearchProfiles("freelancer"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockSearchProfiles).toHaveBeenCalledWith("freelancer", undefined)
  })

  it("returns search results from API", async () => {
    mockSearchProfiles.mockResolvedValue({
      data: [
        {
          user_id: "user-1",
          display_name: "Alice Dupont",
          first_name: "Alice",
          last_name: "Dupont",
          role: "provider",
          title: "Full-Stack Developer",
          photo_url: "https://storage.example.com/photos/alice.jpg",
          referrer_enabled: false,
        },
        {
          user_id: "user-2",
          display_name: "Bob Martin",
          first_name: "Bob",
          last_name: "Martin",
          role: "provider",
          title: "UX Designer",
          photo_url: "",
          referrer_enabled: true,
        },
      ],
      next_cursor: "",
      has_more: false,
    })

    const { result } = renderHook(() => useSearchProfiles("freelancer"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const profiles = result.current.data?.pages.flatMap((p) => p.data) ?? []
    expect(profiles).toHaveLength(2)
    expect(profiles[0].display_name).toBe("Alice Dupont")
    expect(profiles[1].title).toBe("UX Designer")
  })

  it("searches with agency type", async () => {
    mockSearchProfiles.mockResolvedValue({
      data: [
        {
          user_id: "agency-1",
          display_name: "Tech Agency",
          first_name: "",
          last_name: "",
          role: "agency",
          title: "Digital Agency",
          photo_url: "",
          referrer_enabled: false,
        },
      ],
      next_cursor: "",
      has_more: false,
    })

    const { result } = renderHook(() => useSearchProfiles("agency"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockSearchProfiles).toHaveBeenCalledWith("agency", undefined)
    const profiles = result.current.data?.pages.flatMap((p) => p.data) ?? []
    expect(profiles[0].role).toBe("agency")
  })

  it("searches with referrer type", async () => {
    mockSearchProfiles.mockResolvedValue({
      data: [
        {
          user_id: "ref-1",
          display_name: "Claire Referrer",
          first_name: "Claire",
          last_name: "Referrer",
          role: "provider",
          title: "Business Consultant",
          photo_url: "",
          referrer_enabled: true,
        },
      ],
      next_cursor: "",
      has_more: false,
    })

    const { result } = renderHook(() => useSearchProfiles("referrer"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockSearchProfiles).toHaveBeenCalledWith("referrer", undefined)
    const profiles = result.current.data?.pages.flatMap((p) => p.data) ?? []
    expect(profiles[0].referrer_enabled).toBe(true)
  })

  it("returns empty array when no profiles match", async () => {
    mockSearchProfiles.mockResolvedValue({
      data: [],
      next_cursor: "",
      has_more: false,
    })

    const { result } = renderHook(() => useSearchProfiles("agency"), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const profiles = result.current.data?.pages.flatMap((p) => p.data) ?? []
    expect(profiles).toHaveLength(0)
  })
})
