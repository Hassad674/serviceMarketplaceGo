import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useUpdateClientProfile } from "../use-update-client-profile"

const mockUpdateClientProfile = vi.fn()

vi.mock("../../api/client-profile-api", () => ({
  updateClientProfile: (...args: unknown[]) => mockUpdateClientProfile(...args),
}))

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false, gcTime: 0 },
      mutations: { retry: false },
    },
  })
  const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")
  const Wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  return { Wrapper, invalidateSpy }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe("useUpdateClientProfile", () => {
  it("calls updateClientProfile with the payload", async () => {
    mockUpdateClientProfile.mockResolvedValue(undefined)
    const { Wrapper } = createWrapper()

    const { result } = renderHook(() => useUpdateClientProfile(), {
      wrapper: Wrapper,
    })

    await act(async () => {
      result.current.mutate({
        company_name: "Acme",
        client_description: "We ship software.",
      })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(mockUpdateClientProfile).toHaveBeenCalledWith({
      company_name: "Acme",
      client_description: "We ship software.",
    })
  })

  it("invalidates the client-profile, session and public profile caches on success", async () => {
    mockUpdateClientProfile.mockResolvedValue(undefined)
    const { Wrapper, invalidateSpy } = createWrapper()

    const { result } = renderHook(() => useUpdateClientProfile(), {
      wrapper: Wrapper,
    })

    await act(async () => {
      result.current.mutate({ client_description: "Updated" })
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    // Cross-check that every cache surface that mirrors the client
    // profile got invalidated — otherwise stale avatars / descriptions
    // would linger until the next hard refresh.
    const keysInvalidated = invalidateSpy.mock.calls.map((c) => c[0])
    expect(keysInvalidated).toContainEqual({ queryKey: ["client-profile"] })
    expect(keysInvalidated).toContainEqual({ queryKey: ["session"] })
    expect(keysInvalidated).toContainEqual({
      queryKey: ["public-client-profile"],
    })
    // The predicate-based call exists for cross-feature provider
    // profile caches (`["user", uid, "profile"]`).
    expect(
      invalidateSpy.mock.calls.some(
        (call) => typeof (call[0] as { predicate?: unknown }).predicate === "function",
      ),
    ).toBe(true)
  })

  it("surfaces errors from the API", async () => {
    mockUpdateClientProfile.mockRejectedValue(new Error("Forbidden"))
    const { Wrapper } = createWrapper()

    const { result } = renderHook(() => useUpdateClientProfile(), {
      wrapper: Wrapper,
    })

    await act(async () => {
      result.current.mutate({ client_description: "Too long" })
    })

    await waitFor(() => expect(result.current.isError).toBe(true))
    expect(result.current.error?.message).toBe("Forbidden")
  })
})
