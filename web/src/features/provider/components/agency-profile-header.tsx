"use client"

import { useTranslations } from "next-intl"
import { PrestataireProfileHeader } from "@/shared/components/profile/prestataire-profile-header"
import type { FormattablePricing } from "@/shared/lib/profile/pricing-format"
import type { Pricing, Profile } from "../api/profile-api"

interface AgencyProfileHeaderProps {
  profile: Profile
  displayName: string
  rating?: { average: number; count: number }
  editable?: {
    onSaveTitle?: (next: string) => void
    onUploadPhoto?: (file: File) => Promise<void>
    uploadingPhoto?: boolean
  }
  /**
   * When true, marks the photo as a high-priority next/image. Public
   * profile pages opt in because the photo is the LCP element; editable
   * dashboard contexts leave it false (default) so Next.js lazy-loads.
   */
  photoPriority?: boolean
}

// AgencyProfileHeader is a thin adapter over the shared
// PrestataireProfileHeader. It folds the legacy agency Profile shape
// (pricing as an array of rows keyed by `kind`) into the shared
// header config, so the agency hero looks identical to the freelance
// one — same Soleil v2 cover band, 130px portrait, name, italic
// title, meta row and right-rail pricing block.
export function AgencyProfileHeader(props: AgencyProfileHeaderProps) {
  const { profile, displayName, rating, editable, photoPriority } = props
  const tFreelance = useTranslations("profile.freelance")

  const pricingValue = pickDirectPricing(profile)

  return (
    <PrestataireProfileHeader
      header={{
        kind: "agency",
        identity: {
          photoUrl: profile.photo_url,
          displayName,
          title: profile.title,
          availabilityStatus: profile.availability_status,
          organizationId: profile.organization_id,
          city: profile.city,
          countryCode: profile.country_code,
          languagesProfessional: profile.languages_professional,
        },
        pricing: {
          value: pricingValue,
          fromLabel: tFreelance("priceFromLabel"),
        },
        rating,
        editable,
        photoPriority,
      }}
    />
  )
}

// pickDirectPricing collapses the legacy agency pricing rows (an
// array keyed by `kind`) onto the single FormattablePricing shape the
// shared header consumes. Agencies surface their direct rate on the
// hero — referral commissions live on the apporteur surface, not
// here. Returns null when no direct row exists so the rail hides.
function pickDirectPricing(profile: Profile): FormattablePricing | null {
  const row = profile.pricing?.find(
    (p: Pricing) => p.kind === "direct",
  )
  if (!row) return null
  return {
    type: row.type,
    min_amount: row.min_amount,
    max_amount: row.max_amount,
    currency: row.currency,
  }
}
