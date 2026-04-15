// SearchDocument is the SHAPE the card consumes. It mirrors what the
// future Typesense index will return for every document in the
// freelancers / agencies / referrers collections. The card reads ONLY
// these fields — no secondary fetches, no N+1 when the backend swaps
// to Typesense.
//
// The shape is intentionally frozen: renaming a field here is a
// breaking change for every consumer (web card, mobile card, filter
// sidebar, future Typesense schema). New optional fields can be added
// without breaking anything as long as the reader tolerates `undefined`.
//
// All monetary amounts are in the SMALLEST currency unit (cents/centimes
// for fiat, basis points for commission_pct) — matching the backend's
// `profile_pricing.min_amount` semantics and the internal accounting in
// `payment_records` / `proposal_milestones`.

export type SearchDocumentPersona = "freelance" | "agency" | "referrer"

export type SearchDocumentAvailability =
  | "available_now"
  | "available_soon"
  | "not_available"

export type SearchDocumentPricingType =
  | "daily"
  | "hourly"
  | "project_from"
  | "project_range"
  | "commission_pct"
  | "commission_flat"

// SearchDocumentPricing describes the single pricing row surfaced on a
// search result card. For Typesense, this is derived server-side from
// the richer `profile_pricing` table: the "direct" kind for freelance
// and agency documents, the "referral" kind for referrer documents.
// Only one pricing row is exposed — the card never needs both.
export interface SearchDocumentPricing {
  type: SearchDocumentPricingType
  min_amount: number // smallest unit (centimes / basis points)
  max_amount: number | null
  currency: string
  negotiable: boolean
}

export interface SearchDocumentRating {
  average: number // 0..5
  count: number
}

// SearchDocument is the contract every search result card consumes.
// See the file-level comment for the design rules — treat this shape
// as frozen and extend via optional fields only.
export interface SearchDocument {
  id: string // organization_id
  persona: SearchDocumentPersona
  display_name: string
  title: string // e.g. "Développeur Go"
  photo_url: string
  city: string
  country_code: string
  languages_professional: string[]
  availability_status: SearchDocumentAvailability
  expertise_domains: string[]
  skills: string[] // top N skills, already trimmed
  pricing: SearchDocumentPricing | null
  rating: SearchDocumentRating
  total_earned: number // in cents (0 if none)
  completed_projects: number // count
  created_at: string
}
