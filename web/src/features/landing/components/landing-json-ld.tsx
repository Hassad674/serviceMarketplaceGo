import { safeJsonLd } from "@/shared/lib/json-ld"
import { absoluteUrl } from "@/shared/lib/seo/alternates"

// LandingJsonLd renders the structured-data payload Googlebot uses
// to surface the homepage as an Organization + WebSite. We attach a
// `potentialAction` of type `SearchAction` so the rich card surfaces
// the in-site search box. The endpoint targets the freelance
// listing because that's the default tab on the landing search bar
// — agencies/referrers users still get there with one click.
//
// The payload is XSS-hardened via `safeJsonLd` (escapes `</`, U+2028,
// U+2029) before being injected through `dangerouslySetInnerHTML`.

const ORGANIZATION_DESCRIPTION =
  "Marketplace B2B sans commission : 0 % côté client, frais fixes plafonnés côté freelance, agences et apporteurs notés sur des missions réelles."

function buildOrganization(homeUrl: string) {
  return {
    "@context": "https://schema.org",
    "@type": "Organization",
    "@id": `${homeUrl}#organization`,
    name: "DesignedTrust Services",
    url: homeUrl,
    description: ORGANIZATION_DESCRIPTION,
    foundingDate: "2026",
    address: {
      "@type": "PostalAddress",
      addressLocality: "Paris",
      addressCountry: "FR",
    },
  }
}

function buildWebsite(homeUrl: string, searchUrl: string, organizationId: string) {
  return {
    "@context": "https://schema.org",
    "@type": "WebSite",
    "@id": `${homeUrl}#website`,
    url: homeUrl,
    name: "DesignedTrust Services",
    publisher: { "@id": organizationId },
    potentialAction: {
      "@type": "SearchAction",
      target: {
        "@type": "EntryPoint",
        urlTemplate: searchUrl,
      },
      // `query-input` MUST stay literal "required name=search_term_string"
      // per Google's Sitelinks Search Box spec; do not localise.
      "query-input": "required name=search_term_string",
    },
  }
}

export function LandingJsonLd() {
  const homeUrl = absoluteUrl("/")
  const searchUrl = absoluteUrl("/freelancers?q={search_term_string}")
  const organization = buildOrganization(homeUrl)
  const website = buildWebsite(homeUrl, searchUrl, organization["@id"])

  return (
    <>
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: safeJsonLd(organization) }}
      />
      <script
        type="application/ld+json"
        dangerouslySetInnerHTML={{ __html: safeJsonLd(website) }}
      />
    </>
  )
}
