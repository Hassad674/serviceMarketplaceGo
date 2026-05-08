"use client"

import { useTranslations } from "next-intl"
import { PrestataireProfileHeader } from "@/shared/components/profile/prestataire-profile-header"
import type { ReferrerProfile } from "../api/referrer-profile-api"

interface ReferrerProfileHeaderProps {
  profile: ReferrerProfile
  displayName: string
  rating?: { average: number; count: number }
  editable?: {
    onSaveTitle?: (next: string) => void
    onUploadPhoto?: (file: File) => Promise<void>
    uploadingPhoto?: boolean
  }
  /**
   * When true, the underlying header marks the photo as a high-priority
   * image (LCP). Public profile pages opt in.
   */
  photoPriority?: boolean
}

// ReferrerProfileHeader is a thin adapter over the shared
// PrestataireProfileHeader. Same visual shell as freelance/agency,
// plus the apporteur badge under the name and a "Commission" rail
// label so the headline rate stays meaningful.
export function ReferrerProfileHeader(props: ReferrerProfileHeaderProps) {
  const { profile, displayName, rating, editable, photoPriority } = props
  const tReferrer = useTranslations("profile.referrer")
  const tSidebar = useTranslations("sidebar")

  return (
    <PrestataireProfileHeader
      header={{
        kind: "referrer",
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
          value: profile.pricing,
          fromLabel: tReferrer("priceFromLabel"),
        },
        rating,
        badge: { label: tSidebar("businessReferrer") },
        editable,
        photoPriority,
      }}
    />
  )
}
