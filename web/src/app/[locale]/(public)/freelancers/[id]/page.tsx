import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { ArrowLeft } from "lucide-react"
import { Link } from "@i18n/navigation"
import { SendMessageButton } from "@/features/messaging/components/send-message-button"
import {
  FreelancePublicProfileLoader,
} from "@/features/freelance-profile/components/freelance-public-profile-loader"
import {
  fetchFreelanceProfileForMetadata,
} from "@/features/freelance-profile/api/freelance-profile-server"
import { safeJsonLd } from "@/shared/lib/json-ld"
import {
  buildAlternates,
  absoluteUrl,
  type SupportedLocale,
} from "@/shared/lib/seo/alternates"
import { buildBreadcrumbList } from "@/shared/lib/seo/breadcrumbs"
import {
  buildAggregateRating,
  buildReviewItems,
} from "@/shared/lib/seo/rating"
import {
  fetchPublicAverageRating,
  fetchPublicReviews,
  fetchRelatedProfiles,
} from "@/shared/lib/seo/server-fetchers"
import { BreadcrumbNav } from "@/shared/components/seo/breadcrumb-nav"
import { RelatedProfiles } from "@/shared/components/seo/related-profiles"

type Props = {
  params: Promise<{ id: string; locale: string }>
}

// generateMetadata runs on the server and populates the SEO head with
// hreflang alternates, OpenGraph, and Twitter card data. The OG image
// is served by the colocated `opengraph-image.tsx` route — Next.js
// auto-discovers the file and wires it into `metadata.openGraph.images`.
export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { id, locale } = await params
  const t = await getTranslations({ locale, namespace: "profile.freelance" })
  const profile = await fetchFreelanceProfileForMetadata(id)

  const displayName = profile?.title || t("publicTitleSuffix")
  const titleSuffix = t("publicTitleSuffix")
  const title = `${displayName} — ${titleSuffix} | Marketplace Service`
  const description = t("publicDescription", {
    name: displayName,
    title: profile?.title ? profile.title : "empty",
  })
  const alternates = buildAlternates({
    locale: locale as SupportedLocale,
    path: `/freelancers/${id}`,
  })

  return {
    title,
    description,
    alternates,
    openGraph: {
      title,
      description,
      type: "profile",
      url: alternates.canonical,
      locale: locale === "fr" ? "fr_FR" : "en_US",
    },
    twitter: {
      card: "summary_large_image",
      title,
      description,
    },
  }
}

export default async function FreelancerProfilePage({ params }: Props) {
  const { id, locale } = await params
  const [profile, rating, reviews, related, t, tSeo] = await Promise.all([
    fetchFreelanceProfileForMetadata(id),
    fetchPublicAverageRating(id),
    fetchPublicReviews(id, 5),
    fetchRelatedProfiles({
      type: "freelancer",
      excludeOrgId: id,
      primaryExpertise: undefined,
      city: undefined,
    }),
    getTranslations({ locale, namespace: "profile.freelance" }),
    getTranslations({ locale, namespace: "seo" }),
  ])

  // Re-pick related using the resolved profile fields once we have
  // them — fetchRelatedProfiles bootstrapped with empty filters keeps
  // the parallel `Promise.all` shape clean.
  const filteredRelated =
    profile && related.length > 0
      ? related
          .filter((doc) => {
            const docOrg =
              doc.organization_id ?? doc.id.split(":")[0] ?? doc.id
            return docOrg !== id
          })
          .sort((a, b) => {
            const aMatch = primaryMatch(a, profile)
            const bMatch = primaryMatch(b, profile)
            if (aMatch !== bMatch) return bMatch - aMatch
            return b.rating_average - a.rating_average
          })
          .slice(0, 6)
      : related

  const breadcrumbCrumbs = [
    {
      label: tSeo("breadcrumbHome"),
      href: "/",
      item: absoluteUrl(`/${locale}`),
    },
    {
      label: tSeo("breadcrumbFreelancers"),
      href: "/freelancers",
      item: absoluteUrl(`/${locale}/freelancers`),
    },
    {
      label: profile?.title || t("publicTitleSuffix"),
    },
  ]

  return (
    <div className="mx-auto w-full max-w-5xl space-y-5">
      <BreadcrumbNav
        ariaLabel={tSeo("breadcrumbAriaLabel")}
        crumbs={breadcrumbCrumbs.map((c) => ({ label: c.label, href: c.href }))}
      />
      <Link
        href="/freelancers"
        className="inline-flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
      >
        <ArrowLeft className="h-3 w-3" aria-hidden="true" />
        {t("backToList")}
      </Link>
      <FreelancePublicProfileLoader orgId={id} />
      {profile ? (
        <JsonLd
          profileId={id}
          profile={profile}
          rating={rating}
          reviews={reviews}
          anonymousReviewer={tSeo("anonymousReviewer")}
        />
      ) : null}
      <BreadcrumbsJsonLd
        crumbs={breadcrumbCrumbs.map((c) => ({ name: c.label, item: c.item }))}
      />
      <div className="flex justify-center pt-2">
        <SendMessageButton targetOrgId={id} />
      </div>
      <RelatedProfiles
        type="freelancer"
        documents={filteredRelated}
        labels={{
          heading: tSeo("relatedHeadingFreelance"),
          subheading: tSeo("relatedSubheading"),
          viewProfile: tSeo("relatedViewProfile"),
          cityFallback: tSeo("relatedCityFallback"),
        }}
      />
    </div>
  )
}

