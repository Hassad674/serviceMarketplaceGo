// Re-export of the canonical city autocomplete that lives in
// `shared/components/location/`. The brief wants every shared form
// primitive to be reachable under `shared/components/forms/` so the
// search filter and the profile location section both import from
// the same place — keeping the public API surface small and the
// shared/forms folder discoverable for new contributors.
//
// The underlying component is Photon-backed worldwide (with BAN as
// the FR-tuned fast path); see `shared/lib/location/city-search.ts`.
// Anything that wants just the search primitive should import from
// here; anything that needs the lower-level types stays free to
// reach into `shared/components/location/`.
export {
  CityAutocomplete,
  type CitySelection,
} from "@/shared/components/location/city-autocomplete"
