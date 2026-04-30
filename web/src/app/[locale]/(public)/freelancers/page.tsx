import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { SearchPage } from "@/features/provider/components/search-page"
import { fetchListingFirstPage } from "@/features/provider/api/search-server"
import { buildItemList } from "@/features/provider/api/listing-jsonld"
import { safeJsonLd } from "@/shared/lib/json-ld"

// /freelancers lists every organization of type `provider_personal`.
// PERF-W-02: this page is an async Server Component that pre-fetches
// the first 20 results, exposes structured data (ItemList) to
// Googlebot, and hands the seed to the client SearchPage so the
// cards paint without a client round-trip.

type Props = {
  params: Promise<{ locale: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "publicListing" })
  const firstPage = await fetchListingFirstPage("freelancer")
  const count = firstPage?.found ?? 0

  const title = t("freelancers.title", { count })
  const description = t("freelancers.description", { count })

  return {
    title,
    description,
    alternates: { canonical: "/freelancers" },
    openGraph: {
      type: "website",
      title,
      description,
    },
    twitter: {
      card: "summary",
      title,
      description,
    },
  }
}

export default async function FreelancersDirectoryPage() {
  const firstPage = await fetchListingFirstPage("freelancer")
  const itemList = firstPage
    ? buildItemList({
        type: "freelancer",
        documents: firstPage.documents,
        totalFound: firstPage.found,
      })
    : null

  return (
    <>
      {itemList ? (
        <script
          type="application/ld+json"
          dangerouslySetInnerHTML={{ __html: safeJsonLd(itemList) }}
        />
      ) : null}
      <SearchPage type="freelancer" initialFirstPage={firstPage ?? undefined} />
    </>
  )
}
