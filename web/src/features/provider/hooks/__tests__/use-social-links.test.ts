import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import {
  useMySocialLinks,
  useUpsertSocialLink,
  useDeleteSocialLink,
} from "../use-social-links"

const mockGetMySocialLinks = vi.fn()
const mockGetPublicSocialLinks = vi.fn()
const mockUpsertSocialLink = vi.fn()
const mockDeleteSocialLink = vi.fn()

vi.mock("../../api/social-links-api", () => ({
  getMySocialLinks: (...args: unknown[]) => mockGetMySocialLinks(...args),
  getPublicSocialLinks: (...args: unknown[]) =>
    mockGetPublicSocialLinks(...args),
  upsertSocialLink: (...args: unknown[]) => mockUpsertSocialLink(...args),
  deleteSocialLink: (...args: unknown[]) => mockDeleteSocialLink(...args),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "test-user-id",
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  Wrapper.displayName = "TestWrapper"
  return Wrapper
}

describe("useMySocialLinks", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls getMySocialLinks API on mount", async () => {
    mockGetMySocialLinks.mockResolvedValue([])

    const { result } = renderHook(() => useMySocialLinks(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockGetMySocialLinks).toHaveBeenCalledOnce()
  })

  it("returns social links from API", async () => {
    mockGetMySocialLinks.mockResolvedValue([
      {
        id: "sl-1",
        platform: "github",
        url: "https://github.com/alice",
        created_at: "2026-03-20T10:00:00Z",
        updated_at: "2026-03-20T10:00:00Z",
      },
      {
        id: "sl-2",
        platform: "linkedin",
        url: "https://linkedin.com/in/alice",
        created_at: "2026-03-20T10:00:00Z",
        updated_at: "2026-03-20T10:00:00Z",
      },
    ])

    const { result } = renderHook(() => useMySocialLinks(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toHaveLength(2)
    expect(result.current.data?.[0].platform).toBe("github")
    expect(result.current.data?.[1].url).toBe("https://linkedin.com/in/alice")
  })

  it("returns empty array when no social links exist", async () => {
    mockGetMySocialLinks.mockResolvedValue([])

    const { result } = renderHook(() => useMySocialLinks(), {
      wrapper: createWrapper(),
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toHaveLength(0)
  })
})

describe("useUpsertSocialLink", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls upsertSocialLink API on mutate", async () => {
    mockUpsertSocialLink.mockResolvedValue(undefined)

    const { result } = renderHook(() => useUpsertSocialLink(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        platform: "github",
        url: "https://github.com/alice",
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockUpsertSocialLink).toHaveBeenCalledWith(
      "github",
      "https://github.com/alice",
    )
  })

  it("handles upsert failure", async () => {
    mockUpsertSocialLink.mockRejectedValue(new Error("Invalid URL"))

    const { result } = renderHook(() => useUpsertSocialLink(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate({
        platform: "github",
        url: "not-a-url",
      })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("Invalid URL")
  })
})

describe("useDeleteSocialLink", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("calls deleteSocialLink API on mutate", async () => {
    mockDeleteSocialLink.mockResolvedValue(undefined)

    const { result } = renderHook(() => useDeleteSocialLink(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("github")
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockDeleteSocialLink).toHaveBeenCalledWith("github")
  })

  it("handles delete failure", async () => {
    mockDeleteSocialLink.mockRejectedValue(new Error("Link not found"))

    const { result } = renderHook(() => useDeleteSocialLink(), {
      wrapper: createWrapper(),
    })

    await act(async () => {
      result.current.mutate("twitter")
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("Link not found")
  })
})
