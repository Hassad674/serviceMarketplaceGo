/**
 * typesense-client.ts is a thin wrapper around Typesense's
 * `/collections/:name/documents/search` REST endpoint, designed to
 * be called directly from the browser with a SCOPED API key (the
 * key contains the persona filter so a freelance key can never
 * reach an agency document).
 *
 * We deliberately hand-roll this instead of pulling the official
 * `typesense` SDK as a runtime dep on every page bundle. The
 * subset of the API we need is small, the JSON shape is stable,
 * and a 60-line wrapper keeps the public listing pages well below
 * the 200 KB initial-bundle budget.
 *
 * Usage:
 *
 * ```ts
 * const client = new TypesenseSearchClient("http://localhost:8108", scopedKey);
 * const result = await client.search("marketplace_actors", {
 *   q: "alice",
 *   query_by: "display_name,title,skills_text,city",
 *   filter_by: "skills:[react] && availability_status:[available_now]",
 *   per_page: 20,
 * });
 * ```
 */

/** SearchDocumentPersona is the discriminator stored on every doc. */
export type SearchDocumentPersona = "freelance" | "agency" | "referrer"

/**
 * RawSearchDocument mirrors the wire format Typesense returns. The
 * field set matches the backend `internal/search.SearchDocument`
 * Go struct so the frontend stays bit-for-bit compatible with the
 * collection schema seeded in phase 1.
 */
export interface RawSearchDocument {
  id: string
  persona: SearchDocumentPersona
  is_published: boolean
  display_name: string
  title?: string
  photo_url?: string
  city?: string
  country_code?: string
  location?: [number, number]
  work_mode: string[]
  languages_professional: string[]
  languages_conversational: string[]
  availability_status: string
  availability_priority: number
  expertise_domains: string[]
  skills: string[]
  skills_text: string
  pricing_type?: string
  pricing_min_amount?: number
  pricing_max_amount?: number
  pricing_currency?: string
  pricing_negotiable: boolean
  rating_average: number
  rating_count: number
  rating_score: number
  total_earned: number
  completed_projects: number
  profile_completion_score: number
  last_active_at: number
  response_rate: number
  is_verified: boolean
  is_top_rated: boolean
  is_featured: boolean
  created_at: number
  updated_at: number
}

/** TypesenseSearchParams is the typed shape of every supported query parameter. */
export interface TypesenseSearchParams {
  q: string
  query_by: string
  filter_by?: string
  facet_by?: string
  sort_by?: string
  page?: number
  per_page?: number
  exclude_fields?: string
  highlight_fields?: string
  highlight_full_fields?: string
  num_typos?: string
  max_facet_values?: number
}

/** TypesenseHit is one document + its highlights. */
export interface TypesenseHit {
  document: RawSearchDocument
  highlights: TypesenseHighlight[]
}

/** TypesenseHighlight is a single per-field highlight snippet. */
export interface TypesenseHighlight {
  field: string
  snippet: string
  matched_tokens?: string[]
}

/** TypesenseFacetCount is one facet bucket. */
export interface TypesenseFacetCount {
  value: string
  count: number
}

/** TypesenseFacet groups the bucket counts for a single field. */
export interface TypesenseFacet {
  field_name: string
  counts: TypesenseFacetCount[]
}

/** TypesenseRequestParams echoes back what the server actually ran. */
export interface TypesenseRequestParams {
  collection_name: string
  q: string
  first_q?: string
  per_page: number
}

/** TypesenseSearchResponse is the full /search response shape. */
export interface TypesenseSearchResponse {
  found: number
  out_of: number
  page: number
  per_page?: number
  search_time_ms: number
  hits: TypesenseHit[]
  facet_counts: TypesenseFacet[]
  request_params: TypesenseRequestParams
  corrected_query?: string
}

/**
 * TypesenseSearchClient is the minimal HTTP wrapper used by the
 * browser to talk to Typesense via a scoped API key. Stateless and
 * safe to instantiate per request — the constructor only stores
 * the host + key.
 */
export class TypesenseSearchClient {
  private readonly host: string
  private readonly apiKey: string

  constructor(host: string, scopedApiKey: string) {
    if (!host) throw new Error("TypesenseSearchClient: host is required")
    if (!scopedApiKey) throw new Error("TypesenseSearchClient: scoped key is required")
    this.host = host.replace(/\/$/, "")
    this.apiKey = scopedApiKey
  }

  /**
   * search runs a single GET against /collections/:name/documents/search
   * and returns the parsed response. Throws on any non-2xx status so
   * the calling hook can surface the error to TanStack Query.
   */
  async search(
    collection: string,
    params: TypesenseSearchParams,
    signal?: AbortSignal,
  ): Promise<TypesenseSearchResponse> {
    const url = `${this.host}/collections/${encodeURIComponent(collection)}/documents/search?${this.buildQueryString(params)}`
    const res = await fetch(url, {
      method: "GET",
      headers: {
        "X-TYPESENSE-API-KEY": this.apiKey,
        Accept: "application/json",
      },
      signal,
    })
    if (!res.ok) {
      const text = await res.text().catch(() => "")
      throw new Error(`typesense search failed: ${res.status} ${text}`)
    }
    return res.json() as Promise<TypesenseSearchResponse>
  }

  /**
   * buildQueryString flattens the typed params struct into a URL
   * query string. Optional fields are dropped so the wire payload
   * stays minimal, mirroring the backend's hand-rolled client.
   */
  private buildQueryString(params: TypesenseSearchParams): string {
    const search = new URLSearchParams()
    search.set("q", params.q)
    search.set("query_by", params.query_by)
    if (params.filter_by) search.set("filter_by", params.filter_by)
    if (params.facet_by) search.set("facet_by", params.facet_by)
    if (params.sort_by) search.set("sort_by", params.sort_by)
    if (typeof params.page === "number") search.set("page", String(params.page))
    if (typeof params.per_page === "number") search.set("per_page", String(params.per_page))
    if (params.exclude_fields) search.set("exclude_fields", params.exclude_fields)
    if (params.highlight_fields) search.set("highlight_fields", params.highlight_fields)
    if (params.highlight_full_fields) search.set("highlight_full_fields", params.highlight_full_fields)
    if (params.num_typos) search.set("num_typos", params.num_typos)
    if (typeof params.max_facet_values === "number") {
      search.set("max_facet_values", String(params.max_facet_values))
    }
    return search.toString()
  }
}
