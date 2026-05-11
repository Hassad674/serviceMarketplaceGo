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
} from "@/shared/lib/seo/server-fetchers"
import { BreadcrumbNav } from "@/shared/components/seo/breadcrumb-nav"

type Props = {
  params: Promise<{ id: string; locale: string }>
}

// PERF-B: ISR revalidation window for the public freelance profile.
// The Next.js cache stores the rendered RSC payload + the API responses
// it fetched for `revalidate` seconds. After that, the next request
// triggers a background revalidation while still serving the stale
// page (stale-while-revalidate). Combined with the backend's
// `Cache-Control: public, s-maxage=300` on `/api/v1/freelance-profiles/{id}`,
// this collapses the steady-state cost of a profile page to ~0 on
// Vercel's edge once the cache is warm.
//
// 60s matches the backend max-age, so a browser cache hit and a Next.js
// edge revalidation align. Authenticated previews bypass the cache:
// the server-side fetch attaches the session cookie which the backend
// translates into `Cache-Control: private, no-store`, so the edge
// stores nothing for that variant.
export const revalidate = 60

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
  const [profile, rating, reviews, t, tSeo] = await Promise.all([
    fetchFreelanceProfileForMetadata(id),
    fetchPublicAverageRating(id),
    fetchPublicReviews(id, 5),
    getTranslations({ locale, namespace: "profile.freelance" }),
    getTranslations({ locale, namespace: "seo" }),
  ])

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
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Link
          href="/freelancers"
          className="inline-flex items-center gap-1.5 text-xs text-muted-foreground transition-colors hover:text-foreground"
        >
          <ArrowLeft className="h-3 w-3" aria-hidden="true" />
          {t("backToList")}
        </Link>
        <SendMessageButton targetOrgId={id} persona="freelance" />
      </div>
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
