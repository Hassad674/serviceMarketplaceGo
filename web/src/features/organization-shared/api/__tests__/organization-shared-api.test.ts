import { describe, it, expect, vi, beforeEach } from "vitest"
import {
  getOrganizationShared,
  updateOrganizationLocation,
  updateOrganizationLanguages,
  updateOrganizationPhoto,
} from "../organization-shared-api"

const mockApiClient = vi.fn()

vi.mock("@/shared/lib/api-client", () => ({
  apiClient: (...args: unknown[]) => mockApiClient(...args),
}))

const profile = {
  photo_url: "https://x.com/p.jpg",
  city: "Paris",
  country_code: "FR",
  latitude: 48.8,
  longitude: 2.3,
  work_mode: ["remote"],
  travel_radius_km: 50,
  languages_professional: ["fr"],
  languages_conversational: ["en"],
}

beforeEach(() => {
  vi.clearAllMocks()
  mockApiClient.mockResolvedValue({ data: profile })
})

describe("organization-shared-api", () => {
  it("getOrganizationShared GETs and unwraps the envelope", async () => {
    const result = await getOrganizationShared()
    expect(mockApiClient).toHaveBeenCalledWith("/api/v1/organization/shared")
    expect(result).toEqual(profile)
  })

  it("updateOrganizationLocation PUTs the body and unwraps", async () => {
    const input = {
      city: "Lyon",
      country_code: "FR",
      latitude: 45.7,
      longitude: 4.8,
      work_mode: ["hybrid" as const],
      travel_radius_km: 100,
    }
    const result = await updateOrganizationLocation(input)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organization/location",
      { method: "PUT", body: input },
    )
    expect(result).toEqual(profile)
  })

  it("updateOrganizationLanguages PUTs and unwraps", async () => {
    const input = {
      professional: ["fr", "en"],
      conversational: ["es"],
    }
    const result = await updateOrganizationLanguages(input)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organization/languages",
      { method: "PUT", body: input },
    )
    expect(result).toEqual(profile)
  })

  it("updateOrganizationPhoto PUTs and unwraps", async () => {
    const input = { photo_url: "https://x.com/new.jpg" }
    const result = await updateOrganizationPhoto(input)
    expect(mockApiClient).toHaveBeenCalledWith(
      "/api/v1/organization/photo",
      { method: "PUT", body: input },
    )
    expect(result).toEqual(profile)
  })

  it("propagates apiClient rejections (network)", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("network"))
    await expect(getOrganizationShared()).rejects.toThrow("network")
  })

  it("propagates apiClient rejections on writes", async () => {
    mockApiClient.mockRejectedValueOnce(new Error("403"))
    await expect(
      updateOrganizationPhoto({ photo_url: "x" }),
    ).rejects.toThrow("403")
  })

  it("supports null lat/lng on location", async () => {
    await updateOrganizationLocation({
      city: "Remote",
      country_code: "FR",
      latitude: null,
      longitude: null,
      work_mode: ["remote"],
      travel_radius_km: null,
    })
    const body = (mockApiClient.mock.calls[0][1] as { body: { latitude: null } }).body
    expect(body.latitude).toBeNull()
  })
})
