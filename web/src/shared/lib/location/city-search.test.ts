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

  it("filters out BAN features missing coordinates or city name", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          // valid
          {
            geometry: { coordinates: [2.35, 48.85] },
            properties: { name: "Paris", city: "Paris", postcode: "75001" },
          },
          // missing coords -> dropped
          {
            geometry: null,
            properties: { name: "NoGeo", city: "NoGeo" },
          },
          // 1-element coords -> dropped
          {
            geometry: { coordinates: [1] },
            properties: { name: "BadGeo", city: "BadGeo" },
          },
          // missing name -> dropped
          {
            geometry: { coordinates: [0, 0] },
            properties: { postcode: "00000" },
          },
        ],
      }),
    )
    const controller = new AbortController()
    const results = await searchCities("Pa", "FR", controller.signal)
    expect(results).toHaveLength(1)
    expect(results[0].city).toBe("Paris")
  })

  it("renders a BAN postcode-less context when neither postcode nor context are provided", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [2, 48] },
            // no postcode, no context — must render context as ""
            properties: { name: "Anyplace", city: "Anyplace" },
          },
        ],
      }),
    )
    const controller = new AbortController()
    const results = await searchCities("An", "FR", controller.signal)
    expect(results).toHaveLength(1)
    expect(results[0].context).toBe("")
    expect(results[0].postcode).toBe("")
  })

  it("falls back to props.name when props.city is absent", async () => {
    // Some BAN features only carry `name`, not `city` (e.g. arrondissements).
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [2, 48] },
            properties: { name: "Lyon 1er Arrondissement", postcode: "69001" },
          },
        ],
      }),
    )
    const controller = new AbortController()
    const results = await searchCities("Lyo", "FR", controller.signal)
    expect(results[0].city).toBe("Lyon 1er Arrondissement")
  })

  it("treats an empty country code as France and queries BAN", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [2.35, 48.85] },
            properties: { name: "Paris", city: "Paris", postcode: "75001" },
          },
        ],
      }),
    )
    const controller = new AbortController()
    await searchCities("Par", "", controller.signal)
    const calledUrl = fetchMock.mock.calls[0][0] as string
    expect(calledUrl).toContain("api-adresse.data.gouv.fr")
  })

  it("filters Photon results to the requested country code", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [13.4, 52.5] },
            properties: {
              name: "Berlin",
              country: "Allemagne",
              countrycode: "DE",
              osm_value: "city",
            },
          },
          // Different country — must be dropped because user asked for DE.
          {
            geometry: { coordinates: [2.3, 48.8] },
            properties: {
              name: "Berlin (Paris)",
              country: "France",
              countrycode: "FR",
              osm_value: "city",
            },
          },
        ],
      }),
    )
    const controller = new AbortController()
    const results = await searchCities("Berlin", "DE", controller.signal)
    expect(results).toHaveLength(1)
    expect(results[0].countryCode).toBe("DE")
  })

  it("retains Photon results that have no countrycode (treated as agnostic)", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [13.4, 52.5] },
            properties: {
              name: "Limbo",
              osm_value: "city",
            },
          },
        ],
      }),
    )
    const controller = new AbortController()
    const results = await searchCities("Lim", "DE", controller.signal)
    expect(results).toHaveLength(1)
    expect(results[0].countryCode).toBe("")
  })

  it("filters out Photon features that are not city-like (offices, buildings)", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [0, 0] },
            properties: {
              name: "ACME HQ",
              country: "DE",
              countrycode: "DE",
              osm_value: "office", // not city-like
            },
          },
        ],
      }),
    )
    const controller = new AbortController()
    const results = await searchCities("ACM", "DE", controller.signal)
    expect(results).toHaveLength(0)
  })

  it("accepts a Photon hamlet/village/borough as a city-like result", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [10, 47] },
            properties: {
              name: "Tinytown",
              country: "Suisse",
              countrycode: "CH",
              osm_value: "hamlet",
            },
          },
        ],
      }),
    )
    const controller = new AbortController()
    const results = await searchCities("Tin", "CH", controller.signal)
    expect(results[0].city).toBe("Tinytown")
  })

  it("trims whitespace before the minimum-length check", async () => {
    const controller = new AbortController()
    const results = await searchCities("   ", "FR", controller.signal)
    expect(results).toHaveLength(0)
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it("handles a BAN feature whose name is empty by dropping it", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [0, 0] },
            properties: { name: "", city: "" },
          },
        ],
      }),
    )
    const controller = new AbortController()
    const results = await searchCities("Pa", "FR", controller.signal)
    expect(results).toHaveLength(0)
  })

  it("produces a Photon result with no postcode when the upstream omits it", async () => {
    fetchMock.mockResolvedValueOnce(
      mockResponse({
        features: [
          {
            geometry: { coordinates: [2, 48] },
            properties: {
              name: "Foo",
              countrycode: "DE",
              osm_value: "city",
            },
          },
        ],
      }),
    )
    const controller = new AbortController()
    const results = await searchCities("Foo", "DE", controller.signal)
    expect(results[0].postcode).toBe("")
  })
})
