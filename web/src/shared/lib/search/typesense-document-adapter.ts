/**
 * typesense-document-adapter.ts converts the raw Typesense
 * `RawSearchDocument` shape into the frozen UI `SearchDocument`
 * shape consumed by the search result card.
 *
 * The two shapes differ because Typesense flattens pricing into
 * scalar fields (`pricing_min_amount`, `pricing_currency`, …) while
 * the UI card expects a nested `SearchDocumentPricing` object. Doing
 * the conversion in a single adapter file means the card can stay
 * source-agnostic — swapping the data layer is one edit here.
 */

import type {
  SearchDocument,
  SearchDocumentAvailability,
  SearchDocumentPersona,
  SearchDocumentPricing,
  SearchDocumentPricingType,
} from "./search-document"
import type { RawSearchDocument } from "./typesense-client"

const AVAILABILITY_VALUES: readonly SearchDocumentAvailability[] = [
  "available_now",
  "available_soon",
  "not_available",
]

const PRICING_TYPES: readonly SearchDocumentPricingType[] = [
  "daily",
  "hourly",
  "project_from",
  "project_range",
  "commission_pct",
  "commission_flat",
]

/**
 * fromTypesenseDocument maps one Typesense raw document into the
 * frozen card contract. The raw `created_at` is a Unix epoch from
 * Typesense; we render it as an ISO string so the card's "recent
 * activity" line stays compatible with the legacy SQL adapter.
 */
export function fromTypesenseDocument(raw: RawSearchDocument): SearchDocument {
  // Typesense's primary key is now `{orgID}:{persona}` (phase 2
  // ID collision fix). Expose the raw organisation UUID to the card
  // so the profile link stays `/freelancers/{orgID}`.
  return {
    id: raw.organization_id ?? raw.id.split(":")[0] ?? raw.id,
    persona: raw.persona as SearchDocumentPersona,
    display_name: raw.display_name ?? "",
    title: raw.title ?? "",
    photo_url: raw.photo_url ?? "",
    city: raw.city ?? "",
    country_code: raw.country_code ?? "",
    languages_professional: raw.languages_professional ?? [],
    availability_status: toAvailability(raw.availability_status),
    expertise_domains: raw.expertise_domains ?? [],
    skills: (raw.skills ?? []).slice(0, 6),
    pricing: extractPricing(raw),
    rating: {
      average: raw.rating_average ?? 0,
      count: raw.rating_count ?? 0,
    },
    total_earned: raw.total_earned ?? 0,
    completed_projects: raw.completed_projects ?? 0,
    created_at: raw.created_at ? new Date(raw.created_at * 1000).toISOString() : "",
  }
}

function toAvailability(raw: string | undefined): SearchDocumentAvailability {
  if (raw && (AVAILABILITY_VALUES as readonly string[]).includes(raw)) {
    return raw as SearchDocumentAvailability
  }
  return "available_now"
}

function extractPricing(raw: RawSearchDocument): SearchDocumentPricing | null {
  if (!raw.pricing_type) return null
  if (!(PRICING_TYPES as readonly string[]).includes(raw.pricing_type)) return null
  return {
    type: raw.pricing_type as SearchDocumentPricingType,
    min_amount: raw.pricing_min_amount ?? 0,
    max_amount: raw.pricing_max_amount ?? null,
    currency: raw.pricing_currency ?? "EUR",
    negotiable: raw.pricing_negotiable === true,
  }
}
