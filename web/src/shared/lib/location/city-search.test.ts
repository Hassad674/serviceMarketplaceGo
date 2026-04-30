import { afterEach, beforeAll, describe, expect, it, vi } from "vitest"
import { searchCities, CITY_SEARCH_MIN_CHARS } from "./city-search"

const fetchMock = vi.fn()

function mockResponse(body: unknown, ok = true) {
  return {
    ok,
    status: ok ? 200 : 500,
    json: () => Promise.resolve(body),
  } as unknown as Response
}

beforeAll(() => {
  vi.stubGlobal("fetch", fetchMock)
})

afterEach(() => {
  fetchMock.mockReset()
})

describe("searchCities", () => {
  it("returns empty for queries shorter than the minimum", async () => {
    const controller = new AbortController()
    const results = await searchCities("a", "FR", controller.signal)
    expect(results).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
    expect(CITY_SEARCH_MIN_CHARS).toBe(2)
  })

  it("queries the French BAN endpoint for FR and maps the response", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [4.835, 45.758] },
            properties: {
              name: "Lyon",
              city: "Lyon",
              postcode: "69001",
              context: "69, Rhône, Auvergne-Rhône-Alpes",
              type: "municipality",
            },
          },
        ],
      }),
    )

    const controller = new AbortController()
    const results = await searchCities("Lyo", "FR", controller.signal)

    expect(fetchMock).toHaveBeenCalledOnce()
    const calledUrl = fetchMock.mock.calls[0][0] as string
    expect(calledUrl).toContain("api-adresse.data.gouv.fr")
    expect(calledUrl).toContain("type=municipality")
    expect(results).toHaveLength(1)
    expect(results[0]).toMatchObject({
      city: "Lyon",
      countryCode: "FR",
      postcode: "69001",
      latitude: 45.758,
      longitude: 4.835,
    })
  })

  it("routes international countries through Photon and filters city-like kinds", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [13.3951309, 52.5173885] },
            properties: {
              name: "Berlin",
              country: "Allemagne",
              countrycode: "DE",
              osm_value: "city",
            },
          },
          {
            geometry: { coordinates: [0, 0] },
            properties: {
              name: "Office",
              country: "Allemagne",
              countrycode: "DE",
              osm_value: "office",
            },
          },
        ],
      }),
    )

    const controller = new AbortController()
    const results = await searchCities("Berlin", "DE", controller.signal)

    const calledUrl = fetchMock.mock.calls[0][0] as string
    expect(calledUrl).toContain("photon.komoot.io")
    expect(results).toHaveLength(1)
    expect(results[0].city).toBe("Berlin")
    expect(results[0].countryCode).toBe("DE")
  })

  it("throws when the upstream API returns a non-ok status", async () => {
    fetchMock.mockResolvedValueOnce(mockResponse({}, false))
    const controller = new AbortController()
    await expect(searchCities("Paris", "FR", controller.signal)).rejects.toThrow(
      /upstream/,
    )
  })
})
