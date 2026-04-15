/**
 * build-filter-by.ts is the TypeScript counterpart of the Go
 * `BuildFilterBy` function in `internal/app/search/filter_builder.go`.
 *
 * It must produce IDENTICAL output to the backend for the same
 * inputs — parity is enforced by a shared snapshot test (see
 * `__tests__/build-filter-by.test.ts`). Renaming a field here is a
 * breaking change for the whole listing page; coordinate any edits
 * with `BuildFilterBy` in the backend.
 *
 * Why a frontend mirror exists at all: the listing page queries
 * Typesense directly using a scoped API key (zero-hop latency), so
 * the filter string is built in the browser, not the backend.
 */

/** SearchFilterInput mirrors the Go FilterInput struct. */
export interface SearchFilterInput {
  availabilityStatus?: string[]
  pricingMin?: number | null
  pricingMax?: number | null
  city?: string
  countryCode?: string
  geoLat?: number | null
  geoLng?: number | null
  geoRadiusKm?: number | null
  languages?: string[]
  expertiseDomains?: string[]
  skills?: string[]
  ratingMin?: number | null
  workMode?: string[]
  isVerified?: boolean | null
  isTopRated?: boolean | null
  negotiable?: boolean | null
}

/**
 * buildFilterBy assembles the Typesense filter_by string from the
 * given inputs. Returns an empty string when no filter is set so
 * the scoped client's persona clause is the only filter applied.
 *
 * Field order is fixed (mirrors the backend) so unit + parity
 * tests can assert on the exact output.
 */
export function buildFilterBy(input: SearchFilterInput): string {
  const clauses: string[] = []

  pushIf(clauses, availabilityClause(input.availabilityStatus))
  pushIf(clauses, pricingClause(input.pricingMin, input.pricingMax))
  pushIf(clauses, cityClause(input.city))
  pushIf(clauses, countryClause(input.countryCode))
  pushIf(clauses, geoClause(input.geoLat, input.geoLng, input.geoRadiusKm))
  pushIf(clauses, stringSliceClause("languages_professional", input.languages))
  pushIf(clauses, stringSliceClause("expertise_domains", input.expertiseDomains))
  pushIf(clauses, stringSliceClause("skills", input.skills))
  pushIf(clauses, ratingClause(input.ratingMin))
  pushIf(clauses, stringSliceClause("work_mode", input.workMode))
  pushIf(clauses, booleanClause("is_verified", input.isVerified))
  pushIf(clauses, booleanClause("is_top_rated", input.isTopRated))
  pushIf(clauses, booleanClause("pricing_negotiable", input.negotiable))

  return clauses.join(" && ")
}

function pushIf(arr: string[], clause: string): void {
  if (clause) arr.push(clause)
}

function availabilityClause(values?: string[]): string {
  const cleaned = dedupe(values)
  return cleaned.length ? `availability_status:[${cleaned.join(",")}]` : ""
}

function pricingClause(minAmt?: number | null, maxAmt?: number | null): string {
  const parts: string[] = []
  if (minAmt !== undefined && minAmt !== null) {
    parts.push(`pricing_min_amount:>=${minAmt}`)
  }
  if (maxAmt !== undefined && maxAmt !== null) {
    parts.push(`pricing_min_amount:<=${maxAmt}`)
  }
  return parts.join(" && ")
}

function cityClause(city?: string): string {
  const trimmed = (city ?? "").trim()
  return trimmed ? `city:=\`${trimmed}\`` : ""
}

function countryClause(code?: string): string {
  const trimmed = (code ?? "").trim()
  return trimmed ? `country_code:=${trimmed.toLowerCase()}` : ""
}

function geoClause(
  lat?: number | null,
  lng?: number | null,
  radiusKm?: number | null,
): string {
  if (lat == null || lng == null || radiusKm == null) return ""
  if (radiusKm <= 0) return ""
  return `location:(${formatNumber(lat)},${formatNumber(lng)},${formatNumber(radiusKm)} km)`
}

function stringSliceClause(field: string, values?: string[]): string {
  const cleaned = dedupe(values)
  return cleaned.length ? `${field}:[${cleaned.join(",")}]` : ""
}

function ratingClause(minRating?: number | null): string {
  if (minRating == null || minRating <= 0) return ""
  return `rating_average:>=${formatNumber(minRating)}`
}

function booleanClause(field: string, value?: boolean | null): string {
  if (value === null || value === undefined) return ""
  return `${field}:=${value ? "true" : "false"}`
}

function dedupe(values?: string[]): string[] {
  if (!values || values.length === 0) return []
  const seen = new Set<string>()
  const out: string[] = []
  for (const v of values) {
    const trimmed = v.trim()
    if (!trimmed || seen.has(trimmed)) continue
    seen.add(trimmed)
    out.push(trimmed)
  }
  return out
}

/**
 * formatNumber prints a number without trailing zeros so the wire
 * format matches Go's `strconv.FormatFloat(f, 'f', -1, 64)`. We
 * never use scientific notation because Typesense expects plain
 * decimals on its filter_by strings.
 */
function formatNumber(n: number): string {
  if (Number.isInteger(n)) return n.toString()
  // Strip trailing zeros after the decimal point.
  return n.toString().replace(/(\.\d*?[1-9])0+$|\.0+$/, "$1")
}
