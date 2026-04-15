/**
 * feature-flag.ts wraps the NEXT_PUBLIC_SEARCH_ENGINE env var so
 * components can branch on the active search backend without
 * scattering string comparisons across the codebase.
 *
 * Default: "sql" — the legacy Postgres search path is the safe
 * fallback while phase 2 stabilises. Set to "typesense" in
 * `.env.local` to enable the Typesense path end-to-end.
 */

export type SearchEngine = "sql" | "typesense"

const RAW = process.env.NEXT_PUBLIC_SEARCH_ENGINE ?? "sql"

/** searchEngine returns the active search engine, normalised. */
export function searchEngine(): SearchEngine {
  return RAW.toLowerCase() === "typesense" ? "typesense" : "sql"
}

/** isTypesenseEnabled is the convenience boolean for components. */
export function isTypesenseEnabled(): boolean {
  return searchEngine() === "typesense"
}
