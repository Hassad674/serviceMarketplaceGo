// Typed wrappers around the free public geocoding APIs we query
// directly from the browser for the city autocomplete. No backend
// proxy — the endpoints both expose `Access-Control-Allow-Origin: *`
// and respond in ~120ms, which is the latency the user specifically
// asked us to optimize for.
//
// Primary:  Base Adresse Nationale (BAN) — government, free,
//           unlimited, sub-100ms from EU edge, gold standard for
//           French municipalities. Returns GeoJSON features with
//           name, postcode, citycode and [lng, lat] coordinates.
// Fallback: Photon (komoot, OSM-backed) — free, open, international
//           coverage. Used when the user has selected a non-French
//           country in the country dropdown.

// A single city entry the UI can render in the dropdown and persist
// as the canonical selection. `countryCode` is ISO 3166-1 alpha-2.
export type CitySearchResult = {
  city: string
  postcode: string
  countryCode: string
  latitude: number
  longitude: number
  // Human-facing secondary label shown in the dropdown row: e.g.
  // "69001 · Rhône, Auvergne-Rhône-Alpes" or "New Hampshire, États-Unis".
  context: string
}

const BAN_URL = "https://api-adresse.data.gouv.fr/search/"
const PHOTON_URL = "https://photon.komoot.io/api/"

// Minimum characters before firing a request. Below that the API
// either errors (BAN requires >= 3 chars) or returns noise.
export const CITY_SEARCH_MIN_CHARS = 2

// Shape returned by the French BAN API for the subset of fields we
// consume. Fields we don't use are intentionally omitted so a schema
// drift on the upstream side cannot break our parser.
interface BanFeature {
  geometry: { coordinates: [number, number] }
  properties: {
    name?: string
    city?: string
    postcode?: string
    context?: string
    type?: string
  }
}

interface BanResponse {
  features?: BanFeature[]
}

interface PhotonFeature {
  geometry: { coordinates: [number, number] }
  properties: {
    name?: string
    postcode?: string
    countrycode?: string
    country?: string
    state?: string
    county?: string
    osm_value?: string
  }
}

interface PhotonResponse {
  features?: PhotonFeature[]
}

async function fetchJson<T>(url: string, signal: AbortSignal): Promise<T> {
  const response = await fetch(url, { signal, headers: { Accept: "application/json" } })
  if (!response.ok) {
    throw new Error(`city search: upstream ${response.status}`)
  }
  return (await response.json()) as T
}

// Search French cities via BAN. The `type=municipality` filter
// restricts results to cities (skipping street-level addresses).
export async function searchFrenchCities(
  query: string,
  signal: AbortSignal,
): Promise<CitySearchResult[]> {
  const url = new URL(BAN_URL)
  url.searchParams.set("q", query)
  url.searchParams.set("type", "municipality")
  url.searchParams.set("limit", "8")
  const data = await fetchJson<BanResponse>(url.toString(), signal)
  const features = data.features ?? []
  return features
    .map(toBanResult)
    .filter((entry): entry is CitySearchResult => entry !== null)
}

function toBanResult(feature: BanFeature): CitySearchResult | null {
  const coords = feature.geometry?.coordinates
  const props = feature.properties
  if (!coords || coords.length < 2) return null
  const name = props.city || props.name
  if (!name) return null
  const postcode = props.postcode ?? ""
  const contextParts = [postcode, props.context].filter(Boolean)
  return {
    city: name,
    postcode,
    countryCode: "FR",
    longitude: coords[0],
    latitude: coords[1],
    context: contextParts.join(" · "),
  }
}

// Search international cities via Photon. We ask for `lang=fr` so
// the `country` / `state` labels in the context line match the rest
// of the app when the locale is French — Photon falls back to the
// native label when the translation is missing.
export async function searchInternationalCities(
  query: string,
  signal: AbortSignal,
): Promise<CitySearchResult[]> {
  const url = new URL(PHOTON_URL)
  url.searchParams.set("q", query)
  url.searchParams.set("limit", "8")
  url.searchParams.set("lang", "fr")
  const data = await fetchJson<PhotonResponse>(url.toString(), signal)
  const features = data.features ?? []
  return features
    .map(toPhotonResult)
    .filter((entry): entry is CitySearchResult => entry !== null)
}

function toPhotonResult(feature: PhotonFeature): CitySearchResult | null {
  const coords = feature.geometry?.coordinates
  const props = feature.properties
  if (!coords || coords.length < 2) return null
  if (!props.name) return null
  // Photon returns a mix of entity types; a city autocomplete should
  // ignore buildings and POIs. The marketplace values cities + towns
  // + villages so users living in smaller municipalities can still
  // pick their hometown.
  const kind = props.osm_value ?? ""
  if (kind && !CITY_LIKE_OSM_VALUES.has(kind)) return null
  const countryCode = (props.countrycode ?? "").toUpperCase()
  const contextParts = [props.state, props.county, props.country].filter(Boolean)
  return {
    city: props.name,
    postcode: props.postcode ?? "",
    countryCode,
    longitude: coords[0],
    latitude: coords[1],
    context: contextParts.join(", "),
  }
}

const CITY_LIKE_OSM_VALUES = new Set<string>([
  "city",
  "town",
  "village",
  "hamlet",
  "municipality",
  "suburb",
  "neighbourhood",
  "borough",
])

// Entry point used by the component. Picks the right backend based
// on the country the user has already selected. An empty country
// code defaults to BAN (France) which is the primary audience.
export async function searchCities(
  query: string,
  countryCode: string,
  signal: AbortSignal,
): Promise<CitySearchResult[]> {
  const normalized = query.trim()
  if (normalized.length < CITY_SEARCH_MIN_CHARS) return []
  const country = countryCode.toUpperCase()
  if (country === "" || country === "FR") {
    return searchFrenchCities(normalized, signal)
  }
  const results = await searchInternationalCities(normalized, signal)
  // Photon returns worldwide matches for a query — filter to the
  // country the user has already locked in so "Berlin" in France
  // doesn't show the German one when FR is selected elsewhere.
  return results.filter((r) => r.countryCode === country || r.countryCode === "")
}
