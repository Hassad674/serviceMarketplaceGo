/**
 * Avatar-refresh fix (2026-05-09).
 *
 * After a successful photo upload via /client-profile or the agency
 * /profile page, the legacy provider-profile cache MUST be
 * invalidated so the sidebar identity card and the header dropdown
 * (both rendered via <UserAvatar> which reads from `useProfile()`)
 * pick up the new URL without a manual refresh. Same goes for the
 * organization-shared row that surfaces the same photo URL on the
 * split-profile read paths.
 *
 * This regression guard freezes the invalidation contract so a
 * future refactor does not silently drop one of the keys.
 */

import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { createElement, type ReactNode } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: vi.fn(() => "uid-1"),
}))

vi.mock("@/shared/lib/upload-api", () => ({
  uploadPhoto: vi.fn(async () => ({ url: "https://cdn/example.png" })),
}))

import { useUploadPhoto } from "../use-upload-photo"

function withClient() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client }, children)
  return { client, wrapper }
}

describe("useUploadPhoto — cache fan-out", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("invalidates the provider profile cache after a successful upload", async () => {
    const { client, wrapper } = withClient()
    const spy = vi.spyOn(client, "invalidateQueries")

    const { result } = renderHook(() => useUploadPhoto(), { wrapper })

    const file = new File(["x"], "x.png", { type: "image/png" })
    await result.current.mutateAsync(file)

    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const calls = spy.mock.calls.map((c) => JSON.stringify(c[0]?.queryKey))
    expect(calls).toContain(JSON.stringify(["user", "uid-1", "profile"]))
  })

  it("invalidates the organization-shared cache after a successful upload", async () => {
    const { client, wrapper } = withClient()
    const spy = vi.spyOn(client, "invalidateQueries")

    const { result } = renderHook(() => useUploadPhoto(), { wrapper })
    await result.current.mutateAsync(new File(["x"], "x.png", { type: "image/png" }))
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const calls = spy.mock.calls.map((c) => JSON.stringify(c[0]?.queryKey))
    expect(calls).toContain(
      JSON.stringify(["user", "uid-1", "organization-shared"]),
    )
  })

  it("invalidates every persona variant of the completion report", async () => {
    const { client, wrapper } = withClient()
    const spy = vi.spyOn(client, "invalidateQueries")

    const { result } = renderHook(() => useUploadPhoto(), { wrapper })
    await result.current.mutateAsync(new File(["x"], "x.png", { type: "image/png" }))
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const calls = spy.mock.calls.map((c) => JSON.stringify(c[0]?.queryKey))
    expect(calls).toContain(
      JSON.stringify(["user", "uid-1", "profile-completion"]),
    )
  })

  it("invalidates the client-profile facets (shared photo)", async () => {
    const { client, wrapper } = withClient()
    const spy = vi.spyOn(client, "invalidateQueries")

    const { result } = renderHook(() => useUploadPhoto(), { wrapper })
    await result.current.mutateAsync(new File(["x"], "x.png", { type: "image/png" }))
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const calls = spy.mock.calls.map((c) => JSON.stringify(c[0]?.queryKey))
    expect(calls).toContain(JSON.stringify(["client-profile"]))
    expect(calls).toContain(JSON.stringify(["public-client-profile"]))
  })
})
