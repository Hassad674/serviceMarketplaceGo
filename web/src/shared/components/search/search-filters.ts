// SearchFilters is the typed shape of the state the filter sidebar
// reports to its parent. It is intentionally Typesense-friendly: every
// field maps 1:1 to a future filter_by / query_by clause. The sidebar
// mutates this shape via a single onChange callback and the parent is
// free to forward it (unchanged) to TanStack Query once the backend
// wires real filters.
//
// Until the Typesense swap, the apply button is a no-op that logs to
// the console + the reset button re-initializes the state. No calls
// hit the current PostgreSQL endpoint, per the redesign brief.

import type { ExpertiseDomainKey } from "@/shared/lib/profile/expertise"

export type SearchAvailabilityFilter = "now" | "soon" | "all"
export type SearchWorkMode = "remote" | "on_site" | "hybrid"

export interface SearchFilters {
  availability: SearchAvailabilityFilter
  priceMin: number | null
  priceMax: number | null
  city: string
  countryCode: string
  radiusKm: number | null
  languages: string[]
  expertise: ExpertiseDomainKey[]
  skills: string[]
  minRating: number // 0..5
  workModes: SearchWorkMode[]
}

// EMPTY_SEARCH_FILTERS is the canonical "no filter" state: every
// scalar is null / empty, availability is "all", rating is 0. Used
// both as the default for useState and as the target of "reset".
export const EMPTY_SEARCH_FILTERS: SearchFilters = {
  availability: "all",
  priceMin: null,
  priceMax: null,
  city: "",
  countryCode: "",
  radiusKm: null,
  languages: [],
  expertise: [],
  skills: [],
  minRating: 0,
  workModes: [],
}

// isEmptyFilters returns true when the filters exactly match the
// canonical empty state — used by the sidebar to hide the "reset"
// button when there is nothing to clear.
export function isEmptyFilters(filters: SearchFilters): boolean {
  return (
    filters.availability === "all" &&
    filters.priceMin === null &&
    filters.priceMax === null &&
    filters.city === "" &&
    filters.countryCode === "" &&
    filters.radiusKm === null &&
    filters.languages.length === 0 &&
    filters.expertise.length === 0 &&
    filters.skills.length === 0 &&
    filters.minRating === 0 &&
    filters.workModes.length === 0
  )
}
