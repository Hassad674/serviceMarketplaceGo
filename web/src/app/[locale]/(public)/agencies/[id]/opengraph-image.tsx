/**
 * opengraph-image.tsx for agency profiles.
 *
 * See `freelancers/[id]/opengraph-image.tsx` for the rationale —
 * this file mirrors the freelancer renderer with agency-specific
 * data plumbing (different metadata fetcher + role tag).
 */

import { getTranslations } from "next-intl/server"
import { fetchAgencyProfileForMetadata } from "@/features/provider/api/agency-profile-server"
import { fetchPublicAverageRating } from "@/shared/lib/seo/server-fetchers"
import {
  renderProfileOgImage,
  type OgProfileInput,
} from "@/shared/lib/seo/og-profile"
import { OG_DIMENSIONS } from "@/shared/lib/seo/og-tokens"

export const alt = "Marketplace Service — profil agence"
export const size = OG_DIMENSIONS
export const contentType = "image/png"

interface ImageProps {
  params: Promise<{ id: string; locale: string }>
}

export default async function AgencyOgImage({ params }: ImageProps) {
  const { id, locale } = await params
  const [profile, rating, t, tSeo] = await Promise.all([
    fetchAgencyProfileForMetadata(id),
    fetchPublicAverageRating(id),
    getTranslations({ locale, namespace: "publicProfile" }),
    getTranslations({ locale, namespace: "seo" }),
  ])

  const displayName = profile?.title || t("agencyProfile")
  const ratingLine =
    rating && rating.count > 0
      ? tSeo("ratingLine", {
          rating: roundOne(rating.average),
          count: rating.count,
        })
      : undefined

  const input: OgProfileInput = {
    displayName,
    title: profile?.title || undefined,
    city: profile?.city || undefined,
    photoUrl: profile?.photo_url || undefined,
    ratingLine,
    roleTag: tSeo("ogRoleAgency"),
    siteName: tSeo("siteName"),
    footerLabel: tSeo("footerLabel"),
  }

  return renderProfileOgImage(input)
}

function roundOne(value: number): string {
  return (Math.round(value * 10) / 10).toFixed(1)
}