interface JsonLdProps {
  profileId: string
  profile: NonNullable<Awaited<ReturnType<typeof fetchFreelanceProfileForMetadata>>>
  rating: Awaited<ReturnType<typeof fetchPublicAverageRating>>
  reviews: Awaited<ReturnType<typeof fetchPublicReviews>>
  anonymousReviewer: string
}

function JsonLd({
  profileId,
  profile,
  rating,
  reviews,
  anonymousReviewer,
}: JsonLdProps) {
  // Person schema enriched per Google's Rich Results Test guidance:
  // - knowsAbout pulls from skills + expertise_domains so the entity
  //   surfaces in `query intent` matches.
  // - aggregateRating + review[] are emitted only when the entity has
  //   at least one published review; an empty review block disqualifies
  //   the rich result and triggers Search Console warnings.
  const knowsAbout = mergeKnowsAbout(
    profile.skills?.map((s) => s.display_text) ?? [],
    profile.expertise_domains ?? [],
  )
  const aggregateRating = buildAggregateRating({ rating })
  const reviewItems = buildReviewItems({
    reviews,
    anonymousAuthorLabel: anonymousReviewer,
  })

  const payload: Record<string, unknown> = {
    "@context": "https://schema.org",
    "@type": "Person",
    "@id": absoluteUrl(`/freelancers/${profileId}`),
    name: profile.title || profileId,
    jobTitle: profile.title || undefined,
    description: profile.about || undefined,
    image: profile.photo_url ? absoluteUrl(profile.photo_url) : undefined,
    url: absoluteUrl(`/freelancers/${profileId}`),
    address: profile.city
      ? {
          "@type": "PostalAddress",
          addressLocality: profile.city,
          addressCountry: profile.country_code || undefined,
        }
      : undefined,
    knowsAbout: knowsAbout.length > 0 ? knowsAbout : undefined,
    knowsLanguage: profile.languages_professional,
    aggregateRating,
    review: reviewItems,
  }
  return (
    <script
      type="application/ld+json"
      // SEO JSON-LD must be rendered as raw JSON; React escaping would
      // break the schema. `profile.about` is user-authored text, so we
      // route through safeJsonLd() to neutralize </script>, --> and the
      // unicode line/paragraph separators before injecting via
      // dangerouslySetInnerHTML.
      dangerouslySetInnerHTML={{ __html: safeJsonLd(payload) }}
    />
  )
}

interface BreadcrumbsJsonLdProps {
  crumbs: Array<{ name: string; item?: string }>
}

function BreadcrumbsJsonLd({ crumbs }: BreadcrumbsJsonLdProps) {
  return (
    <script
      type="application/ld+json"
      dangerouslySetInnerHTML={{
        __html: safeJsonLd(buildBreadcrumbList(crumbs)),
      }}
    />
  )
}

function mergeKnowsAbout(skills: string[], expertises: string[]): string[] {
  const combined = [...skills, ...expertises]
    .map((entry) => entry.trim())
    .filter(Boolean)
  return Array.from(new Set(combined))
}

function primaryMatch(
  doc: { expertise_domains: string[]; city?: string },
  profile: { expertise_domains?: string[]; city?: string },
): number {
  let score = 0
  const primary = profile.expertise_domains?.[0]
  if (primary && doc.expertise_domains?.includes(primary)) score += 10
  if (profile.city && doc.city === profile.city) score += 4
  return score
}
