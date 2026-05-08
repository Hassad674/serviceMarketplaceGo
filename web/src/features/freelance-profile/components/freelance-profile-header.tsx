"use client"

import { useTranslations } from "next-intl"
import { PrestataireProfileHeader } from "@/shared/components/profile/prestataire-profile-header"
import type { FreelanceProfile } from "../api/freelance-profile-api"

interface FreelanceProfileHeaderProps {
  profile: FreelanceProfile
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

// FreelanceProfileHeader is a thin adapter over the shared
// PrestataireProfileHeader. It maps the FreelanceProfile aggregate
// onto the shared header config so the freelance, agency and referrer
// hero rows stay visually identical (Soleil v2 cover band + portrait
// + identity + pricing rail) — DRY-ing the three personas under a
// single source of truth.
export function FreelanceProfileHeader(props: FreelanceProfileHeaderProps) {
  const { profile, displayName, rating, editable, photoPriority } = props
  const tFreelance = useTranslations("profile.freelance")

  return (
    <PrestataireProfileHeader
      header={{
        kind: "freelance",
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
          fromLabel: tFreelance("priceFromLabel"),
        },
        rating,
        editable,
        photoPriority,
      }}
    />
  )
}
