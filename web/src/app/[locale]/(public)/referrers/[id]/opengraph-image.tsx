/**
 * opengraph-image.tsx for referrer profiles.
 *
 * Same renderer / pattern as the freelance + agency variants. Uses
 * the referrer-specific metadata fetcher and tag.
 */

import { getTranslations } from "next-intl/server"
import { fetchReferrerProfileForMetadata } from "@/features/referrer-profile/api/referrer-profile-server"
import { fetchPublicAverageRating } from "@/shared/lib/seo/server-fetchers"
import {
  renderProfileOgImage,
  type OgProfileInput,
} from "@/shared/lib/seo/og-profile"
import { OG_DIMENSIONS } from "@/shared/lib/seo/og-tokens"

export const alt = "Marketplace Service — profil apporteur d'affaires"
export const size = OG_DIMENSIONS
export const contentType = "image/png"

interface ImageProps {
  params: Promise<{ id: string; locale: string }>
}

export default async function ReferrerOgImage({ params }: ImageProps) {
  const { id, locale } = await params
  const [profile, rating, t, tSeo] = await Promise.all([
    fetchReferrerProfileForMetadata(id),
    fetchPublicAverageRating(id),
    getTranslations({ locale, namespace: "profile.referrer" }),
    getTranslations({ locale, namespace: "seo" }),
  ])

  const fallback = t("displayNameFallback")
  const displayName = profile?.title || fallback
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
    roleTag: tSeo("ogRoleReferrer"),
    siteName: tSeo("siteName"),
    footerLabel: tSeo("footerLabel"),
  }

  return renderProfileOgImage(input)
}

function roundOne(value: number): string {
  return (Math.round(value * 10) / 10).toFixed(1)
}
