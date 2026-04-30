import { describe, it, expect, vi, beforeEach } from "vitest"
import { renderHook, waitFor, act } from "@testing-library/react"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { createElement } from "react"
import { useOrganizationShared, organizationSharedQueryKey } from "../use-organization-shared"
import { useUpdateOrganizationLocation, invalidateSharedDependents } from "../use-update-organization-location"
import { useUpdateOrganizationLanguages } from "../use-update-organization-languages"
import { useUploadOrganizationPhoto } from "../use-update-organization-photo"

const mockGet = vi.fn()
const mockUpdateLocation = vi.fn()
const mockUpdateLanguages = vi.fn()
const mockUpdatePhoto = vi.fn()
const mockUploadPhoto = vi.fn()

vi.mock("../../api/organization-shared-api", () => ({
  getOrganizationShared: () => mockGet(),
  updateOrganizationLocation: (...a: unknown[]) => mockUpdateLocation(...a),
  updateOrganizationLanguages: (...a: unknown[]) => mockUpdateLanguages(...a),
  updateOrganizationPhoto: (...a: unknown[]) => mockUpdatePhoto(...a),
}))

vi.mock("../../api/photo-upload-api", () => ({
  uploadOrganizationPhoto: (...a: unknown[]) => mockUploadPhoto(...a),
}))

vi.mock("@/shared/hooks/use-current-user-id", () => ({
  useCurrentUserId: () => "uid-1",
}))

const profile = {
  photo_url: "https://x/p.jpg",
  city: "Paris",
  country_code: "FR",
  latitude: 48.8,
  longitude: 2.3,
  work_mode: ["remote"] as const,
  travel_radius_km: null,
  languages_professional: ["fr"],
  languages_conversational: ["en"],
}

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 } },
  })
  const wrapper = ({ children }: { children: React.ReactNode }) =>
    createElement(QueryClientProvider, { client: queryClient }, children)
  return { queryClient, wrapper }
}

beforeEach(() => {
  vi.clearAllMocks()
  // Default sensible mock returns. Individual tests override these.
  mockGet.mockResolvedValue(profile)
  mockUpdateLocation.mockResolvedValue(profile)
  mockUpdateLanguages.mockResolvedValue(profile)
  mockUpdatePhoto.mockResolvedValue(profile)
  mockUploadPhoto.mockResolvedValue({ url: "https://default" })
})

describe("organizationSharedQueryKey", () => {
  it("scopes the key under user id", () => {
    expect(organizationSharedQueryKey("uid-1")).toEqual([
      "user",
      "uid-1",
      "organization-shared",
    ])
  })

  it("uses undefined uid when no current user", () => {
    expect(organizationSharedQueryKey(undefined)).toEqual([
      "user",
      undefined,
      "organization-shared",
    ])
  })
})

describe("useOrganizationShared", () => {
  it("returns the profile from the API", async () => {
    mockGet.mockResolvedValue(profile)
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useOrganizationShared(), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(result.current.data).toEqual(profile)
  })

  it("propagates errors", async () => {
    mockGet.mockRejectedValue(new Error("403"))
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useOrganizationShared(), { wrapper })
    await waitFor(() => expect(result.current.isError).toBe(true))
  })
})

describe("useUpdateOrganizationLocation", () => {
  it("writes location and invalidates dependents on success", async () => {
    const next = { ...profile, city: "Lyon" }
    mockUpdateLocation.mockResolvedValue(next)

    const { queryClient, wrapper } = createWrapper()
    const setSpy = vi.spyOn(queryClient, "setQueryData")
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries")

    const { result } = renderHook(() => useUpdateOrganizationLocation(), { wrapper })
    await act(async () => {
      await result.current.mutateAsync({
        city: "Lyon",
        country_code: "FR",
        latitude: 45.7,
        longitude: 4.8,
        work_mode: ["hybrid"],
        travel_radius_km: 100,
      })
    })

    expect(mockUpdateLocation).toHaveBeenCalledWith({
      city: "Lyon",
      country_code: "FR",
      latitude: 45.7,
      longitude: 4.8,
      work_mode: ["hybrid"],
      travel_radius_km: 100,
    })
    expect(setSpy).toHaveBeenCalledWith(
      organizationSharedQueryKey("uid-1"),
      next,
    )
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["user", "uid-1", "freelance-profile"],
    })
    expect(invalidateSpy).toHaveBeenCalledWith({
      queryKey: ["user", "uid-1", "referrer-profile"],
    })
  })
})

describe("invalidateSharedDependents", () => {
  it("invalidates each dependent prefix once", () => {
    const queryClient = new QueryClient()
    const spy = vi.spyOn(queryClient, "invalidateQueries")
    invalidateSharedDependents(queryClient, "uid-7")
    expect(spy).toHaveBeenCalledWith({
      queryKey: ["user", "uid-7", "freelance-profile"],
    })
    expect(spy).toHaveBeenCalledWith({
      queryKey: ["user", "uid-7", "referrer-profile"],
    })
  })
})

describe("useUpdateOrganizationLanguages", () => {
  it("PUTs languages and updates the shared cache", async () => {
    const next = { ...profile, languages_professional: ["en"] }
    mockUpdateLanguages.mockResolvedValue(next)
    const { queryClient, wrapper } = createWrapper()
    const setSpy = vi.spyOn(queryClient, "setQueryData")

    const { result } = renderHook(() => useUpdateOrganizationLanguages(), { wrapper })

    await act(async () => {
      await result.current.mutateAsync({
        professional: ["en"],
        conversational: ["fr"],
      })
    })

    expect(setSpy).toHaveBeenCalledWith(
      organizationSharedQueryKey("uid-1"),
      next,
    )
  })

  it("propagates errors", async () => {
    mockUpdateLanguages.mockRejectedValue(new Error("422"))
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useUpdateOrganizationLanguages(), { wrapper })

    await expect(
      result.current.mutateAsync({ professional: [], conversational: [] }),
    ).rejects.toThrow("422")
  })
})

describe("useUploadOrganizationPhoto", () => {
  it("orchestrates upload + PUT in one mutation and writes the cache", async () => {
    mockUploadPhoto.mockResolvedValue({ url: "https://cdn/img.jpg" })
    const next = { ...profile, photo_url: "https://cdn/img.jpg" }
    mockUpdatePhoto.mockResolvedValue(next)

    const { queryClient, wrapper } = createWrapper()
    const setSpy = vi.spyOn(queryClient, "setQueryData")
    const { result } = renderHook(() => useUploadOrganizationPhoto(), { wrapper })

    const fakeFile = new File(["x"], "p.jpg", { type: "image/jpeg" })
    await act(async () => {
      await result.current.mutateAsync(fakeFile)
    })

    expect(mockUploadPhoto).toHaveBeenCalledWith(fakeFile)
    expect(mockUpdatePhoto).toHaveBeenCalledWith({
      photo_url: "https://cdn/img.jpg",
    })
    expect(setSpy).toHaveBeenCalledWith(
      organizationSharedQueryKey("uid-1"),
      next,
    )
  })

  it("propagates upload errors without calling PUT", async () => {
    mockUploadPhoto.mockRejectedValue(new Error("upload-failed"))
    const { wrapper } = createWrapper()
    const { result } = renderHook(() => useUploadOrganizationPhoto(), { wrapper })

    const fakeFile = new File(["x"], "p.jpg", { type: "image/jpeg" })
    await expect(result.current.mutateAsync(fakeFile)).rejects.toThrow(
      "upload-failed",
    )
    expect(mockUpdatePhoto).not.toHaveBeenCalled()
  })
})
