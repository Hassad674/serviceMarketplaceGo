/**
 * opengraph-image.tsx for freelancer profiles.
 *
 * Next.js 16 picks up this file automatically and wires the rendered
 * 1200x630 image into `metadata.openGraph.images` for the same route.
 * The same image is reused for the Twitter `summary_large_image`
 * card by referencing it via `metadata.twitter.images`, which Next.js
 * also auto-populates from the Open Graph default.
 *
 * The renderer runs in the Edge runtime — no DB access, only the
 * server-only HTTP fetcher we already use for SEO metadata.
 */

import { getTranslations } from "next-intl/server"
import {
  fetchFreelanceProfileForMetadata,
} from "@/features/freelance-profile/api/freelance-profile-server"
import { fetchPublicAverageRating } from "@/shared/lib/seo/server-fetchers"
import {
  renderProfileOgImage,
  type OgProfileInput,
} from "@/shared/lib/seo/og-profile"
import { OG_DIMENSIONS } from "@/shared/lib/seo/og-tokens"

export const alt = "Marketplace Service — profil freelance"
export const size = OG_DIMENSIONS
export const contentType = "image/png"

interface ImageProps {
  params: Promise<{ id: string; locale: string }>
}

export default async function FreelancerOgImage({ params }: ImageProps) {
  const { id, locale } = await params
  const [profile, rating, t, tSeo] = await Promise.all([
    fetchFreelanceProfileForMetadata(id),
    fetchPublicAverageRating(id),
    getTranslations({ locale, namespace: "profile.freelance" }),
    getTranslations({ locale, namespace: "seo" }),
  ])

  const displayName = profile?.title || t("publicTitleSuffix")
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
    roleTag: tSeo("ogRoleFreelance"),
    siteName: tSeo("siteName"),
    footerLabel: tSeo("footerLabel"),
  }

  return renderProfileOgImage(input)
}

function roundOne(value: number): string {
  return (Math.round(value * 10) / 10).toFixed(1)
}
