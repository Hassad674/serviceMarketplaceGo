/**
 * Avatar-refresh fix (2026-05-09).
 *
 * Uploading a photo via /profile (freelance) or /referral (referrer)
 * goes through `useUploadOrganizationPhoto` — the two-step pipeline
 * that uploads to MinIO/R2 then PUT-stamps the URL onto the org row.
 * The fix added two new keys to the post-upload fan-out:
 *
 *   1. `["user", uid, "profile"]` — the legacy provider-profile cache
 *      that <UserAvatar> reads to render the sidebar + header avatars.
 *      Without this, those two surfaces stayed on the OLD photo (or
 *      a Portrait fallback) until the next manual refresh.
 *   2. `["user", uid, "profile-completion"]` (every persona variant)
 *      — flips the "Photo" section to filled in both the freelance
 *      and the referrer bars without waiting for the 30s staleTime.
 *
 * This regression guard freezes that contract.
 */

import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { createElement, type ReactNode } from "react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: vi.fn(() => "uid-1"),
}))

vi.mock("../../api/photo-upload-api", () => ({
  uploadOrganizationPhoto: vi.fn(async () => ({
    url: "https://cdn/x.png",
  })),
}))

vi.mock("../../api/organization-shared-api", () => ({
  updateOrganizationPhoto: vi.fn(async () => ({
    organization_id: "org-1",
    photo_url: "https://cdn/x.png",
    city: "",
    country_code: "",
    latitude: null,
    longitude: null,
    work_mode: [],
    travel_radius_km: null,
    languages_professional: [],
    languages_conversational: [],
    updated_at: "2026-05-09T00:00:00Z",
  })),
}))

import { useUploadOrganizationPhoto } from "../use-update-organization-photo"

function withClient() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  const wrapper = ({ children }: { children: ReactNode }) =>
    createElement(QueryClientProvider, { client }, children)
  return { client, wrapper }
}

describe("useUploadOrganizationPhoto — cache fan-out", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("invalidates the legacy provider profile cache after upload", async () => {
    const { client, wrapper } = withClient()
    const spy = vi.spyOn(client, "invalidateQueries")

    const { result } = renderHook(() => useUploadOrganizationPhoto(), {
      wrapper,
    })

    await result.current.mutateAsync(
      new File(["x"], "x.png", { type: "image/png" }),
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const calls = spy.mock.calls.map((c) => JSON.stringify(c[0]?.queryKey))
    expect(calls).toContain(JSON.stringify(["user", "uid-1", "profile"]))
  })

  it("invalidates every persona variant of the completion report", async () => {
    const { client, wrapper } = withClient()
    const spy = vi.spyOn(client, "invalidateQueries")

    const { result } = renderHook(() => useUploadOrganizationPhoto(), {
      wrapper,
    })

    await result.current.mutateAsync(
      new File(["x"], "x.png", { type: "image/png" }),
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const calls = spy.mock.calls.map((c) => JSON.stringify(c[0]?.queryKey))
    expect(calls).toContain(
      JSON.stringify(["user", "uid-1", "profile-completion"]),
    )
  })

  it("invalidates the client-profile facets (shared photo)", async () => {
    const { client, wrapper } = withClient()
    const spy = vi.spyOn(client, "invalidateQueries")

    const { result } = renderHook(() => useUploadOrganizationPhoto(), {
      wrapper,
    })

    await result.current.mutateAsync(
      new File(["x"], "x.png", { type: "image/png" }),
    )
    await waitFor(() => expect(result.current.isSuccess).toBe(true))

    const calls = spy.mock.calls.map((c) => JSON.stringify(c[0]?.queryKey))
    expect(calls).toContain(JSON.stringify(["client-profile"]))
    expect(calls).toContain(JSON.stringify(["public-client-profile"]))
  })
})
