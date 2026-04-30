/**
 * listing-jsonld.ts builds the JSON-LD `ItemList` payload for the
 * three public listing pages. The schema points each item at the
 * canonical detail page so Google can crawl the full directory.
 *
 * Reference: https://schema.org/ItemList
 *
 * `safeJsonLd` from `shared/lib/json-ld.ts` is applied at the
 * render site to neutralise stored-XSS sequences before injecting
 * via `dangerouslySetInnerHTML`.
 */

import type { RawSearchDocument } from "@/shared/lib/search/typesense-client"
import type { SearchType } from "@/shared/lib/search/search-api"

const TYPE_TO_PATH: Record<SearchType, string> = {
  freelancer: "/freelancers",
  agency: "/agencies",
  referrer: "/referrers",
  enterprise: "/clients",
}

const TYPE_TO_ITEM_TYPE: Record<SearchType, string> = {
  freelancer: "Person",
  agency: "Organization",
  referrer: "Person",
  enterprise: "Organization",
}

export interface ListingItemListInput {
  type: SearchType
  documents: RawSearchDocument[]
  totalFound: number
}

/**
 * buildItemList returns a JSON-LD-compatible plain object. Pass the
 * result through `safeJsonLd()` before embedding in the DOM.
 */
export function buildItemList(input: ListingItemListInput): Record<string, unknown> {
  const path = TYPE_TO_PATH[input.type]
  const itemType = TYPE_TO_ITEM_TYPE[input.type]
  return {
    "@context": "https://schema.org",
    "@type": "ItemList",
    name: listingName(input.type),
    numberOfItems: input.totalFound,
    itemListElement: input.documents.map((doc, index) => ({
      "@type": "ListItem",
      position: index + 1,
      item: {
        "@type": itemType,
        "@id": `${path}/${doc.organization_id ?? doc.id.split(":")[0] ?? doc.id}`,
        name: doc.display_name,
        description: doc.title || undefined,
        image: doc.photo_url || undefined,
        ...(doc.city
          ? {
              address: {
                "@type": "PostalAddress",
                addressLocality: doc.city,
                addressCountry: doc.country_code || undefined,
              },
            }
          : {}),
      },
    })),
  }
}

function listingName(type: SearchType): string {
  switch (type) {
    case "freelancer":
      return "Freelancers"
    case "agency":
      return "Agencies"
    case "referrer":
      return "Business referrers"
    case "enterprise":
    default:
      return "Listings"
  }
}
