import type { CountryData, Region } from "country-region-data"

/**
 * Lazy-loaded country data for countries where Stripe requires a state/province field.
 * Each entry returns a Promise resolving to the CountryData tuple [name, code, regions].
 * Using dynamic imports keeps the initial bundle small — only the selected country is loaded.
 */
const COUNTRY_LOADERS: Record<string, () => Promise<CountryData>> = {
  US: () => import("country-region-data").then((m) => m.US),
  AU: () => import("country-region-data").then((m) => m.AU),
  CA: () => import("country-region-data").then((m) => m.CA),
  IN: () => import("country-region-data").then((m) => m.IN),
  BR: () => import("country-region-data").then((m) => m.BR),
  MX: () => import("country-region-data").then((m) => m.MX),
  MY: () => import("country-region-data").then((m) => m.MY),
  JP: () => import("country-region-data").then((m) => m.JP),
  TH: () => import("country-region-data").then((m) => m.TH),
  SG: () => import("country-region-data").then((m) => m.SG),
  IE: () => import("country-region-data").then((m) => m.IE),
  GB: () => import("country-region-data").then((m) => m.GB),
  NZ: () => import("country-region-data").then((m) => m.NZ),
  IT: () => import("country-region-data").then((m) => m.IT),
  ES: () => import("country-region-data").then((m) => m.ES),
}

export type StateOption = { code: string; name: string }

/** In-memory cache to avoid re-importing on every render. */
const stateCache = new Map<string, StateOption[]>()

/**
 * Loads the list of states/provinces for a given ISO country code.
 * Returns an empty array for countries not in the supported list.
 */
export async function getStatesForCountry(countryCode: string): Promise<StateOption[]> {
  const cached = stateCache.get(countryCode)
  if (cached) return cached

  const loader = COUNTRY_LOADERS[countryCode]
  if (!loader) return []

  const [, , regions] = await loader()
  const states = regions.map(([name, code]: Region) => ({ code, name }))
  stateCache.set(countryCode, states)
  return states
}

/** Returns true if the country has a known list of states/provinces. */
export function hasStates(countryCode: string): boolean {
  return countryCode in COUNTRY_LOADERS
}

/** Returns true if the field should render as a state dropdown. */
export function isStateField(labelKey: string, path: string): boolean {
  return (
    labelKey === "state" ||
    labelKey === "businessState" ||
    path.includes(".address.state")
  )
}
