import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { SearchPage } from "@/features/provider/components/search-page"
import { fetchListingFirstPage } from "@/features/provider/api/search-server"
import { buildItemList } from "@/features/provider/api/listing-jsonld"
import { safeJsonLd } from "@/shared/lib/json-ld"

// /referrers lists every organization of type `provider_personal`
// with `referrer_enabled = true`. PERF-W-02: async Server Component
// that pre-fetches the first 20 results, exposes JSON-LD ItemList
// for Googlebot, and seeds the client SearchPage so cards paint
// without a client refetch.

type Props = {
  params: Promise<{ locale: string }>
}

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({ locale, namespace: "publicListing" })
  const firstPage = await fetchListingFirstPage("referrer")
  const count = firstPage?.found ?? 0

  const title = t("referrers.title", { count })
  const description = t("referrers.description", { count })

  return {
    title,
    description,
    alternates: { canonical: "/referrers" },
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

export default async function ReferrersDirectoryPage() {
  const firstPage = await fetchListingFirstPage("referrer")
  const itemList = firstPage
    ? buildItemList({
        type: "referrer",
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
      <SearchPage type="referrer" initialFirstPage={firstPage ?? undefined} />
    </>
  )
}
