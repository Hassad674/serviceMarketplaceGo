import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { PublicClientProfileLoader } from "@/features/client-profile/components/public-client-profile-loader"
import { fetchPublicClientProfileForMetadata } from "@/features/client-profile/api/client-profile-server"
import { safeJsonLd } from "@/shared/lib/json-ld"

type PageProps = {
  params: Promise<{ id: string; locale: string }>
}

// generateMetadata runs on the server with ISR caching (2 min). The
// client page re-fetches via TanStack Query on hydration; because
// staleTime matches revalidate, the second request lands in the
// TanStack cache instead of hitting the network a second time.
export async function generateMetadata({
  params,
}: PageProps): Promise<Metadata> {
  const { id, locale } = await params
  const t = await getTranslations({ locale, namespace: "clientProfile" })
  const profile = await fetchPublicClientProfileForMetadata(id)

  const companyName = profile?.company_name || t("pageTitle")
  const title = `${t("pageTitlePublic", { companyName })} | Marketplace Service`
  const description = profile?.client_description
    ? profile.client_description.slice(0, 160)
    : t("pageTitle")
  const canonical = `/clients/${id}`

  return {
    title,
    description,
    alternates: { canonical },
    openGraph: {
      title,
      description,
      type: "profile",
      url: canonical,
      images: profile?.avatar_url
        ? [
            {
              url: profile.avatar_url,
              width: 400,
              height: 400,
              alt: companyName,
            },
          ]
        : undefined,
    },
    twitter: {
      card: "summary",
      title,
      description,
    },
  }
}

export default async function PublicClientProfilePage({ params }: PageProps) {
  const { id } = await params
  const profile = await fetchPublicClientProfileForMetadata(id)
  return (
    <>
      <PublicClientProfileLoader orgId={id} />
      {profile ? <ClientJsonLd profile={profile} /> : null}
    </>
  )
}

interface ClientJsonLdProps {
  profile: NonNullable<
    Awaited<ReturnType<typeof fetchPublicClientProfileForMetadata>>
  >
}

function ClientJsonLd({ profile }: ClientJsonLdProps) {
  const payload = {
    "@context": "https://schema.org",
    "@type": "Organization",
    "@id": `/clients/${profile.organization_id}`,
    name: profile.company_name,
    description: profile.client_description || undefined,
    image: profile.avatar_url || undefined,
    aggregateRating:
      profile.review_count > 0
        ? {
            "@type": "AggregateRating",
            ratingValue: profile.average_rating,
            reviewCount: profile.review_count,
            bestRating: 5,
            worstRating: 1,
          }
        : undefined,
  }
  return (
    <script
      type="application/ld+json"
      // Rendered as raw JSON on purpose — React escaping would break
      // the structured-data schema. `client_description` is user-authored
      // so we route through safeJsonLd() to neutralize </script>, --> and
      // U+2028 / U+2029 separators that JSON.stringify leaves intact.
      dangerouslySetInnerHTML={{ __html: safeJsonLd(payload) }}
    />
  )
}
