// Adapter that projects the current `/api/v1/profiles/search` envelope
// into a `SearchDocument`. The current backend returns a slightly
// different shape (PublicProfileSummary) that will eventually be
// replaced by a direct Typesense index — keeping the card strictly
// typed against `SearchDocument` means that swap becomes a one-file
// change in this adapter, with zero churn on the UI.
//
// The adapter tolerates incomplete payloads (older backend versions
// that do not yet ship total_earned, expertise_domains, etc.) by
// defaulting to safe zero values. That lets the new card ship in
// lockstep with the backend extension instead of forcing a hard
// version pin.

import type {
  SearchDocument,
  SearchDocumentAvailability,
  SearchDocumentPersona,
  SearchDocumentPricing,
  SearchDocumentPricingType,
} from "./search-document"

// RawSearchDocumentLike is the intentionally-loose input contract the
// adapter accepts. It mirrors the legacy PublicProfileSummary fields
// that the backend currently returns, plus the new aggregate fields
// added in this redesign. Every field is optional so legacy callers
// (integration tests, older fixtures) do not need to fill everything.
export interface RawSearchDocumentLike {
  organization_id?: string
  id?: string
  owner_user_id?: string
  name?: string
  display_name?: string
  org_type?: string
  persona?: string
  title?: string
  photo_url?: string
  city?: string
  country_code?: string
  languages_professional?: string[]
  availability_status?: string
  expertise_domains?: string[]
  skills?: Array<{ display_text?: string; skill_text?: string } | string>
  pricing?: Array<{
    kind?: string
    type?: string
    min_amount?: number
    max_amount?: number | null
    currency?: string
    negotiable?: boolean
  }> | null
  average_rating?: number
  review_count?: number
  total_earned?: number
  completed_projects?: number
  referrer_enabled?: boolean
  created_at?: string
}

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

// inferPersona maps the legacy org_type + referrer flag combo into the
// three frozen SearchDocument personas. This is where "provider_personal"
// splits into "freelance" or "referrer" depending on which directory
// surfaced the document.
export function inferPersona(
  orgType: string | undefined,
  hint: SearchDocumentPersona | undefined,
): SearchDocumentPersona {
  if (hint) return hint
  if (orgType === "agency") return "agency"
  return "freelance"
}

function toAvailability(
  raw: string | undefined,
): SearchDocumentAvailability {
  if (raw && (AVAILABILITY_VALUES as readonly string[]).includes(raw)) {
    return raw as SearchDocumentAvailability
  }
  return "available_now"
}

function toPricingType(
  raw: string | undefined,
): SearchDocumentPricingType | null {
  if (raw && (PRICING_TYPES as readonly string[]).includes(raw)) {
    return raw as SearchDocumentPricingType
  }
  return null
}

function pickPricing(
  rows: RawSearchDocumentLike["pricing"],
  persona: SearchDocumentPersona,
): SearchDocumentPricing | null {
  if (!rows || rows.length === 0) return null
  // Referrers surface the "referral" row; freelancers and agencies
  // surface the "direct" row. Fall back to the first row if the
  // preferred kind is missing — better than hiding pricing entirely.
  const preferredKind = persona === "referrer" ? "referral" : "direct"
  const preferred = rows.find((row) => row.kind === preferredKind)
  const row = preferred ?? rows[0]
  const type = toPricingType(row.type)
  if (!type) return null
  const min = typeof row.min_amount === "number" ? row.min_amount : 0
  const max = typeof row.max_amount === "number" ? row.max_amount : null
  const currency = row.currency ?? "EUR"
  const negotiable = row.negotiable === true
  return {
    type,
    min_amount: min,
    max_amount: max,
    currency,
    negotiable,
  }
}

function pickSkills(
  skills: RawSearchDocumentLike["skills"],
  limit: number,
): string[] {
  if (!skills || skills.length === 0) return []
  const out: string[] = []
  for (const entry of skills) {
    if (out.length >= limit) break
    if (typeof entry === "string") {
      if (entry) out.push(entry)
      continue
    }
    const label = entry.display_text ?? entry.skill_text
    if (label) out.push(label)
  }
  return out
}

// toSearchDocument converts a loosely-typed legacy search envelope into
// a fully-typed SearchDocument. Missing fields are defaulted so the
// card can always render — an older backend that does not yet expose
// total_earned simply reports 0 and the card hides the "gagnés" line.
export function toSearchDocument(
  raw: RawSearchDocumentLike,
  persona: SearchDocumentPersona,
): SearchDocument {
  const displayName = raw.display_name ?? raw.name ?? ""
  const id = raw.id ?? raw.organization_id ?? ""
  return {
    id,
    persona,
    display_name: displayName,
    title: raw.title ?? "",
    photo_url: raw.photo_url ?? "",
    city: raw.city ?? "",
    country_code: raw.country_code ?? "",
    languages_professional: raw.languages_professional ?? [],
    availability_status: toAvailability(raw.availability_status),
    expertise_domains: raw.expertise_domains ?? [],
    skills: pickSkills(raw.skills, 6),
    pricing: pickPricing(raw.pricing, persona),
    rating: {
      average: raw.average_rating ?? 0,
      count: raw.review_count ?? 0,
    },
    total_earned: raw.total_earned ?? 0,
    completed_projects: raw.completed_projects ?? 0,
    created_at: raw.created_at ?? "",
  }
}
