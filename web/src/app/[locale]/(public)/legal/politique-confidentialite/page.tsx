import type { Metadata } from "next"
import { getTranslations } from "next-intl/server"
import { LegalDocument } from "@/shared/components/legal/legal-document"
import type { LegalSection } from "@/shared/components/legal/legal-document"

// /legal/politique-confidentialite — D4 (GDPR Phase C). Public long
// version of the privacy policy. Source Markdown canonique :
// /legal/politique-confidentialite.md. The short /privacy page remains
// the entry point linked from the legal footer.
export async function generateMetadata({
  params,
}: {
  params: Promise<{ locale: string }>
}): Promise<Metadata> {
  const { locale } = await params
  const t = await getTranslations({
    locale,
    namespace: "legal.docs.politiqueConfidentialite",
  })
  return {
    title: `${t("title")} | Marketplace Service`,
    description: t("subtitle"),
    robots: { index: false, follow: false },
  }
}

export default async function LegalPolitiquePage({
  params,
}: {
  params: Promise<{ locale: string }>
}) {
  const { locale } = await params
  const t = await getTranslations({
    locale,
    namespace: "legal.docs.politiqueConfidentialite",
  })
  const tDocs = await getTranslations({ locale, namespace: "legal.docs" })

  const summary = t("summaryItems").split("|")
  const rights = t("rightsItems").split("|")

  const sections: LegalSection[] = [
    {
      id: "summary",
      heading: t("summaryHeading"),
      blocks: [{ type: "ul", items: summary }],
    },
    {
      id: "rights",
      heading: t("rightsHeading"),
      blocks: [{ type: "ul", items: rights }],
    },
    {
      id: "retention",
      heading: t("retentionHeading"),
      blocks: [{ type: "p", content: t("retentionBody") }],
    },
    {
      id: "transfer",
      heading: t("transferHeading"),
      blocks: [{ type: "p", content: t("transferBody") }],
    },
    {
      id: "contact",
      heading: t("contactHeading"),
      blocks: [{ type: "p", content: t("contactBody") }],
    },
  ]

  return (
    <LegalDocument
      title={t("title")}
      subtitle={t("subtitle")}
      lastUpdatedISO={tDocs("lastUpdatedISO")}
      sourceHref="https://github.com/Hassad674/serviceMarketplaceGo/blob/main/legal/politique-confidentialite.md"
      englishNotice={tDocs("englishNotice")}
      sections={sections}
    />
  )
}
